package pgsql

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"

	"github.com/jmoiron/sqlx"
)

type InventarioRepo struct {
	db *sqlx.DB
}

func NewInventarioRepo(db *sqlx.DB) *InventarioRepo {
	return &InventarioRepo{db: db}
}

// DB expone la conexion para que servicios coordinen transacciones que cruzan
// inventario, finanzas y reservas.
func (r *InventarioRepo) DB() *sqlx.DB { return r.db }

func redondear3(v float64) float64 { return math.Round(v*1000) / 1000 }

const itemsSelectBase = `
	SELECT i.id, i.codigo, i.nombre, i.descripcion, i.categoria_id, c.nombre AS categoria_nombre,
	       i.unidad_medida, i.costo_referencia, i.stock_minimo,
	       COALESCE((SELECT SUM(s.cantidad) FROM inventario_stock s
	                 WHERE s.item_id = i.id %s), 0) AS stock_actual,
	       i.imagen_key, i.activo
	FROM inventario_items i
	LEFT JOIN categorias_inventario c ON c.id = i.categoria_id`

// ─── Items ────────────────────────────────────────────────────────────────────

func (r *InventarioRepo) ListItems(f repository.FiltroItemsInventario) ([]models.InventarioItem, error) {
	args := []interface{}{}
	localCond := ""
	if f.LocalID != nil {
		args = append(args, *f.LocalID)
		localCond = fmt.Sprintf("AND s.local_id = $%d", len(args))
	}

	conds := []string{"TRUE"}
	if f.Activo != nil {
		args = append(args, *f.Activo)
		conds = append(conds, fmt.Sprintf("i.activo = $%d", len(args)))
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		args = append(args, "%"+strings.ToLower(q)+"%")
		conds = append(conds, fmt.Sprintf("(LOWER(i.nombre) LIKE $%d OR LOWER(i.codigo) LIKE $%d)", len(args), len(args)))
	}
	if f.CategoriaID != nil {
		args = append(args, *f.CategoriaID)
		conds = append(conds, fmt.Sprintf("i.categoria_id = $%d", len(args)))
	}

	query := fmt.Sprintf(itemsSelectBase, localCond) + " WHERE " + strings.Join(conds, " AND ")
	if f.BajoStock {
		query = "SELECT * FROM (" + query + ") t WHERE t.stock_actual <= t.stock_minimo"
		query += " ORDER BY t.nombre, t.id"
	} else {
		query += " ORDER BY i.nombre, i.id"
	}
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	args = append(args, limit)
	query += fmt.Sprintf(" LIMIT $%d", len(args))

	items := []models.InventarioItem{}
	if err := r.db.Select(&items, query, args...); err != nil {
		return nil, fmt.Errorf("error al listar items de inventario: %w", err)
	}
	return items, nil
}

func (r *InventarioRepo) GetItem(id int) (*models.InventarioItem, error) {
	var item models.InventarioItem
	err := r.db.Get(&item, fmt.Sprintf(itemsSelectBase, "")+" WHERE i.id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrItemNoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("error al obtener item: %w", err)
	}
	if err := r.db.Select(&item.StockPorLocal, `
		SELECT l.id AS local_id, l.nombre AS local_nombre, COALESCE(s.cantidad, 0) AS cantidad
		FROM locales l
		LEFT JOIN inventario_stock s ON s.local_id = l.id AND s.item_id = $1
		WHERE l.activo = TRUE OR s.cantidad IS NOT NULL
		ORDER BY l.nombre`, id); err != nil {
		return nil, fmt.Errorf("error al obtener stock por local: %w", err)
	}
	return &item, nil
}

type ItemInput struct {
	Nombre          string
	Descripcion     *string
	CategoriaID     *int
	UnidadMedida    string
	CostoReferencia *float64
	StockMinimo     float64
}

func (r *InventarioRepo) CreateItem(in ItemInput) (int, error) {
	var id int
	err := r.db.QueryRowx(`
		INSERT INTO inventario_items (nombre, descripcion, categoria_id, unidad_medida, costo_referencia, stock_minimo)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		in.Nombre, in.Descripcion, in.CategoriaID, in.UnidadMedida, in.CostoReferencia, in.StockMinimo,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("error al crear item: %w", err)
	}
	return id, nil
}

// ItemPatch contiene solo los campos presentes en el PATCH.
type ItemPatch struct {
	Nombre             *string
	Descripcion        *string
	DescripcionSet     bool
	CategoriaID        *int
	CategoriaSet       bool
	UnidadMedida       *string
	CostoReferencia    *float64
	CostoReferenciaSet bool
	StockMinimo        *float64
	Activo             *bool
}

func (r *InventarioRepo) UpdateItem(id int, p ItemPatch) error {
	sets := []string{"fecha_modificacion = NOW()"}
	args := []interface{}{}
	add := func(col string, v interface{}) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	if p.Nombre != nil {
		add("nombre", *p.Nombre)
	}
	if p.DescripcionSet {
		add("descripcion", p.Descripcion)
	}
	if p.CategoriaSet {
		add("categoria_id", p.CategoriaID)
	}
	if p.UnidadMedida != nil {
		add("unidad_medida", *p.UnidadMedida)
	}
	if p.CostoReferenciaSet {
		add("costo_referencia", p.CostoReferencia)
	}
	if p.StockMinimo != nil {
		add("stock_minimo", *p.StockMinimo)
	}
	if p.Activo != nil {
		add("activo", *p.Activo)
	}
	args = append(args, id)
	res, err := r.db.Exec(
		fmt.Sprintf("UPDATE inventario_items SET %s WHERE id = $%d", strings.Join(sets, ", "), len(args)), args...)
	if err != nil {
		return fmt.Errorf("error al actualizar item: %w", err)
	}
	n, err := affectedRows(res, "actualizar item")
	if err != nil {
		return err
	}
	if n == 0 {
		return repository.ErrItemNoEncontrado
	}
	return nil
}

func (r *InventarioRepo) SetImagenKey(id int, key *string) error {
	res, err := r.db.Exec(
		`UPDATE inventario_items SET imagen_key = $1, fecha_modificacion = NOW() WHERE id = $2`, key, id)
	if err != nil {
		return fmt.Errorf("error al actualizar imagen del item: %w", err)
	}
	n, err := affectedRows(res, "actualizar imagen del item")
	if err != nil {
		return err
	}
	if n == 0 {
		return repository.ErrItemNoEncontrado
	}
	return nil
}

func (r *InventarioRepo) ItemExiste(id int) (bool, error) {
	var n int
	if err := r.db.Get(&n, `SELECT COUNT(*) FROM inventario_items WHERE id = $1`, id); err != nil {
		return false, err
	}
	return n > 0, nil
}

// ─── Categorias ───────────────────────────────────────────────────────────────

func (r *InventarioRepo) ListCategorias() ([]models.CategoriaInventario, error) {
	cats := []models.CategoriaInventario{}
	if err := r.db.Select(&cats, `SELECT id, nombre, activo FROM categorias_inventario ORDER BY nombre`); err != nil {
		return nil, fmt.Errorf("error al listar categorias de inventario: %w", err)
	}
	return cats, nil
}

func (r *InventarioRepo) CreateCategoria(nombre string) (int, error) {
	var id int
	if err := r.db.QueryRowx(
		`INSERT INTO categorias_inventario (nombre) VALUES ($1) RETURNING id`, nombre).Scan(&id); err != nil {
		return 0, traducirDuplicado(err, "ya existe una categoria de inventario con ese nombre")
	}
	return id, nil
}

func (r *InventarioRepo) UpdateCategoria(id int, nombre *string, activo *bool) error {
	res, err := r.db.Exec(`
		UPDATE categorias_inventario
		SET nombre = COALESCE($1, nombre), activo = COALESCE($2, activo)
		WHERE id = $3`, nombre, activo, id)
	if err != nil {
		return traducirDuplicado(err, "ya existe una categoria de inventario con ese nombre")
	}
	n, err := affectedRows(res, "actualizar categoria de inventario")
	if err != nil {
		return err
	}
	if n == 0 {
		return repository.ErrCategoriaNoEnc
	}
	return nil
}

// ─── Stock y kardex ───────────────────────────────────────────────────────────

func signoMovimiento(tipo string) float64 {
	switch tipo {
	case repository.MovEntradaCompra, repository.MovReversionServicio, repository.MovAjustePositivo:
		return 1
	default:
		return -1
	}
}

// AplicarMovimientoStock bloquea la fila de stock, valida y actualiza la
// cantidad y registra el kardex en la misma transaccion (RN-02). Devuelve el
// stock resultante. El bloqueo FOR UPDATE evita que dos descuentos simultaneos
// lean el mismo stock.
func AplicarMovimientoStock(tx *sqlx.Tx, in repository.MovimientoStockInput) (float64, error) {
	if in.Cantidad <= 0 {
		return 0, errors.New("la cantidad debe ser mayor a 0")
	}
	if _, err := tx.Exec(`
		INSERT INTO inventario_stock (item_id, local_id, cantidad) VALUES ($1, $2, 0)
		ON CONFLICT (item_id, local_id) DO NOTHING`, in.ItemID, in.LocalID); err != nil {
		return 0, fmt.Errorf("error al preparar stock: %w", err)
	}
	var actual float64
	if err := tx.Get(&actual,
		`SELECT cantidad FROM inventario_stock WHERE item_id = $1 AND local_id = $2 FOR UPDATE`,
		in.ItemID, in.LocalID); err != nil {
		return 0, fmt.Errorf("error al bloquear stock: %w", err)
	}

	nuevo := redondear3(actual + signoMovimiento(in.Tipo)*in.Cantidad)
	if nuevo < 0 && !in.PermitirNegativo {
		return 0, fmt.Errorf("%w: stock disponible %v, se requieren %v", repository.ErrStockInsuficiente, actual, in.Cantidad)
	}

	if _, err := tx.Exec(`
		UPDATE inventario_stock SET cantidad = $1, fecha_modificacion = NOW()
		WHERE item_id = $2 AND local_id = $3`, nuevo, in.ItemID, in.LocalID); err != nil {
		return 0, fmt.Errorf("error al actualizar stock: %w", err)
	}
	if _, err := tx.Exec(`
		INSERT INTO inventario_movimientos
			(item_id, local_id, tipo, cantidad, stock_resultante, costo_unitario, reserva_id,
			 plan_movimiento_id, compra_id, motivo, usuario_id, usuario_nombre)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		in.ItemID, in.LocalID, in.Tipo, in.Cantidad, nuevo, in.CostoUnitario, in.ReservaID,
		in.PlanMovimientoID, in.CompraID, nullStr(in.Motivo), in.UsuarioID, nullStr(in.UsuarioNombre)); err != nil {
		return 0, fmt.Errorf("error al registrar movimiento de inventario: %w", err)
	}
	return nuevo, nil
}

// Ajustar registra un ajuste manual en su propia transaccion.
func (r *InventarioRepo) Ajustar(in repository.MovimientoStockInput) (float64, error) {
	existe, err := r.ItemExiste(in.ItemID)
	if err != nil {
		return 0, err
	}
	if !existe {
		return 0, repository.ErrItemNoEncontrado
	}
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	nuevo, err := AplicarMovimientoStock(tx, in)
	if err != nil {
		return 0, err
	}
	return nuevo, tx.Commit()
}

func (r *InventarioRepo) ListMovimientos(f repository.FiltroMovimientosInventario) ([]models.MovimientoInventario, error) {
	conds := []string{"TRUE"}
	args := []interface{}{}
	add := func(cond string, v interface{}) {
		args = append(args, v)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	if f.ItemID != nil {
		add("m.item_id = $%d", *f.ItemID)
	}
	if f.LocalID != nil {
		add("m.local_id = $%d", *f.LocalID)
	}
	if f.Tipo != "" {
		add("m.tipo = $%d", f.Tipo)
	}
	if f.FechaDesde != nil {
		add("m.creado_en >= $%d", *f.FechaDesde)
	}
	if f.FechaHasta != nil {
		add("m.creado_en < $%d", f.FechaHasta.AddDate(0, 0, 1))
	}
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	args = append(args, limit)

	movs := []models.MovimientoInventario{}
	err := r.db.Select(&movs, fmt.Sprintf(`
		SELECT m.id, m.item_id, i.nombre AS item_nombre, m.local_id, l.nombre AS local_nombre, m.tipo,
		       m.cantidad, m.stock_resultante, m.costo_unitario, m.reserva_id, m.compra_id, m.motivo,
		       m.usuario_nombre, m.creado_en
		FROM inventario_movimientos m
		JOIN inventario_items i ON i.id = m.item_id
		JOIN locales l ON l.id = m.local_id
		WHERE %s
		ORDER BY m.creado_en DESC, m.id DESC
		LIMIT $%d`, strings.Join(conds, " AND "), len(args)), args...)
	if err != nil {
		return nil, fmt.Errorf("error al listar movimientos de inventario: %w", err)
	}
	return movs, nil
}

// ─── Insumos por servicio ─────────────────────────────────────────────────────

func (r *InventarioRepo) ServicioExiste(id int) (bool, error) {
	var n int
	if err := r.db.Get(&n, `SELECT COUNT(*) FROM servicios WHERE id = $1`, id); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *InventarioRepo) ListInsumos(servicioID int) ([]models.ServicioInsumo, error) {
	insumos := []models.ServicioInsumo{}
	err := r.db.Select(&insumos, `
		SELECT si.item_id, i.nombre AS item_nombre, i.unidad_medida, si.cantidad_por_sesion
		FROM servicio_insumos si JOIN inventario_items i ON i.id = si.item_id
		WHERE si.servicio_id = $1 ORDER BY i.nombre`, servicioID)
	if err != nil {
		return nil, fmt.Errorf("error al listar insumos del servicio: %w", err)
	}
	return insumos, nil
}

// ReplaceInsumos reemplaza la lista completa de insumos del servicio.
func (r *InventarioRepo) ReplaceInsumos(servicioID int, insumos []models.ServicioInsumo) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM servicio_insumos WHERE servicio_id = $1`, servicioID); err != nil {
		return fmt.Errorf("error al limpiar insumos: %w", err)
	}
	for _, in := range insumos {
		if _, err := tx.Exec(`
			INSERT INTO servicio_insumos (servicio_id, item_id, cantidad_por_sesion) VALUES ($1,$2,$3)`,
			servicioID, in.ItemID, in.CantidadPorSesion); err != nil {
			return fmt.Errorf("error al guardar insumo %d: %w", in.ItemID, err)
		}
	}
	return tx.Commit()
}

// ServiciosQueUsanItem lista los nombres de servicios que consumen el item.
func (r *InventarioRepo) ServiciosQueUsanItem(itemID int) ([]string, error) {
	nombres := []string{}
	if err := r.db.Select(&nombres, `
		SELECT s.nombre FROM servicio_insumos si JOIN servicios s ON s.id = si.servicio_id
		WHERE si.item_id = $1 ORDER BY s.nombre`, itemID); err != nil {
		return nil, err
	}
	return nombres, nil
}

// ─── Alertas ──────────────────────────────────────────────────────────────────

type reservaAlertaRow struct {
	ID        int    `db:"id"`
	Fecha     string `db:"fecha"`
	HoraDesde string `db:"hora_desde"`
	Cliente   string `db:"cliente"`
	Servicio  string `db:"servicio"`
	Local     string `db:"local"`
}

// ReservasConAlerta devuelve reservas futuras activas marcadas con alerta de inventario.
func (r *InventarioRepo) ReservasConAlerta(localID *int) ([]models.AlertaReservaInventario, error) {
	args := []interface{}{}
	cond := ""
	if localID != nil {
		args = append(args, *localID)
		cond = "AND r.local_id = $1"
	}
	var rows []reservaAlertaRow
	err := r.db.Select(&rows, fmt.Sprintf(`
		SELECT r.id, r.fecha::text AS fecha, to_char(r.hora_desde, 'HH24:MI') AS hora_desde, r.cliente,
		       COALESCE(NULLIF(r.servicio_confirmado, ''), r.servicio_nombre, r.servicio_solicitado, '') AS servicio,
		       r.local_nombre AS local
		FROM reservas r
		WHERE r.activo = TRUE AND r.alerta_inventario = TRUE AND r.fecha >= CURRENT_DATE
		  AND COALESCE(r.estado, 'PENDIENTE') IN ('PENDIENTE', 'AGENDADO') %s
		ORDER BY r.fecha, r.hora_desde LIMIT 100`, cond), args...)
	if err != nil {
		return nil, fmt.Errorf("error al listar reservas con alerta: %w", err)
	}
	out := make([]models.AlertaReservaInventario, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.AlertaReservaInventario{
			ReservaID: row.ID, Fecha: row.Fecha, HoraDesde: row.HoraDesde, Cliente: row.Cliente,
			Servicio: row.Servicio, Local: row.Local, Advertencias: []models.AdvertenciaInventario{},
		})
	}
	return out, nil
}

// ValorStock suma stock * costo_referencia de items activos (opcionalmente de un local).
func (r *InventarioRepo) ValorStock(localID *int) (float64, int, error) {
	args := []interface{}{}
	cond := ""
	if localID != nil {
		args = append(args, *localID)
		cond = "AND s.local_id = $1"
	}
	var valor float64
	if err := r.db.Get(&valor, fmt.Sprintf(`
		SELECT COALESCE(SUM(GREATEST(s.cantidad, 0) * COALESCE(i.costo_referencia, 0)), 0)
		FROM inventario_stock s JOIN inventario_items i ON i.id = s.item_id
		WHERE i.activo = TRUE %s`, cond), args...); err != nil {
		return 0, 0, fmt.Errorf("error al calcular valor de stock: %w", err)
	}
	var bajos int
	if err := r.db.Get(&bajos, fmt.Sprintf(`
		SELECT COUNT(*) FROM (
			SELECT i.id FROM inventario_items i
			WHERE i.activo = TRUE
			  AND COALESCE((SELECT SUM(s.cantidad) FROM inventario_stock s
			                WHERE s.item_id = i.id %s), 0) <= i.stock_minimo
		) t`, cond), args...); err != nil {
		return 0, 0, fmt.Errorf("error al contar items bajo stock: %w", err)
	}
	return valor, bajos, nil
}

// ConsumoServicios suma el costo de las salidas por servicio en el rango (valorizado a costo unitario del item).
func (r *InventarioRepo) ConsumoServicios(desde, hasta string, localID *int) (float64, error) {
	args := []interface{}{desde, hasta}
	cond := ""
	if localID != nil {
		args = append(args, *localID)
		cond = "AND m.local_id = $3"
	}
	var total float64
	err := r.db.Get(&total, fmt.Sprintf(`
		SELECT COALESCE(SUM(
			CASE WHEN m.tipo = 'SALIDA_SERVICIO' THEN m.cantidad ELSE -m.cantidad END
			* COALESCE(m.costo_unitario, i.costo_referencia, 0)), 0)
		FROM inventario_movimientos m JOIN inventario_items i ON i.id = m.item_id
		WHERE m.tipo IN ('SALIDA_SERVICIO', 'REVERSION_SERVICIO')
		  AND m.creado_en >= $1::date AND m.creado_en < ($2::date + 1) %s`, cond), args...)
	if err != nil {
		return 0, fmt.Errorf("error al calcular consumo de servicios: %w", err)
	}
	return total, nil
}

func traducirDuplicado(err error, mensaje string) error {
	if err != nil && strings.Contains(err.Error(), "23505") {
		return errors.New(mensaje)
	}
	return err
}
