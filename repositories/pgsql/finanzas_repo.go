package pgsql

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"

	"github.com/jmoiron/sqlx"
)

type FinanzasRepo struct {
	db *sqlx.DB
}

func NewFinanzasRepo(db *sqlx.DB) *FinanzasRepo {
	return &FinanzasRepo{db: db}
}

func redondear2(v float64) float64 { return math.Round(v*100) / 100 }

// ─── Categorias financieras ───────────────────────────────────────────────────

func (r *FinanzasRepo) ListCategorias() ([]models.CategoriaFinanciera, error) {
	cats := []models.CategoriaFinanciera{}
	if err := r.db.Select(&cats,
		`SELECT id, nombre, naturaleza, activo FROM categorias_financieras ORDER BY naturaleza DESC, nombre`); err != nil {
		return nil, fmt.Errorf("error al listar categorias financieras: %w", err)
	}
	return cats, nil
}

func (r *FinanzasRepo) CreateCategoria(nombre, naturaleza string) (int, error) {
	var id int
	if err := r.db.QueryRowx(
		`INSERT INTO categorias_financieras (nombre, naturaleza) VALUES ($1, $2) RETURNING id`,
		nombre, naturaleza).Scan(&id); err != nil {
		return 0, traducirDuplicado(err, "ya existe una categoria financiera con ese nombre")
	}
	return id, nil
}

func (r *FinanzasRepo) UpdateCategoria(id int, nombre *string, activo *bool, naturaleza *string) error {
	if naturaleza != nil && *naturaleza != "AMBOS" {
		// Solo se puede acotar a INGRESO/EGRESO si no hay movimientos de la otra naturaleza.
		var enConflicto int
		if err := r.db.Get(&enConflicto,
			`SELECT COUNT(*) FROM movimientos_financieros WHERE categoria_id = $1 AND naturaleza <> $2`,
			id, *naturaleza); err != nil {
			return fmt.Errorf("error al validar la naturaleza de la categoria: %w", err)
		}
		if enConflicto > 0 {
			return errors.New("la categoria ya tiene movimientos de otro tipo, solo puede cambiarse a AMBOS")
		}
	}
	res, err := r.db.Exec(`
		UPDATE categorias_financieras SET nombre = COALESCE($1, nombre), activo = COALESCE($2, activo),
			naturaleza = COALESCE($3, naturaleza)
		WHERE id = $4`, nombre, activo, naturaleza, id)
	if err != nil {
		return traducirDuplicado(err, "ya existe una categoria financiera con ese nombre")
	}
	n, err := affectedRows(res, "actualizar categoria financiera")
	if err != nil {
		return err
	}
	if n == 0 {
		return repository.ErrCategoriaNoEnc
	}
	return nil
}

// GetCategoria devuelve la categoria o ErrCategoriaNoEnc.
func (r *FinanzasRepo) GetCategoria(id int) (*models.CategoriaFinanciera, error) {
	var c models.CategoriaFinanciera
	err := r.db.Get(&c, `SELECT id, nombre, naturaleza, activo FROM categorias_financieras WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrCategoriaNoEnc
	}
	return &c, err
}

// ─── Movimientos financieros ──────────────────────────────────────────────────

const movimientoSelect = `
	SELECT m.id, m.codigo, m.naturaleza, m.clase, m.categoria_id, c.nombre AS categoria_nombre,
	       m.gasto_fijo_id, m.periodo::text AS periodo, m.compra_id, m.planilla_id, m.local_id, m.local_nombre,
	       m.fecha::text AS fecha, m.monto, m.descripcion, m.proveedor, m.metodo_pago, m.estado,
	       m.motivo_anulacion, m.nombre_usuario
	FROM movimientos_financieros m
	JOIN categorias_financieras c ON c.id = m.categoria_id`

func (r *FinanzasRepo) ListMovimientos(f repository.FiltroMovimientosFinancieros) ([]models.MovimientoFinanciero, error) {
	conds := []string{"TRUE"}
	args := []interface{}{}
	add := func(cond string, v interface{}) {
		args = append(args, v)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	if f.Naturaleza != "" {
		add("m.naturaleza = $%d", f.Naturaleza)
	}
	if f.Clase != "" {
		add("m.clase = $%d", f.Clase)
	}
	if f.CategoriaID != nil {
		add("m.categoria_id = $%d", *f.CategoriaID)
	}
	if f.Estado != "" {
		add("m.estado = $%d", f.Estado)
	}
	if f.LocalID != nil {
		add("m.local_id = $%d", *f.LocalID)
	}
	if f.FechaDesde != nil {
		add("m.fecha >= $%d", *f.FechaDesde)
	}
	if f.FechaHasta != nil {
		add("m.fecha <= $%d", *f.FechaHasta)
	}
	movs := []models.MovimientoFinanciero{}
	if err := r.db.Select(&movs, movimientoSelect+" WHERE "+strings.Join(conds, " AND ")+
		" ORDER BY m.fecha DESC, m.id DESC LIMIT 1000", args...); err != nil {
		return nil, fmt.Errorf("error al listar movimientos financieros: %w", err)
	}
	return movs, nil
}

func (r *FinanzasRepo) GetMovimiento(id int) (*models.MovimientoFinanciero, error) {
	var m models.MovimientoFinanciero
	err := r.db.Get(&m, movimientoSelect+" WHERE m.id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrMovimientoFinNoEnc
	}
	if err != nil {
		return nil, fmt.Errorf("error al obtener movimiento financiero: %w", err)
	}
	return &m, nil
}

type MovimientoInput struct {
	Naturaleza    string
	Clase         *string
	CategoriaID   int
	LocalID       *int
	Fecha         time.Time
	Monto         float64
	Descripcion   *string
	Proveedor     *string
	MetodoPago    *string
	Estado        string
	UsuarioID     *int
	UsuarioNombre string
}

func (r *FinanzasRepo) CreateMovimiento(in MovimientoInput) (int, error) {
	var id int
	err := r.db.QueryRowx(`
		INSERT INTO movimientos_financieros
			(naturaleza, clase, categoria_id, local_id, local_nombre, fecha, monto, descripcion, proveedor,
			 metodo_pago, estado, id_usuario, nombre_usuario)
		VALUES ($1, $2, $3, $4, (SELECT nombre FROM locales WHERE id = $4), $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`,
		in.Naturaleza, in.Clase, in.CategoriaID, in.LocalID, in.Fecha, in.Monto, in.Descripcion, in.Proveedor,
		in.MetodoPago, in.Estado, in.UsuarioID, nullStr(in.UsuarioNombre)).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("error al registrar movimiento financiero: %w", err)
	}
	return id, nil
}

// MovimientoPatch contiene los campos editables de un movimiento no anulado.
type MovimientoPatch struct {
	CategoriaID    *int
	LocalID        *int
	LocalSet       bool
	Fecha          *time.Time
	Monto          *float64
	Descripcion    *string
	DescripcionSet bool
	Proveedor      *string
	ProveedorSet   bool
	MetodoPago     *string
	MetodoSet      bool
	Estado         *string
}

func (r *FinanzasRepo) UpdateMovimiento(id int, p MovimientoPatch) error {
	sets := []string{"fecha_modificacion = NOW()"}
	args := []interface{}{}
	add := func(col string, v interface{}) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	if p.CategoriaID != nil {
		add("categoria_id", *p.CategoriaID)
	}
	if p.LocalSet {
		add("local_id", p.LocalID)
		args = append(args, p.LocalID)
		sets = append(sets, fmt.Sprintf("local_nombre = (SELECT nombre FROM locales WHERE id = $%d)", len(args)))
	}
	if p.Fecha != nil {
		add("fecha", *p.Fecha)
	}
	if p.Monto != nil {
		add("monto", *p.Monto)
	}
	if p.DescripcionSet {
		add("descripcion", p.Descripcion)
	}
	if p.ProveedorSet {
		add("proveedor", p.Proveedor)
	}
	if p.MetodoSet {
		add("metodo_pago", p.MetodoPago)
	}
	if p.Estado != nil {
		add("estado", *p.Estado)
	}
	args = append(args, id)
	res, err := r.db.Exec(fmt.Sprintf(
		"UPDATE movimientos_financieros SET %s WHERE id = $%d AND estado <> 'ANULADO'",
		strings.Join(sets, ", "), len(args)), args...)
	if err != nil {
		return fmt.Errorf("error al actualizar movimiento financiero: %w", err)
	}
	n, err := affectedRows(res, "actualizar movimiento financiero")
	if err != nil {
		return err
	}
	if n == 0 {
		return repository.ErrMovimientoFinNoEnc
	}
	return nil
}

func (r *FinanzasRepo) CambiarEstadoMovimiento(id int, estado string, motivo *string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("error al cambiar estado del movimiento: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(`
		UPDATE movimientos_financieros
		SET estado = $1, motivo_anulacion = CASE WHEN $1 = 'ANULADO' THEN $2 ELSE motivo_anulacion END,
		    fecha_modificacion = NOW()
		WHERE id = $3 AND estado <> 'ANULADO'`, estado, motivo, id)
	if err != nil {
		return fmt.Errorf("error al cambiar estado del movimiento: %w", err)
	}
	n, err := affectedRows(res, "cambiar estado del movimiento")
	if err != nil {
		return err
	}
	if n == 0 {
		return repository.ErrMovimientoFinNoEnc
	}
	// Pagar el egreso de una planilla desde Finanzas tambien paga la planilla, para que no queden desalineados.
	if estado == "PAGADO" {
		if _, err := tx.Exec(`
			UPDATE planillas SET estado = 'PAGADO', fecha_modificacion = NOW()
			WHERE estado = 'PENDIENTE' AND id = (SELECT planilla_id FROM movimientos_financieros WHERE id = $1)`, id); err != nil {
			return fmt.Errorf("error al pagar la planilla del movimiento: %w", err)
		}
	}
	return tx.Commit()
}

// ─── Gastos fijos ─────────────────────────────────────────────────────────────

const gastoFijoSelect = `
	SELECT g.id, g.nombre, g.categoria_id, c.nombre AS categoria_nombre, g.monto, g.dia_cobro::int AS dia_cobro,
	       g.local_id, l.nombre AS local_nombre, g.metodo_pago, g.proveedor, g.fecha_inicio::text AS fecha_inicio,
	       g.fecha_fin::text AS fecha_fin, g.activo, g.notas, g.frecuencia,
	       (SELECT MAX(m.fecha)::text FROM movimientos_financieros m
	        WHERE m.gasto_fijo_id = g.id AND m.estado = 'PAGADO') AS pagado_hasta
	FROM gastos_fijos g
	JOIN categorias_financieras c ON c.id = g.categoria_id
	LEFT JOIN locales l ON l.id = g.local_id`

func (r *FinanzasRepo) ListGastosFijos(soloActivos bool) ([]models.GastoFijo, error) {
	cond := ""
	if soloActivos {
		cond = " WHERE g.activo = TRUE"
	}
	gastos := []models.GastoFijo{}
	if err := r.db.Select(&gastos, gastoFijoSelect+cond+" ORDER BY g.activo DESC, g.nombre"); err != nil {
		return nil, fmt.Errorf("error al listar gastos fijos: %w", err)
	}
	return gastos, nil
}

func (r *FinanzasRepo) GetGastoFijo(id int) (*models.GastoFijo, error) {
	var g models.GastoFijo
	err := r.db.Get(&g, gastoFijoSelect+" WHERE g.id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrGastoFijoNoEnc
	}
	return &g, err
}

type GastoFijoInput struct {
	Nombre      string
	CategoriaID int
	Monto       float64
	DiaCobro    int
	LocalID     *int
	MetodoPago  *string
	Proveedor   *string
	FechaInicio time.Time
	FechaFin    *time.Time
	Activo      bool
	Notas       *string
	Frecuencia  string
}

func (r *FinanzasRepo) CreateGastoFijo(in GastoFijoInput) (int, error) {
	var id int
	err := r.db.QueryRowx(`
		INSERT INTO gastos_fijos
			(nombre, categoria_id, monto, dia_cobro, local_id, metodo_pago, proveedor, fecha_inicio, fecha_fin, activo, notas, frecuencia)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`,
		in.Nombre, in.CategoriaID, in.Monto, in.DiaCobro, in.LocalID, in.MetodoPago, in.Proveedor,
		in.FechaInicio, in.FechaFin, in.Activo, in.Notas, in.Frecuencia).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("error al crear gasto fijo: %w", err)
	}
	return id, nil
}

// UpdateGastoFijo reemplaza todos los campos de la plantilla (el servicio arma el estado final).
// No toca egresos ya generados (RN-34).
func (r *FinanzasRepo) UpdateGastoFijo(id int, in GastoFijoInput) error {
	res, err := r.db.Exec(`
		UPDATE gastos_fijos SET nombre = $1, categoria_id = $2, monto = $3, dia_cobro = $4, local_id = $5,
			metodo_pago = $6, proveedor = $7, fecha_inicio = $8, fecha_fin = $9, activo = $10, notas = $11,
			frecuencia = $12, fecha_modificacion = NOW()
		WHERE id = $13`,
		in.Nombre, in.CategoriaID, in.Monto, in.DiaCobro, in.LocalID, in.MetodoPago, in.Proveedor,
		in.FechaInicio, in.FechaFin, in.Activo, in.Notas, in.Frecuencia, id)
	if err != nil {
		return fmt.Errorf("error al actualizar gasto fijo: %w", err)
	}
	n, err := affectedRows(res, "actualizar gasto fijo")
	if err != nil {
		return err
	}
	if n == 0 {
		return repository.ErrGastoFijoNoEnc
	}
	return nil
}

// DesactivarGastoFijo da de baja la plantilla y anula los egresos que generó y siguen PENDIENTES (nadie los
// pagó, así que dejan de cobrarse). Los ya PAGADOS se conservan como historial.
func (r *FinanzasRepo) DesactivarGastoFijo(id int) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("error al desactivar gasto fijo: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(
		`UPDATE gastos_fijos SET activo = FALSE, fecha_modificacion = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("error al desactivar gasto fijo: %w", err)
	}
	n, err := affectedRows(res, "desactivar gasto fijo")
	if err != nil {
		return err
	}
	if n == 0 {
		return repository.ErrGastoFijoNoEnc
	}
	if _, err := tx.Exec(`
		UPDATE movimientos_financieros
		SET estado = 'ANULADO', motivo_anulacion = 'Gasto fijo desactivado', fecha_modificacion = NOW()
		WHERE gasto_fijo_id = $1 AND estado = 'PENDIENTE'`, id); err != nil {
		return fmt.Errorf("error al anular los egresos pendientes del gasto fijo: %w", err)
	}
	return tx.Commit()
}

// PeriodosGenerados devuelve los periodos que ya tienen egreso (de cualquier estado, anulados incluidos) para
// no volver a generarlos ni ofrecerlos. La clave es la fecha YYYY-MM-DD del periodo.
func (r *FinanzasRepo) PeriodosGenerados(gastoFijoID int) (map[string]bool, error) {
	var periodos []string
	if err := r.db.Select(&periodos,
		`SELECT periodo::text FROM movimientos_financieros WHERE gasto_fijo_id = $1 AND periodo IS NOT NULL`,
		gastoFijoID); err != nil {
		return nil, fmt.Errorf("error al leer los periodos generados: %w", err)
	}
	out := make(map[string]bool, len(periodos))
	for _, p := range periodos {
		out[p] = true
	}
	return out, nil
}

// AtrasadosPorGasto cuenta, por gasto fijo, los egresos pendientes cuya fecha ya paso.
func (r *FinanzasRepo) AtrasadosPorGasto(hoy time.Time) (map[int]int, error) {
	rows := []struct {
		ID int `db:"gasto_fijo_id"`
		N  int `db:"n"`
	}{}
	if err := r.db.Select(&rows, `
		SELECT gasto_fijo_id, COUNT(*) AS n FROM movimientos_financieros
		WHERE gasto_fijo_id IS NOT NULL AND estado = 'PENDIENTE' AND fecha < $1
		GROUP BY gasto_fijo_id`, hoy); err != nil {
		return nil, fmt.Errorf("error al contar los egresos fijos atrasados: %w", err)
	}
	out := make(map[int]int, len(rows))
	for _, r := range rows {
		out[r.ID] = r.N
	}
	return out, nil
}

// PendientesRegistradosHasta cuenta los egresos pendientes de un gasto fijo con fecha hasta `hoy`.
func (r *FinanzasRepo) PendientesRegistradosHasta(gastoFijoID int, hoy time.Time) (int, error) {
	var n int
	if err := r.db.Get(&n, `
		SELECT COUNT(*) FROM movimientos_financieros
		WHERE gasto_fijo_id = $1 AND estado = 'PENDIENTE' AND fecha <= $2`, gastoFijoID, hoy); err != nil {
		return 0, fmt.Errorf("error al contar los pagos pendientes del gasto fijo: %w", err)
	}
	return n, nil
}

// CobroPagar identifica un cobro de un gasto fijo: su periodo (clave de unicidad) y la fecha en que se cobra.
type CobroPagar struct {
	GastoFijoID int
	Periodo     time.Time
	Fecha       time.Time
}

// PagarCobrosGastoFijo deja PAGADO el egreso de cada cobro en una sola transaccion: si ya existe pendiente lo
// paga; si no existe lo crea ya pagado. Pagar dos veces no duplica (indice unico gasto+periodo). Un cobro
// anulado no se revive: devuelve error.
func (r *FinanzasRepo) PagarCobrosGastoFijo(cobros []CobroPagar, usuarioID *int, usuarioNombre string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("error al pagar gastos fijos: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, c := range cobros {
		if _, err := tx.Exec(`
			INSERT INTO movimientos_financieros
				(naturaleza, clase, categoria_id, gasto_fijo_id, periodo, local_id, local_nombre, fecha, monto,
				 descripcion, proveedor, metodo_pago, estado, id_usuario, nombre_usuario)
			SELECT 'EGRESO', 'FIJO', g.categoria_id, g.id, $2, g.local_id, l.nombre, $3, g.monto,
			       g.nombre, g.proveedor, g.metodo_pago, 'PAGADO', $4, $5
			FROM gastos_fijos g LEFT JOIN locales l ON l.id = g.local_id
			WHERE g.id = $1
			ON CONFLICT (gasto_fijo_id, periodo) WHERE gasto_fijo_id IS NOT NULL
			DO UPDATE SET estado = 'PAGADO', fecha_modificacion = NOW()
			WHERE movimientos_financieros.estado = 'PENDIENTE'`,
			c.GastoFijoID, c.Periodo, c.Fecha, usuarioID, usuarioNombre); err != nil {
			return fmt.Errorf("error al pagar el gasto fijo: %w", err)
		}
		var estado string
		if err := tx.Get(&estado,
			`SELECT estado FROM movimientos_financieros WHERE gasto_fijo_id = $1 AND periodo = $2`,
			c.GastoFijoID, c.Periodo); err != nil {
			return fmt.Errorf("error al verificar el pago del gasto fijo: %w", err)
		}
		if estado == "ANULADO" {
			return fmt.Errorf("el cobro del %s ya fue anulado y no puede pagarse", c.Fecha.Format("2006-01-02"))
		}
	}
	return tx.Commit()
}

// GenerarMovimientoGastoFijo inserta el egreso FIJO del periodo copiando monto, categoria, local y
// metodo de la plantilla en este momento (snapshot). Devuelve false si ya existia (RN-32).
func (r *FinanzasRepo) GenerarMovimientoGastoFijo(
	gastoFijoID int, periodo, fechaCobro time.Time, estado string,
) (bool, error) {
	res, err := r.db.Exec(`
		INSERT INTO movimientos_financieros
			(naturaleza, clase, categoria_id, gasto_fijo_id, periodo, local_id, local_nombre, fecha, monto,
			 descripcion, proveedor, metodo_pago, estado, nombre_usuario)
		SELECT 'EGRESO', 'FIJO', g.categoria_id, g.id, $2, g.local_id, l.nombre, $3, g.monto,
		       g.nombre, g.proveedor, g.metodo_pago, $4, 'sistema'
		FROM gastos_fijos g LEFT JOIN locales l ON l.id = g.local_id
		WHERE g.id = $1
		ON CONFLICT (gasto_fijo_id, periodo) WHERE gasto_fijo_id IS NOT NULL DO NOTHING`,
		gastoFijoID, periodo, fechaCobro, estado)
	if err != nil {
		return false, fmt.Errorf("error al generar gasto fijo: %w", err)
	}
	n, err := affectedRows(res, "generar gasto fijo")
	return n > 0, err
}

// ─── Compras ──────────────────────────────────────────────────────────────────

type CompraLinea struct {
	ItemID        int
	Cantidad      float64
	CostoUnitario float64
}

type CompraInput struct {
	LocalID       int
	Proveedor     *string
	Fecha         time.Time
	Notas         *string
	MetodoPago    *string
	Lineas        []CompraLinea
	UsuarioID     *int
	UsuarioNombre string
}

// CrearCompra registra compra + detalle + entradas de stock + egreso VARIABLE + costo_referencia en una
// sola transaccion: si algo falla no queda nada a medias (RN-21).
func (r *FinanzasRepo) CrearCompra(in CompraInput) (int, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var localNombre string
	if err := tx.Get(&localNombre, `SELECT nombre FROM locales WHERE id = $1 AND activo = TRUE`, in.LocalID); err != nil {
		return 0, errors.New("local no encontrado o inactivo")
	}

	type lineaCalc struct {
		CompraLinea
		subtotal float64
	}
	lineas := make([]lineaCalc, 0, len(in.Lineas))
	total := 0.0
	for _, l := range in.Lineas {
		var activo bool
		if err := tx.Get(&activo, `SELECT activo FROM inventario_items WHERE id = $1`, l.ItemID); err != nil {
			return 0, fmt.Errorf("%w: id %d", repository.ErrItemNoEncontrado, l.ItemID)
		}
		if !activo {
			return 0, fmt.Errorf("el item %d esta inactivo y no puede comprarse", l.ItemID)
		}
		sub := redondear2(l.Cantidad * l.CostoUnitario)
		total += sub
		lineas = append(lineas, lineaCalc{CompraLinea: l, subtotal: sub})
	}
	total = redondear2(total)

	var compraID int
	var codigo string
	if err := tx.QueryRowx(`
		INSERT INTO compras_inventario (local_id, proveedor, fecha, total, notas, id_usuario, nombre_usuario)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id, codigo`,
		in.LocalID, in.Proveedor, in.Fecha, total, in.Notas, in.UsuarioID, nullStr(in.UsuarioNombre),
	).Scan(&compraID, &codigo); err != nil {
		return 0, fmt.Errorf("error al registrar compra: %w", err)
	}

	for _, l := range lineas {
		if _, err := tx.Exec(`
			INSERT INTO detalle_compras (compra_id, item_id, cantidad, costo_unitario, subtotal)
			VALUES ($1,$2,$3,$4,$5)`, compraID, l.ItemID, l.Cantidad, l.CostoUnitario, l.subtotal); err != nil {
			return 0, fmt.Errorf("error al registrar detalle de compra: %w", err)
		}
		costo := l.CostoUnitario
		cid := compraID
		if _, err := AplicarMovimientoStock(tx, repository.MovimientoStockInput{
			ItemID: l.ItemID, LocalID: in.LocalID, Tipo: repository.MovEntradaCompra, Cantidad: l.Cantidad,
			CostoUnitario: &costo, CompraID: &cid, UsuarioID: in.UsuarioID, UsuarioNombre: in.UsuarioNombre,
			Motivo: "Compra " + codigo,
		}); err != nil {
			return 0, err
		}
		if _, err := tx.Exec(
			`UPDATE inventario_items SET costo_referencia = $1, fecha_modificacion = NOW() WHERE id = $2`,
			l.CostoUnitario, l.ItemID); err != nil {
			return 0, fmt.Errorf("error al actualizar costo de referencia: %w", err)
		}
	}

	categoriaID, err := categoriaComprasTx(tx)
	if err != nil {
		return 0, err
	}
	desc := "Compra " + codigo
	if _, err := tx.Exec(`
		INSERT INTO movimientos_financieros
			(naturaleza, clase, categoria_id, compra_id, local_id, local_nombre, fecha, monto, descripcion,
			 proveedor, metodo_pago, estado, id_usuario, nombre_usuario)
		VALUES ('EGRESO','VARIABLE',$1,$2,$3,$4,$5,$6,$7,$8,$9,'PAGADO',$10,$11)`,
		categoriaID, compraID, in.LocalID, localNombre, in.Fecha, total, desc, in.Proveedor, in.MetodoPago,
		in.UsuarioID, nullStr(in.UsuarioNombre)); err != nil {
		return 0, fmt.Errorf("error al registrar egreso de la compra: %w", err)
	}
	return compraID, tx.Commit()
}

// categoriaComprasTx devuelve la categoria "Compras de inventario", creandola si no existe.
func categoriaComprasTx(tx *sqlx.Tx) (int, error) {
	var id int
	err := tx.Get(&id, `
		SELECT id FROM categorias_financieras
		WHERE LOWER(nombre) = 'compras de inventario' AND naturaleza = 'EGRESO' LIMIT 1`)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	if err := tx.QueryRowx(
		`INSERT INTO categorias_financieras (nombre, naturaleza) VALUES ('Compras de inventario', 'EGRESO') RETURNING id`,
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("error al crear categoria de compras: %w", err)
	}
	return id, nil
}

const compraSelect = `
	SELECT c.id, c.codigo, c.local_id, l.nombre AS local_nombre, c.proveedor, c.fecha::text AS fecha, c.total,
	       c.estado, c.notas, c.nombre_usuario,
	       (SELECT m.codigo FROM movimientos_financieros m WHERE m.compra_id = c.id ORDER BY m.id LIMIT 1)
	           AS movimiento_financiero_codigo
	FROM compras_inventario c JOIN locales l ON l.id = c.local_id`

func (r *FinanzasRepo) ListCompras(localID *int) ([]models.Compra, error) {
	args := []interface{}{}
	cond := ""
	if localID != nil {
		args = append(args, *localID)
		cond = " WHERE c.local_id = $1"
	}
	compras := []models.Compra{}
	if err := r.db.Select(&compras, compraSelect+cond+" ORDER BY c.fecha DESC, c.id DESC LIMIT 500", args...); err != nil {
		return nil, fmt.Errorf("error al listar compras: %w", err)
	}
	return compras, nil
}

func (r *FinanzasRepo) GetCompra(id int) (*models.Compra, error) {
	var c models.Compra
	err := r.db.Get(&c, compraSelect+" WHERE c.id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrCompraNoEncontrada
	}
	if err != nil {
		return nil, fmt.Errorf("error al obtener compra: %w", err)
	}
	if err := r.db.Select(&c.Detalle, `
		SELECT d.item_id, i.nombre AS item_nombre, i.unidad_medida, d.cantidad, d.costo_unitario, d.subtotal
		FROM detalle_compras d JOIN inventario_items i ON i.id = d.item_id
		WHERE d.compra_id = $1 ORDER BY d.id`, id); err != nil {
		return nil, fmt.Errorf("error al obtener detalle de compra: %w", err)
	}
	return &c, nil
}

// AnularCompra anula la compra y su egreso y revierte las entradas de stock (RN-23). Si el stock de algun
// item ya se consumio y no alcanza para revertir, no cambia nada y devuelve stock insuficiente.
func (r *FinanzasRepo) AnularCompra(id int, motivo string, actor repository.ConsumoActor) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var cab struct {
		Codigo  string `db:"codigo"`
		LocalID int    `db:"local_id"`
		Estado  string `db:"estado"`
	}
	if err := tx.Get(&cab, `SELECT codigo, local_id, estado FROM compras_inventario WHERE id = $1 FOR UPDATE`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.ErrCompraNoEncontrada
		}
		return err
	}
	if cab.Estado == "ANULADA" {
		return errors.New("la compra ya esta anulada")
	}

	var detalle []struct {
		ItemID   int     `db:"item_id"`
		Nombre   string  `db:"nombre"`
		Cantidad float64 `db:"cantidad"`
	}
	if err := tx.Select(&detalle, `
		SELECT d.item_id, i.nombre, d.cantidad FROM detalle_compras d
		JOIN inventario_items i ON i.id = d.item_id WHERE d.compra_id = $1 ORDER BY d.id`, id); err != nil {
		return err
	}

	faltantes := []string{}
	for _, d := range detalle {
		var actual float64
		err := tx.Get(&actual,
			`SELECT cantidad FROM inventario_stock WHERE item_id = $1 AND local_id = $2 FOR UPDATE`, d.ItemID, cab.LocalID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if actual < d.Cantidad {
			faltantes = append(faltantes, fmt.Sprintf("%s (hay %v, se debe revertir %v)", d.Nombre, actual, d.Cantidad))
		}
	}
	if len(faltantes) > 0 {
		return fmt.Errorf("%w para revertir la compra: parte del stock ya se consumio: %s",
			repository.ErrStockInsuficiente, strings.Join(faltantes, "; "))
	}

	for _, d := range detalle {
		cid := id
		if _, err := AplicarMovimientoStock(tx, repository.MovimientoStockInput{
			ItemID: d.ItemID, LocalID: cab.LocalID, Tipo: repository.MovAjusteNegativo, Cantidad: d.Cantidad,
			CompraID: &cid, UsuarioID: actor.UsuarioID, UsuarioNombre: actor.UsuarioNombre,
			Motivo: fmt.Sprintf("Anulacion %s: %s", cab.Codigo, motivo),
		}); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`UPDATE compras_inventario SET estado = 'ANULADA', motivo_anulacion = $1 WHERE id = $2`, motivo, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE movimientos_financieros SET estado = 'ANULADO', motivo_anulacion = $1, fecha_modificacion = NOW()
		WHERE compra_id = $2 AND estado <> 'ANULADO'`, motivo, id); err != nil {
		return err
	}
	return tx.Commit()
}

// ─── Agregados del reporte ────────────────────────────────────────────────────

// AgregadoMovimientos son las sumas de movimientos manuales de un periodo.
type AgregadoMovimientos struct {
	IngresosOtros     float64 `db:"ingresos_otros"`
	EgresosFijos      float64 `db:"egresos_fijos"`
	EgresosVariables  float64 `db:"egresos_variables"`
	EgresosPendientes float64 `db:"egresos_pendientes"`
}

// Agregado suma movimientos PAGADO (pendientes por separado; anulados nunca). localID nil = todos los locales
// incluyendo los generales; soloGenerales limita a movimientos sin local.
func (r *FinanzasRepo) Agregado(desde, hasta time.Time, localID *int, soloGenerales bool) (AgregadoMovimientos, error) {
	args := []interface{}{desde, hasta}
	cond := ""
	if localID != nil {
		args = append(args, *localID)
		cond = " AND m.local_id = $3"
	} else if soloGenerales {
		cond = " AND m.local_id IS NULL"
	}
	var a AgregadoMovimientos
	err := r.db.Get(&a, fmt.Sprintf(`
		SELECT
			COALESCE(SUM(m.monto) FILTER (WHERE m.naturaleza = 'INGRESO' AND m.estado = 'PAGADO'), 0) AS ingresos_otros,
			COALESCE(SUM(m.monto) FILTER (WHERE m.naturaleza = 'EGRESO' AND m.clase = 'FIJO' AND m.estado = 'PAGADO'), 0) AS egresos_fijos,
			COALESCE(SUM(m.monto) FILTER (WHERE m.naturaleza = 'EGRESO' AND m.clase = 'VARIABLE' AND m.estado = 'PAGADO'), 0) AS egresos_variables,
			COALESCE(SUM(m.monto) FILTER (WHERE m.naturaleza = 'EGRESO' AND m.estado = 'PENDIENTE'), 0) AS egresos_pendientes
		FROM movimientos_financieros m
		WHERE m.fecha >= $1 AND m.fecha <= $2 AND m.estado <> 'ANULADO'%s`, cond), args...)
	if err != nil {
		return a, fmt.Errorf("error al agregar movimientos financieros: %w", err)
	}
	return a, nil
}

func (r *FinanzasRepo) EgresosPorCategoria(desde, hasta time.Time, localID *int) ([]models.EgresoPorCategoria, error) {
	args := []interface{}{desde, hasta}
	cond := ""
	if localID != nil {
		args = append(args, *localID)
		cond = " AND m.local_id = $3"
	}
	out := []models.EgresoPorCategoria{}
	err := r.db.Select(&out, fmt.Sprintf(`
		SELECT c.nombre AS categoria, m.clase, SUM(m.monto) AS total
		FROM movimientos_financieros m JOIN categorias_financieras c ON c.id = m.categoria_id
		WHERE m.naturaleza = 'EGRESO' AND m.estado = 'PAGADO' AND m.fecha >= $1 AND m.fecha <= $2%s
		GROUP BY c.nombre, m.clase ORDER BY total DESC`, cond), args...)
	if err != nil {
		return nil, fmt.Errorf("error al agrupar egresos por categoria: %w", err)
	}
	return out, nil
}

// VentasPeriodo suma pagos PAGADO activos del rango (misma regla que /bd/pagos/resumen).
func (r *FinanzasRepo) VentasPeriodo(desde, hasta time.Time, localID *int) (float64, error) {
	args := []interface{}{desde, hasta.AddDate(0, 0, 1)}
	cond := ""
	if localID != nil {
		args = append(args, *localID)
		cond = " AND p.local_id = $3"
	}
	var total float64
	if err := r.db.Get(&total, fmt.Sprintf(`
		SELECT COALESCE(SUM(p.total_final), 0) FROM pagos p
		WHERE p.activo = TRUE AND p.estado = 'PAGADO'
		  AND p.fecha_creacion >= $1 AND p.fecha_creacion < $2%s`, cond), args...); err != nil {
		return 0, fmt.Errorf("error al sumar ventas del periodo: %w", err)
	}
	return total, nil
}

// PuntoSerie es un periodo (dia o mes) con ingresos y egresos pagados.
type PuntoSerie struct {
	Periodo  string  `db:"periodo"`
	Ingresos float64 `db:"ingresos"`
	Egresos  float64 `db:"egresos"`
}

// Serie devuelve ingresos (ventas + otros ingresos) y egresos por periodo ('YYYY-MM' o 'YYYY-MM-DD').
func (r *FinanzasRepo) Serie(desde, hasta time.Time, localID *int, porDia bool) ([]PuntoSerie, error) {
	formato := "YYYY-MM"
	if porDia {
		formato = "YYYY-MM-DD"
	}
	args := []interface{}{desde, hasta, formato}
	condMov, condPag := "", ""
	if localID != nil {
		args = append(args, *localID)
		condMov = " AND m.local_id = $4"
		condPag = " AND p.local_id = $4"
	}
	out := []PuntoSerie{}
	err := r.db.Select(&out, fmt.Sprintf(`
		WITH mov AS (
			SELECT to_char(m.fecha, $3) AS periodo,
			       COALESCE(SUM(m.monto) FILTER (WHERE m.naturaleza = 'INGRESO'), 0) AS ingresos,
			       COALESCE(SUM(m.monto) FILTER (WHERE m.naturaleza = 'EGRESO'), 0) AS egresos
			FROM movimientos_financieros m
			WHERE m.estado = 'PAGADO' AND m.fecha >= $1 AND m.fecha <= $2%s GROUP BY 1
		), ven AS (
			SELECT to_char(p.fecha_creacion, $3) AS periodo, SUM(p.total_final) AS ingresos
			FROM pagos p
			WHERE p.activo = TRUE AND p.estado = 'PAGADO'
			  AND p.fecha_creacion >= $1 AND p.fecha_creacion < ($2::date + 1)%s GROUP BY 1
		)
		SELECT COALESCE(mov.periodo, ven.periodo) AS periodo,
		       COALESCE(mov.ingresos, 0) + COALESCE(ven.ingresos, 0) AS ingresos,
		       COALESCE(mov.egresos, 0) AS egresos
		FROM mov FULL OUTER JOIN ven ON ven.periodo = mov.periodo
		ORDER BY 1`, condMov, condPag), args...)
	if err != nil {
		return nil, fmt.Errorf("error al calcular serie financiera: %w", err)
	}
	return out, nil
}

// LocalBasico es un local activo para el detalle por local.
type LocalBasico struct {
	ID     int    `db:"id"`
	Nombre string `db:"nombre"`
}

func (r *FinanzasRepo) LocalesActivos() ([]LocalBasico, error) {
	var locales []LocalBasico
	if err := r.db.Select(&locales, `SELECT id, nombre FROM locales WHERE activo = TRUE ORDER BY nombre`); err != nil {
		return nil, err
	}
	return locales, nil
}

func (r *FinanzasRepo) LocalIDPorNombre(nombre string) (*int, error) {
	var id int
	err := r.db.Get(&id, `SELECT id FROM locales WHERE UPPER(nombre) = UPPER($1)`, nombre)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("local no encontrado")
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}
