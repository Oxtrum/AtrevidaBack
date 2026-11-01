package pgsql

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"atrevida-agenda-api/cuenta"
	"atrevida-agenda-api/models"
)

// Consultas sobre pago_aplicaciones (en pantalla: cobros, "este pago cubre X Bs de este cargo"). Ninguna fila se
// borra ni se edita: anular marca la fila y las sumas ignoran las anuladas.
const (
	sqlListarCargos = `
		SELECT c.id, c.cliente_id, c.local_id, c.tipo, c.concepto, c.monto,
		       COALESCE((SELECT SUM(a.monto) FROM pago_aplicaciones a WHERE a.cargo_id = c.id AND a.anulado = FALSE), 0) AS pagado,
		       c.fecha_hecho, c.registrado_en, c.reserva_id, c.plan_id, c.anulado, c.motivo_anulacion
		FROM cargos_cliente c
		WHERE c.cliente_id = $1 AND ($2 OR c.anulado = FALSE)
		ORDER BY c.fecha_hecho, c.id`

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

// ErrReglaCuenta es un error de regla de negocio de la cuenta (cliente o cargo inexistente, aplicacion que excede
// lo pendiente...) con un mensaje apto para el usuario. Los demas errores del repositorio son de infraestructura.
type ErrReglaCuenta struct{ Msg string }

func (e ErrReglaCuenta) Error() string { return e.Msg }

func errCuenta(format string, args ...interface{}) error {
	return ErrReglaCuenta{Msg: fmt.Sprintf(format, args...)}
}

// CrearCargo registra un cargo para un cliente existente.
func (r *CuentaRepo) CrearCargo(in CargoInput) (int, error) {
	var existe bool
	if err := r.db.Get(&existe, `SELECT EXISTS (SELECT 1 FROM clientes WHERE id = $1)`, in.ClienteID); err != nil {
		return 0, fmt.Errorf("no se pudo verificar el cliente: %w", err)
	}
	if !existe {
		return 0, errCuenta("cliente no encontrado")
	}
	var id int
	err := r.db.Get(&id, `
		INSERT INTO cargos_cliente (
			cliente_id, local_id, tipo, concepto, monto, fecha_hecho, reserva_id, plan_id, creado_por_id, creado_por
		)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6::timestamptz, NOW()), $7, $8, $9, $10)
		RETURNING id`,
		in.ClienteID, in.LocalID, in.Tipo, in.Concepto, in.Monto, in.FechaHecho, in.ReservaID, in.PlanID, in.CreadoPorID, in.CreadoPor)
	if err != nil {
		return 0, fmt.Errorf("no se pudo crear el cargo: %w", err)
	}
	return id, nil
}

// AnularCargo anula un cargo con motivo y libera lo que los abonos le habian aplicado: ese dinero vuelve a ser
// saldo a favor sin aplicar.
func (r *CuentaRepo) AnularCargo(id int, motivo string) error {
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
	return cargos, nil
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
// Lo que no se aplica queda como saldo a favor. Devuelve el codigo del pago.
func (r *CuentaRepo) CrearAbono(in AbonoInput) (string, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var cliente struct {
		Nombre string `db:"nombre"`
		NIT    string `db:"nit"`
	}
	if err := tx.Get(&cliente, `SELECT nombre || ' ' || apellido AS nombre, COALESCE(nit, '') AS nit FROM clientes WHERE id = $1`, in.ClienteID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errCuenta("cliente no encontrado")
		}
		return "", fmt.Errorf("no se pudo leer el cliente: %w", err)
	}
	pendientes, err := pendientesBloqueados(tx, in.ClienteID)
	if err != nil {
		return "", err
	}
	if err := cuenta.ValidarAplicaciones(in.Monto, 0, in.Aplicaciones, pendientes); err != nil {
		return "", ErrReglaCuenta{Msg: err.Error()}
	}

	var pagoID int
	var codigo string
	err = tx.QueryRowx(`
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
		return "", pagoInsertError(err)
	}
	if err := insertarAplicaciones(tx, pagoID, in.Aplicaciones); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("no se pudo registrar el abono")
	}
	return codigo, nil
}

// AplicarAbono reparte entre cargos lo que un abono ya existente tiene sin aplicar (p. ej. saldo a favor que
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
