package pgsql

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"atrevida-agenda-api/cuenta"
	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

// Consultas sobre pago_aplicaciones (en pantalla: cobros, "este pago cubre X Bs de este cargo"). Ninguna fila se
// borra ni se edita: anular marca la fila y las sumas ignoran las anuladas.
const (
	sqlListarCargos = `
		SELECT c.id, c.cliente_id, c.local_id, c.tipo, c.concepto, c.monto, c.item_id, c.cantidad,
		       COALESCE((SELECT SUM(a.monto) FROM pago_aplicaciones a WHERE a.cargo_id = c.id AND a.anulado = FALSE), 0) AS pagado,
		       c.fecha_hecho, c.registrado_en, c.reserva_id, c.plan_id, c.anulado, c.motivo_anulacion,
		       rv.fecha AS reserva_fecha,
		       COALESCE(rv.servicio_confirmado, rv.servicio_nombre, rv.servicio_solicitado) AS reserva_servicio,
		       pl.combo_nombre_texto AS plan_nombre
		FROM cargos_cliente c
		LEFT JOIN reservas rv ON rv.id = c.reserva_id
		LEFT JOIN planes pl ON pl.id = c.plan_id
		WHERE c.cliente_id = $1 AND ($2 OR c.anulado = FALSE)
		ORDER BY c.fecha_hecho, c.id`

	// Pagos que cubren cada cargo, agrupados por pago (un pago puede aplicarse en varias filas).
	sqlPagosDeCargos = `
		SELECT a.cargo_id, p.codigo_pago, p.tipo_pago, SUM(a.monto) AS monto
		FROM pago_aplicaciones a
		JOIN pagos p ON p.id = a.pago_id
		WHERE a.cargo_id = ANY($1) AND a.anulado = FALSE
		GROUP BY a.cargo_id, p.id, p.codigo_pago, p.tipo_pago
		ORDER BY a.cargo_id, MIN(a.id)`

	sqlListarAbonos = `
		SELECT p.codigo_pago, p.total_final AS monto, p.tipo_pago, p.fecha_creacion AS fecha_hecho, p.registrado_en,
		       COALESCE((SELECT SUM(a.monto) FROM pago_aplicaciones a WHERE a.pago_id = p.id AND a.anulado = FALSE), 0) AS aplicado
		FROM pagos p
		WHERE p.cliente_id = $1 AND p.es_abono = TRUE AND p.activo = TRUE AND p.estado = 'PAGADO'
		ORDER BY p.fecha_creacion DESC, p.id DESC`

	sqlPendientesPorCargo = `
		SELECT c.id,
		       c.monto - COALESCE((SELECT SUM(a.monto) FROM pago_aplicaciones a WHERE a.cargo_id = c.id AND a.anulado = FALSE), 0) AS pendiente
		FROM cargos_cliente c
		WHERE c.cliente_id = $1 AND c.anulado = FALSE`

	// Abonos con su dinero sin usar, del mas antiguo al mas nuevo. Se bloquean para que dos cobros simultaneos no
	// gasten el mismo saldo.
	sqlAbonosConSaldo = `
		SELECT p.id,
		       p.total_final - COALESCE((SELECT SUM(a.monto) FROM pago_aplicaciones a WHERE a.pago_id = p.id AND a.anulado = FALSE), 0) AS disponible
		FROM pagos p
		WHERE p.cliente_id = $1 AND p.es_abono = TRUE AND p.activo = TRUE AND p.estado = 'PAGADO'
		ORDER BY p.fecha_creacion, p.id
		FOR UPDATE OF p`

	sqlCubiertoDeUnPago = `SELECT COALESCE(SUM(monto), 0) FROM pago_aplicaciones WHERE pago_id = $1 AND anulado = FALSE`

	// Cada cobro es un registro nuevo; nunca se suma sobre uno existente.
	sqlInsertarCobro = `
		INSERT INTO pago_aplicaciones (pago_id, cargo_id, monto)
		VALUES ($1, $2, $3)`

	sqlAnularCobrosDelCargo = `
		UPDATE pago_aplicaciones
		SET anulado = TRUE, motivo_anulacion = $2, anulado_en = NOW()
		WHERE cargo_id = $1 AND anulado = FALSE`

	sqlAnularCobrosDelPago = `
		UPDATE pago_aplicaciones
		SET anulado = TRUE, motivo_anulacion = $2, anulado_en = NOW()
		WHERE pago_id = $1 AND anulado = FALSE`
)

// CuentaRepo persiste la cuenta del cliente: cargos, abonos y aplicaciones. Las reglas de integridad
// (un abono no aplica mas que su monto, un cargo no recibe mas que el suyo) viven en el paquete cuenta y se
// comprueban dentro de la transaccion, con los cargos del cliente bloqueados, para que dos cobros simultaneos
// no sobreapliquen.
type CuentaRepo struct {
	db *sqlx.DB
}

func NewCuentaRepo(db *sqlx.DB) *CuentaRepo { return &CuentaRepo{db: db} }

// CargoInput crea un cargo.
type CargoInput struct {
	ClienteID int
	LocalID   *int
	Tipo      string
	Concepto  string
	Monto     float64
	// FechaHecho es cuando ocurrio lo que se cobra; nil usa el momento actual.
	FechaHecho  *time.Time
	ReservaID   *int
	PlanID      *int
	CreadoPorID *int
	CreadoPor   string
	// ItemID y Cantidad: insumo del inventario que se cobra (solo cargos de tipo INSUMO). Al crear el cargo baja su
	// stock en LocalID, en la misma transaccion.
	ItemID   *int
	Cantidad *float64
	// PermitirStockNegativo deja cobrar un insumo sin stock suficiente (inventario.permitir_stock_negativo).
	PermitirStockNegativo bool
}

// AbonoInput crea un abono (pago a cuenta) y lo reparte entre cargos.
type AbonoInput struct {
	ClienteID   int
	LocalID     int
	LocalNombre string
	Monto       float64
	TipoPago    string
	// FechaHecho es cuando se entrego el dinero; nil usa el momento actual.
	FechaHecho   *time.Time
	Aplicaciones []cuenta.Reparto
	CajeroID     *int
	CajeroNombre string
	CajeroUser   *string
}

// CobroInput es un cobro de caja: el cajero marco cargos que se pagan completos.
type CobroInput struct {
	ClienteID   int
	LocalID     int
	LocalNombre string
	CargoIDs    []int
	// UsarSaldo descuenta primero el saldo disponible del cliente; false si prefiere guardarlo.
	UsarSaldo bool
	// Recarga es dinero extra que queda como saldo disponible.
	Recarga float64
	// TipoPago es el metodo del dinero nuevo; vacio solo si el saldo cubre todo y no hay recarga.
	TipoPago string
	// FechaHecho es cuando ocurrio el cobro; nil usa el momento actual.
	FechaHecho   *time.Time
	CajeroID     *int
	CajeroNombre string
	CajeroUser   *string
}

// CobroResultado es lo que se hizo: el pago nuevo (vacio si el saldo cubrio todo) y el resumen del cobro.
type CobroResultado struct {
	// CodigoPago del dinero nuevo que entro; vacio si no entro dinero.
	CodigoPago string           `json:"codigo_pago,omitempty" example:"PAGO-000123"`
	Plan       cuenta.PlanCobro `json:"plan"`
}

// ErrReglaCuenta es un error de regla de negocio de la cuenta (cliente o cargo inexistente, aplicacion que excede
// lo pendiente...) con un mensaje apto para el usuario. Los demas errores del repositorio son de infraestructura.
type ErrReglaCuenta struct{ Msg string }

func (e ErrReglaCuenta) Error() string { return e.Msg }

func errCuenta(format string, args ...interface{}) error {
	return ErrReglaCuenta{Msg: fmt.Sprintf(format, args...)}
}

// CrearCargo registra un cargo para un cliente existente.
func (r *CuentaRepo) CrearCargo(in CargoInput) (int, error) {
	ids, err := r.CrearCargos([]CargoInput{in})
	if err != nil {
		return 0, err
	}
	return ids[0], nil
}

// CrearCargos registra varios cargos en una sola transaccion: si uno falla (cliente inexistente, insumo sin stock),
// no queda ninguno ni se descuenta stock.
func (r *CuentaRepo) CrearCargos(cargos []CargoInput) ([]int, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	ids := make([]int, 0, len(cargos))
	for _, in := range cargos {
		id, err := crearCargoTx(tx, in)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("no se pudo crear el cargo: %w", err)
	}
	return ids, nil
}

func crearCargoTx(tx *sqlx.Tx, in CargoInput) (int, error) {
	var existe bool
	if err := tx.Get(&existe, `SELECT EXISTS (SELECT 1 FROM clientes WHERE id = $1)`, in.ClienteID); err != nil {
		return 0, fmt.Errorf("no se pudo verificar el cliente: %w", err)
	}
	if !existe {
		return 0, errCuenta("cliente no encontrado")
	}
	var id int
	err := tx.Get(&id, `
		INSERT INTO cargos_cliente (
			cliente_id, local_id, tipo, concepto, monto, fecha_hecho, reserva_id, plan_id, creado_por_id, creado_por,
			item_id, cantidad
		)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6::timestamptz, NOW()), $7, $8, $9, $10, $11, $12)
		RETURNING id`,
		in.ClienteID, in.LocalID, in.Tipo, in.Concepto, in.Monto, in.FechaHecho, in.ReservaID, in.PlanID, in.CreadoPorID, in.CreadoPor,
		in.ItemID, in.Cantidad)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" && in.ItemID != nil {
			return 0, repository.ErrItemNoEncontrado
		}
		return 0, fmt.Errorf("no se pudo crear el cargo: %w", err)
	}
	// El cargo y el descuento de stock van juntos: si el insumo no alcanza, no queda el cargo.
	if in.ItemID != nil {
		if err := descontarInsumoDeCargo(tx, id, in); err != nil {
			return 0, err
		}
	}
	return id, nil
}

// AnularCargo anula un cargo con motivo y libera lo que los abonos le habian aplicado: ese dinero vuelve a ser
// saldo disponible.
func (r *CuentaRepo) AnularCargo(id int, motivo string, actor repository.ConsumoActor) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE cargos_cliente
		SET anulado = TRUE, motivo_anulacion = $2, anulado_en = NOW()
		WHERE id = $1 AND anulado = FALSE`, id, motivo)
	if err != nil {
		return fmt.Errorf("no se pudo anular el cargo: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errCuenta("cargo no encontrado")
	}
	if _, err := tx.Exec(sqlAnularCobrosDelCargo, id, "Cargo anulado: "+motivo); err != nil {
		return fmt.Errorf("no se pudieron liberar los cobros del cargo: %w", err)
	}
	// Si el cargo cobraba un insumo, vuelve al stock.
	if err := revertirInsumoDeCargo(tx, id, actor); err != nil {
		return err
	}
	return tx.Commit()
}

// ListarCargos devuelve los cargos de un cliente, del mas antiguo al mas nuevo, con lo ya aplicado de cada uno.
// Estado y pendiente los calcula el servicio.
func (r *CuentaRepo) ListarCargos(clienteID int, incluirAnulados bool) ([]models.CargoCliente, error) {
	cargos := []models.CargoCliente{}
	err := r.db.Select(&cargos, sqlListarCargos, clienteID, incluirAnulados)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron listar los cargos: %w", err)
	}
	if err := r.adjuntarPagos(cargos); err != nil {
		return nil, err
	}
	return cargos, nil
}

// adjuntarPagos agrega a cada cargo los pagos que lo cubren, en una sola consulta para todos.
func (r *CuentaRepo) adjuntarPagos(cargos []models.CargoCliente) error {
	ids := make([]int64, len(cargos))
	for i := range cargos {
		ids[i] = int64(cargos[i].ID)
		cargos[i].Pagos = []models.CargoPago{}
	}
	if len(ids) == 0 {
		return nil
	}
	var filas []models.CargoPago
	if err := r.db.Select(&filas, sqlPagosDeCargos, pq.Array(ids)); err != nil {
		return fmt.Errorf("no se pudieron listar los pagos de los cargos: %w", err)
	}
	porCargo := make(map[int][]models.CargoPago, len(cargos))
	for _, f := range filas {
		porCargo[f.CargoID] = append(porCargo[f.CargoID], f)
	}
	for i := range cargos {
		if pagos := porCargo[cargos[i].ID]; pagos != nil {
			cargos[i].Pagos = pagos
		}
	}
	return nil
}

// ListarAbonos devuelve los abonos activos de un cliente, del mas reciente al mas antiguo.
func (r *CuentaRepo) ListarAbonos(clienteID int) ([]models.AbonoCliente, error) {
	abonos := []models.AbonoCliente{}
	err := r.db.Select(&abonos, sqlListarAbonos, clienteID)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron listar los abonos: %w", err)
	}
	return abonos, nil
}

// pendientesBloqueados bloquea los cargos vigentes del cliente y devuelve lo que le falta cubrir a cada uno.
// Solo incluye los que tienen algo pendiente.
func pendientesBloqueados(tx *sqlx.Tx, clienteID int) (map[int]float64, error) {
	if _, err := tx.Exec(`SELECT 1 FROM cargos_cliente WHERE cliente_id = $1 AND anulado = FALSE FOR UPDATE`, clienteID); err != nil {
		return nil, fmt.Errorf("no se pudieron bloquear los cargos: %w", err)
	}
	var filas []struct {
		ID        int     `db:"id"`
		Pendiente float64 `db:"pendiente"`
	}
	err := tx.Select(&filas, sqlPendientesPorCargo, clienteID)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron calcular los pendientes: %w", err)
	}
	pendientes := make(map[int]float64, len(filas))
	for _, f := range filas {
		if cuenta.PendienteDe(f.Pendiente, 0) > 0 {
			pendientes[f.ID] = f.Pendiente
		}
	}
	return pendientes, nil
}

func insertarAplicaciones(tx *sqlx.Tx, pagoID int, aplicaciones []cuenta.Reparto) error {
	for _, a := range aplicaciones {
		if _, err := tx.Exec(sqlInsertarCobro, pagoID, a.CargoID, a.Monto); err != nil {
			return fmt.Errorf("no se pudo aplicar el abono al cargo %d: %w", a.CargoID, err)
		}
	}
	return nil
}

// CrearAbono registra un pago a cuenta del cliente y lo reparte entre sus cargos en una sola transaccion.
// Lo que no se aplica queda como saldo disponible. Devuelve el codigo del pago.
func (r *CuentaRepo) CrearAbono(in AbonoInput) (string, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	cliente, err := clienteParaPago(tx, in.ClienteID)
	if err != nil {
		return "", err
	}
	pendientes, err := pendientesBloqueados(tx, in.ClienteID)
	if err != nil {
		return "", err
	}
	if err := cuenta.ValidarAplicaciones(in.Monto, 0, in.Aplicaciones, pendientes); err != nil {
		return "", ErrReglaCuenta{Msg: err.Error()}
	}

	pagoID, codigo, err := insertarPagoAbono(tx, in, cliente)
	if err != nil {
		return "", err
	}
	if err := insertarAplicaciones(tx, pagoID, in.Aplicaciones); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("no se pudo registrar el abono")
	}
	return codigo, nil
}

type clientePago struct {
	Nombre string `db:"nombre"`
	NIT    string `db:"nit"`
}

// clienteParaPago lee los datos del cliente que se copian al pago.
func clienteParaPago(tx *sqlx.Tx, clienteID int) (clientePago, error) {
	var cliente clientePago
	if err := tx.Get(&cliente, `SELECT nombre || ' ' || apellido AS nombre, COALESCE(nit, '') AS nit FROM clientes WHERE id = $1`, clienteID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return clientePago{}, errCuenta("cliente no encontrado")
		}
		return clientePago{}, fmt.Errorf("no se pudo leer el cliente: %w", err)
	}
	return cliente, nil
}

// insertarPagoAbono crea el pago de un abono (sin detalle) y devuelve su id y codigo.
func insertarPagoAbono(tx *sqlx.Tx, in AbonoInput, cliente clientePago) (int, string, error) {
	var pagoID int
	var codigo string
	err := tx.QueryRowx(`
		INSERT INTO pagos (
			local_id, local_nombre, cliente_id, cliente_nit, cliente_nombre,
			subtotal, descuento, total_final, tipo_pago, estado, activo,
			id_cajero, nombre_cajero, username_cajero, es_abono, fecha_creacion
		)
		VALUES ($1, $2, $3, $4, $5, $6, 0, $6, $7, 'PAGADO', TRUE, $8, $9, $10, TRUE, COALESCE($11::timestamptz, NOW()))
		RETURNING id, codigo_pago`,
		in.LocalID, in.LocalNombre, in.ClienteID, cliente.NIT, cliente.Nombre,
		in.Monto, in.TipoPago, in.CajeroID, in.CajeroNombre, in.CajeroUser, in.FechaHecho,
	).Scan(&pagoID, &codigo)
	if err != nil {
		return 0, "", pagoInsertError(err)
	}
	return pagoID, codigo, nil
}

// Cobrar registra un cobro de caja en una sola transaccion: paga completos los cargos marcados con el saldo
// disponible del cliente (si quiere) y con dinero nuevo, y guarda la recarga como saldo disponible. Si algo falla
// no queda nada a medias.
func (r *CuentaRepo) Cobrar(in CobroInput) (CobroResultado, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return CobroResultado{}, err
	}
	defer tx.Rollback()

	cliente, err := clienteParaPago(tx, in.ClienteID)
	if err != nil {
		return CobroResultado{}, err
	}
	pendientes, err := pendientesBloqueados(tx, in.ClienteID)
	if err != nil {
		return CobroResultado{}, err
	}
	cargos := make([]cuenta.CargoACobrar, 0, len(in.CargoIDs))
	var total float64
	for _, id := range in.CargoIDs {
		pendiente, ok := pendientes[id]
		if !ok {
			return CobroResultado{}, errCuenta("cargo %d no encontrado entre los pendientes del cliente", id)
		}
		cargos = append(cargos, cuenta.CargoACobrar{ID: id, Pendiente: pendiente})
		total += pendiente
	}

	var saldos []struct {
		ID         int     `db:"id"`
		Disponible float64 `db:"disponible"`
	}
	if err := tx.Select(&saldos, sqlAbonosConSaldo, in.ClienteID); err != nil {
		return CobroResultado{}, fmt.Errorf("no se pudo leer el saldo disponible: %w", err)
	}
	var fuentes []cuenta.FuenteDinero
	var disponible float64
	for _, s := range saldos {
		if cuenta.PendienteDe(s.Disponible, 0) > 0 {
			fuentes = append(fuentes, cuenta.FuenteDinero{PagoID: s.ID, Disponible: s.Disponible})
			disponible += s.Disponible
		}
	}
	if !in.UsarSaldo {
		fuentes = nil
	}

	plan := cuenta.PlanearCobro(total, disponible, in.UsarSaldo, in.Recarga)
	resultado := CobroResultado{Plan: plan}
	if plan.NuevoPago > 0 {
		if in.TipoPago == "" {
			return CobroResultado{}, errCuenta("tipo_pago es requerido para registrar el dinero que entra (%.2f)", plan.NuevoPago)
		}
		pagoID, codigo, err := insertarPagoAbono(tx, AbonoInput{
			ClienteID: in.ClienteID, LocalID: in.LocalID, LocalNombre: in.LocalNombre, Monto: plan.NuevoPago,
			TipoPago: in.TipoPago, FechaHecho: in.FechaHecho,
			CajeroID: in.CajeroID, CajeroNombre: in.CajeroNombre, CajeroUser: in.CajeroUser,
		}, cliente)
		if err != nil {
			return CobroResultado{}, err
		}
		fuentes = append(fuentes, cuenta.FuenteDinero{PagoID: pagoID, Disponible: plan.NuevoPago})
		resultado.CodigoPago = codigo
	}

	cobros, err := cuenta.CubrirCargos(cargos, fuentes)
	if err != nil {
		return CobroResultado{}, ErrReglaCuenta{Msg: err.Error()}
	}
	for _, c := range cobros {
		if _, err := tx.Exec(sqlInsertarCobro, c.PagoID, c.CargoID, c.Monto); err != nil {
			return CobroResultado{}, fmt.Errorf("no se pudo registrar el cobro del cargo %d: %w", c.CargoID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return CobroResultado{}, fmt.Errorf("no se pudo registrar el cobro")
	}
	return resultado, nil
}

// AplicarAbono reparte entre cargos lo que un abono ya existente tiene sin aplicar (p. ej. saldo disponible que
// llego antes que el cargo).
func (r *CuentaRepo) AplicarAbono(codigoPago string, clienteID int, aplicaciones []cuenta.Reparto) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var abono struct {
		ID        int     `db:"id"`
		Monto     float64 `db:"monto"`
		ClienteID int     `db:"cliente_id"`
	}
	err = tx.Get(&abono, `
		SELECT id, total_final AS monto, cliente_id FROM pagos
		WHERE codigo_pago = $1 AND es_abono = TRUE AND activo = TRUE AND estado = 'PAGADO'
		FOR UPDATE`, codigoPago)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && abono.ClienteID != clienteID) {
		return errCuenta("abono no encontrado")
	}
	if err != nil {
		return fmt.Errorf("no se pudo leer el abono: %w", err)
	}
	var aplicado float64
	if err := tx.Get(&aplicado, sqlCubiertoDeUnPago, abono.ID); err != nil {
		return fmt.Errorf("no se pudo leer lo aplicado: %w", err)
	}
	pendientes, err := pendientesBloqueados(tx, clienteID)
	if err != nil {
		return err
	}
	if err := cuenta.ValidarAplicaciones(abono.Monto, aplicado, aplicaciones, pendientes); err != nil {
		return ErrReglaCuenta{Msg: err.Error()}
	}
	if err := insertarAplicaciones(tx, abono.ID, aplicaciones); err != nil {
		return err
	}
	return tx.Commit()
}

// TotalesCuenta devuelve el total de abonos (activos y pagados) y de cargos (no anulados) de un cliente.
func (r *CuentaRepo) TotalesCuenta(clienteID int) (abonos, cargos float64, err error) {
	if err = r.db.Get(&abonos, `
		SELECT COALESCE(SUM(total_final), 0) FROM pagos
		WHERE cliente_id = $1 AND es_abono = TRUE AND activo = TRUE AND estado = 'PAGADO'`, clienteID); err != nil {
		return 0, 0, fmt.Errorf("no se pudieron sumar los abonos: %w", err)
	}
	if err = r.db.Get(&cargos, `
		SELECT COALESCE(SUM(monto), 0) FROM cargos_cliente WHERE cliente_id = $1 AND anulado = FALSE`, clienteID); err != nil {
		return 0, 0, fmt.Errorf("no se pudieron sumar los cargos: %w", err)
	}
	return abonos, cargos, nil
}

// ClienteExiste indica si el cliente esta registrado.
func (r *CuentaRepo) ClienteExiste(clienteID int) (bool, error) {
	var existe bool
	if err := r.db.Get(&existe, `SELECT EXISTS (SELECT 1 FROM clientes WHERE id = $1)`, clienteID); err != nil {
		return false, fmt.Errorf("no se pudo verificar el cliente: %w", err)
	}
	return existe, nil
}
