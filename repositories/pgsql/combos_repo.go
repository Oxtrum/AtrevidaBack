package pgsql

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

var _ repository.CombosRepository = (*CombosRepo)(nil)

type CombosRepo struct {
	db *sqlx.DB
}

func NewCombosRepo(db *sqlx.DB) *CombosRepo {
	return &CombosRepo{db: db}
}

func (r *CombosRepo) ListCombos(f repository.FiltroCombos) ([]models.ComboCatalogoPG, int, error) {
	conditions, args := comboConditions(f)
	idx := len(args) + 1
	if f.CursorSet {
		conditions = append(conditions, fmt.Sprintf("(cb.nombre, cb.id) > ($%d, $%d)", idx, idx+1))
		args = append(args, f.CursorNombre, f.CursorID)
		idx += 2
	}
	limitClause := ""
	if f.PageLimit > 0 {
		limitClause = fmt.Sprintf(" LIMIT $%d", idx)
		args = append(args, f.PageLimit)
	}
	where := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT
			cb.id, cb.nombre, cb.descripcion, cb.categoria_id,
			COALESCE(c.nombre, '') AS categoria,
			cb.tipo_precio, cb.precio_paquete,
			COALESCE((SELECT SUM(cs2.costo * cs2.sesiones) FROM combo_servicios cs2 WHERE cs2.combo_id = cb.id AND cs2.activo = TRUE), 0) AS precio_items,
			CASE
				WHEN cb.tipo_precio = 'PRECIO_PAQUETE' THEN COALESCE(cb.precio_paquete, cb.costo_total, 0)
				ELSE COALESCE((SELECT SUM(cs2.costo * cs2.sesiones) FROM combo_servicios cs2 WHERE cs2.combo_id = cb.id AND cs2.activo = TRUE), 0)
			END AS precio_final,
			cb.moneda, cb.sesiones_totales, cb.duracion_min, cb.activo, cb.creado_en, cb.actualizado_en, cb.imagen_path,
			cb.paquete_id, cb.precio_regular, cb.nota
		FROM combos cb
		LEFT JOIN categorias c ON c.id = cb.categoria_id
		LEFT JOIN combo_local cl ON cl.combo_id = cb.id
		LEFT JOIN locales l ON l.id = cl.local_id
		WHERE %s
		GROUP BY cb.id, c.nombre
		ORDER BY cb.nombre, cb.id%s
	`, where, limitClause)

	var combos []models.ComboCatalogoPG
	ctx := queryContext(f.Context)
	if err := r.db.SelectContext(ctx, &combos, query, args...); err != nil {
		return nil, 0, fmt.Errorf("error al listar combos: %w", err)
	}
	if combos == nil {
		combos = []models.ComboCatalogoPG{}
	}
	if err := r.cargarDetallesCombos(ctx, combos); err != nil {
		return nil, 0, err
	}
	return combos, len(combos), nil
}

func (r *CombosRepo) CountCombos(f repository.FiltroCombos) (int, error) {
	conditions, args := comboConditions(f)
	query := fmt.Sprintf(`SELECT COUNT(DISTINCT cb.id)
		FROM combos cb
		LEFT JOIN categorias c ON c.id = cb.categoria_id
		LEFT JOIN combo_local cl ON cl.combo_id = cb.id
		LEFT JOIN locales l ON l.id = cl.local_id
		WHERE %s`, strings.Join(conditions, " AND "))
	var total int
	if err := r.db.GetContext(queryContext(f.Context), &total, query, args...); err != nil {
		return 0, fmt.Errorf("error al contar combos: %w", err)
	}
	return total, nil
}

func (r *CombosRepo) cargarDetallesCombos(ctx context.Context, combos []models.ComboCatalogoPG) error {
	if len(combos) == 0 {
		return nil
	}
	ids := make([]int, 0, len(combos))
	indices := make(map[int]int, len(combos))
	for i := range combos {
		ids = append(ids, combos[i].ID)
		indices[combos[i].ID] = i
		combos[i].Locales = []models.LocalPG{}
		combos[i].Servicios = []models.ComboServicioDetallePG{}
	}
	q, args, err := sqlx.In(`SELECT cl.combo_id, l.id, l.nombre, l.activo
		FROM combo_local cl JOIN locales l ON l.id=cl.local_id
		WHERE cl.combo_id IN (?) ORDER BY cl.combo_id,l.nombre`, ids)
	if err != nil {
		return fmt.Errorf("error al preparar locales de combos: %w", err)
	}
	var locales []struct {
		ComboID int `db:"combo_id"`
		models.LocalPG
	}
	if err := r.db.SelectContext(ctx, &locales, r.db.Rebind(q), args...); err != nil {
		return fmt.Errorf("error al cargar locales de combos: %w", err)
	}
	for _, item := range locales {
		if i, ok := indices[item.ComboID]; ok {
			combos[i].Locales = append(combos[i].Locales, item.LocalPG)
		}
	}
	q, args, err = sqlx.In(`SELECT cs.id,cs.combo_id,cb.nombre combo_nombre,cs.servicio_id,
		cs.servicio_texto,COALESCE(cs.servicio_texto,'') servicio_nombre,cs.tiempo,cs.costo,
		cs.sesiones,cs.sesion_numero,cs.orden,cs.activo
		FROM combo_servicios cs JOIN combos cb ON cb.id=cs.combo_id
		WHERE cs.combo_id IN (?) AND cs.activo=TRUE ORDER BY cs.combo_id,cs.orden,cs.id`, ids)
	if err != nil {
		return fmt.Errorf("error al preparar servicios de combos: %w", err)
	}
	var servicios []models.ComboServicioDetallePG
	if err := r.db.SelectContext(ctx, &servicios, r.db.Rebind(q), args...); err != nil {
		return fmt.Errorf("error al cargar servicios de combos: %w", err)
	}
	for _, item := range servicios {
		if i, ok := indices[item.ComboID]; ok {
			combos[i].Servicios = append(combos[i].Servicios, item)
		}
	}
	return nil
}

func (r *CombosRepo) GetComboByID(id int, incluirInactivo bool) (*models.ComboCatalogoPG, error) {
	condition := "cb.id = $1"
	if !incluirInactivo {
		condition += " AND cb.activo = TRUE"
	}
	var combo models.ComboCatalogoPG
	err := r.db.Get(&combo, fmt.Sprintf(`
		SELECT
			cb.id, cb.nombre, cb.descripcion, cb.categoria_id,
			COALESCE(c.nombre, '') AS categoria,
			cb.tipo_precio, cb.precio_paquete,
			COALESCE(SUM(cs.costo * cs.sesiones) FILTER (WHERE cs.activo = TRUE), 0) AS precio_items,
			CASE
				WHEN cb.tipo_precio = 'PRECIO_PAQUETE' THEN COALESCE(cb.precio_paquete, cb.costo_total, 0)
				ELSE COALESCE(SUM(cs.costo * cs.sesiones) FILTER (WHERE cs.activo = TRUE), 0)
			END AS precio_final,
			cb.moneda, cb.sesiones_totales, cb.duracion_min, cb.activo, cb.creado_en, cb.actualizado_en, cb.imagen_path,
			cb.paquete_id, cb.precio_regular, cb.nota
		FROM combos cb
		LEFT JOIN categorias c ON c.id = cb.categoria_id
		LEFT JOIN combo_servicios cs ON cs.combo_id = cb.id
		WHERE %s
		GROUP BY cb.id, c.nombre
	`, condition), id)
	if err != nil {
		return nil, fmt.Errorf("%w: combo con id %d", repository.ErrComboNoEncontrado, id)
	}
	if err := r.cargarDetalleCombo(&combo, incluirInactivo); err != nil {
		return nil, err
	}
	return &combo, nil
}

func (r *CombosRepo) CreateCombo(input repository.CrearComboInput) (int, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, fmt.Errorf("error al iniciar transaccion de combo: %w", err)
	}
	defer tx.Rollback()

	if err := validarCategoriaTx(tx, input.CategoriaID); err != nil {
		return 0, err
	}
	if err := validarLocalesTx(tx, input.LocalIDs); err != nil {
		return 0, err
	}
	servicios, err := materializarServiciosTx(tx, input.Servicios)
	if err != nil {
		return 0, err
	}
	if err := validarPrecioPorItems(input.TipoPrecio, servicios); err != nil {
		return 0, err
	}

	precioItems, _ := resumenServicios(servicios)
	precioFinal := precioItems
	if input.TipoPrecio == "PRECIO_PAQUETE" {
		precioFinal = *input.PrecioPaquete
	}
	// Sesiones del paquete = cantidad de sesiones distintas presentes en las líneas.
	sesionesTotales := contarSesiones(servicios)
	var comboID int
	err = tx.QueryRowx(`
		INSERT INTO combos (
			nombre, descripcion, categoria_id, tipo_precio, precio_paquete,
			moneda, costo_total, sesiones_totales, duracion_min, activo
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,TRUE)
		RETURNING id
	`, input.Nombre, nullStr(pointerString(input.Descripcion)), input.CategoriaID,
		input.TipoPrecio, input.PrecioPaquete, input.Moneda, precioFinal, sesionesTotales, input.DuracionMin).Scan(&comboID)
	if err != nil {
		return 0, fmt.Errorf("error al crear combo: %w", err)
	}
	if err := insertarLocalesTx(tx, comboID, input.LocalIDs); err != nil {
		return 0, err
	}
	if err := insertarServiciosTx(tx, comboID, servicios); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("error al confirmar combo: %w", err)
	}
	return comboID, nil
}

func (r *CombosRepo) UpdateCombo(input repository.ActualizarComboInput) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var current struct {
		TipoPrecio string   `db:"tipo_precio"`
		Precio     *float64 `db:"precio_paquete"`
	}
	if err := tx.Get(&current, `SELECT tipo_precio, precio_paquete FROM combos WHERE id = $1`, input.ID); err != nil {
		return fmt.Errorf("%w: combo con id %d", repository.ErrComboNoEncontrado, input.ID)
	}
	if err := validarCategoriaTx(tx, input.CategoriaID); err != nil {
		return err
	}

	tipoPrecio := current.TipoPrecio
	if input.TipoPrecio != nil {
		tipoPrecio = *input.TipoPrecio
	}
	precioPaquete := current.Precio
	if input.PrecioPaquete != nil {
		precioPaquete = input.PrecioPaquete
	}
	if tipoPrecio == "PRECIO_PAQUETE" && precioPaquete == nil {
		return fmt.Errorf("%w: precio_paquete es requerido para PRECIO_PAQUETE", repository.ErrComboDatosInvalidos)
	}
	if tipoPrecio == "POR_ITEMS" && input.PrecioPaquete != nil {
		return fmt.Errorf("%w: precio_paquete no aplica para POR_ITEMS", repository.ErrComboDatosInvalidos)
	}

	sets := []string{"actualizado_en = NOW()"}
	args := []interface{}{}
	index := 1
	add := func(column string, value interface{}) {
		sets = append(sets, fmt.Sprintf("%s = $%d", column, index))
		args = append(args, value)
		index++
	}
	if input.Nombre != nil {
		add("nombre", *input.Nombre)
	}
	if input.Descripcion != nil {
		add("descripcion", nullStr(*input.Descripcion))
	}
	if input.CategoriaID != nil {
		add("categoria_id", *input.CategoriaID)
	}
	if input.TipoPrecio != nil {
		add("tipo_precio", *input.TipoPrecio)
	}
	if input.PrecioPaquete != nil && !(input.TipoPrecio != nil && *input.TipoPrecio == "POR_ITEMS") {
		add("precio_paquete", *input.PrecioPaquete)
	}
	if input.Moneda != nil {
		add("moneda", *input.Moneda)
	}
	if input.DuracionMin != nil {
		add("duracion_min", *input.DuracionMin)
	}
	if input.TipoPrecio != nil && *input.TipoPrecio == "POR_ITEMS" {
		add("precio_paquete", nil)
	}
	args = append(args, input.ID)
	query := fmt.Sprintf("UPDATE combos SET %s WHERE id = $%d", strings.Join(sets, ", "), index)
	if _, err := tx.Exec(query, args...); err != nil {
		return fmt.Errorf("error al actualizar combo: %w", err)
	}
	if err := actualizarResumenTx(tx, input.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *CombosRepo) SetComboImagen(id int, path *string) error {
	result, err := r.db.Exec(`UPDATE combos SET imagen_path = $1, actualizado_en = NOW() WHERE id = $2`, path, id)
	if err != nil {
		return fmt.Errorf("error al actualizar imagen de combo: %w", err)
	}
	n, err := affectedRows(result, "actualizar imagen de combo")
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%w: combo con id %d", repository.ErrComboNoEncontrado, id)
	}
	return nil
}

func (r *CombosRepo) SetComboActivo(id int, activo bool) error {
	result, err := r.db.Exec(`UPDATE combos SET activo = $1, actualizado_en = NOW() WHERE id = $2`, activo, id)
	if err != nil {
		return fmt.Errorf("error al actualizar estado de combo: %w", err)
	}
	n, err := affectedRows(result, "actualizar estado de combo")
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%w: combo con id %d", repository.ErrComboNoEncontrado, id)
	}
	return nil
}

func (r *CombosRepo) SetComboLocales(comboID int, localIDs []int) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := validarComboExisteTx(tx, comboID); err != nil {
		return err
	}
	if err := validarLocalesTx(tx, localIDs); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM combo_local WHERE combo_id = $1`, comboID); err != nil {
		return fmt.Errorf("error al reemplazar locales de combo: %w", err)
	}
	if err := insertarLocalesTx(tx, comboID, localIDs); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE combos SET actualizado_en = NOW() WHERE id = $1`, comboID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *CombosRepo) ReplaceComboServicios(comboID int, inputs []repository.ComboServicioCatalogoInput) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var tipoPrecio string
	if err := tx.Get(&tipoPrecio, `SELECT tipo_precio FROM combos WHERE id = $1 AND activo = TRUE`, comboID); err != nil {
		return fmt.Errorf("%w: combo con id %d no encontrado o inactivo", repository.ErrComboNoEncontrado, comboID)
	}
	servicios, err := materializarServiciosTx(tx, inputs)
	if err != nil {
		return err
	}
	if err := validarPrecioPorItems(tipoPrecio, servicios); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE combo_servicios SET activo = FALSE WHERE combo_id = $1 AND activo = TRUE`, comboID); err != nil {
		return fmt.Errorf("error al desactivar detalle anterior: %w", err)
	}
	if err := insertarServiciosTx(tx, comboID, servicios); err != nil {
		return err
	}
	if err := actualizarResumenTx(tx, comboID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *CombosRepo) cargarDetalleCombo(combo *models.ComboCatalogoPG, incluirInactivo bool) error {
	var locales []models.LocalPG
	if err := r.db.Select(&locales, `
		SELECT l.id, l.nombre, l.activo
		FROM locales l JOIN combo_local cl ON cl.local_id = l.id
		WHERE cl.combo_id = $1 ORDER BY l.nombre
	`, combo.ID); err != nil {
		return fmt.Errorf("error al cargar locales de combo: %w", err)
	}
	if locales == nil {
		locales = []models.LocalPG{}
	}
	combo.Locales = locales

	condition := "cs.combo_id = $1"
	if !incluirInactivo {
		condition += " AND cs.activo = TRUE"
	}
	var servicios []models.ComboServicioDetallePG
	if err := r.db.Select(&servicios, fmt.Sprintf(`
		SELECT cs.id, cs.combo_id, cb.nombre AS combo_nombre, cs.servicio_id,
			cs.servicio_texto, COALESCE(cs.servicio_texto, '') AS servicio_nombre,
			cs.tiempo, cs.costo, cs.sesiones, cs.sesion_numero, cs.orden, cs.activo
		FROM combo_servicios cs JOIN combos cb ON cb.id = cs.combo_id
		WHERE %s ORDER BY cs.orden, cs.id
	`, condition), combo.ID); err != nil {
		return fmt.Errorf("error al cargar servicios de combo: %w", err)
	}
	if servicios == nil {
		servicios = []models.ComboServicioDetallePG{}
	}
	combo.Servicios = servicios
	return nil
}

func comboConditions(f repository.FiltroCombos) ([]string, []interface{}) {
	conditions := []string{"1 = 1"}
	args := []interface{}{}
	index := 1
	add := func(condition string, value interface{}) {
		conditions = append(conditions, fmt.Sprintf(condition, index))
		args = append(args, value)
		index++
	}
	if f.Activo != nil {
		add("cb.activo = $%d", *f.Activo)
	}
	if f.Nombre != "" {
		add("cb.nombre ILIKE '%%' || $%d || '%%'", f.Nombre)
	}
	if f.Categoria != "" {
		add("c.nombre ILIKE '%%' || $%d || '%%'", f.Categoria)
	}
	if f.LocalID != nil {
		add("l.id = $%d", *f.LocalID)
	}
	if f.Local != "" {
		add("l.nombre ILIKE '%%' || $%d || '%%'", f.Local)
	}
	return conditions, args
}

type servicioMaterializado struct {
	ServicioID    *int
	ServicioTexto string
	Tiempo        *string
	Costo         *float64
	Sesiones      int
	SesionNumero  int
	Orden         int
}

func materializarServiciosTx(tx *sqlx.Tx, inputs []repository.ComboServicioCatalogoInput) ([]servicioMaterializado, error) {
	ids := make([]int, 0, len(inputs))
	vistos := map[int]bool{}
	for _, input := range inputs {
		if input.ServicioID != nil && !vistos[*input.ServicioID] {
			vistos[*input.ServicioID] = true
			ids = append(ids, *input.ServicioID)
		}
	}
	type catalogo struct {
		ID     int      `db:"id"`
		Nombre string   `db:"nombre"`
		Tiempo *string  `db:"tiempo"`
		Costo  *float64 `db:"costo"`
	}
	porID := map[int]catalogo{}
	if len(ids) > 0 {
		var encontrados []catalogo
		if err := tx.Select(&encontrados, `SELECT id,nombre,tiempo,costo FROM servicios WHERE id=ANY($1) AND activo=TRUE`, ids); err != nil {
			return nil, fmt.Errorf("error al validar servicios: %w", err)
		}
		for _, s := range encontrados {
			porID[s.ID] = s
		}
	}
	resultado := make([]servicioMaterializado, 0, len(inputs))
	for _, input := range inputs {
		item := servicioMaterializado{ServicioID: input.ServicioID, ServicioTexto: input.ServicioTexto, Tiempo: input.Tiempo, Costo: input.Costo, Sesiones: input.Sesiones, SesionNumero: input.SesionNumero, Orden: input.Orden}
		if input.ServicioID != nil {
			servicio, ok := porID[*input.ServicioID]
			if !ok {
				return nil, fmt.Errorf("%w: servicio con id %d no encontrado o inactivo", repository.ErrComboNoEncontrado, *input.ServicioID)
			}
			if strings.TrimSpace(item.ServicioTexto) == "" {
				item.ServicioTexto = servicio.Nombre
			}
			if item.Tiempo == nil {
				item.Tiempo = servicio.Tiempo
			}
			if item.Costo == nil {
				item.Costo = servicio.Costo
			}
		}
		if strings.TrimSpace(item.ServicioTexto) == "" {
			return nil, fmt.Errorf("servicio_texto no puede quedar vacio")
		}
		resultado = append(resultado, item)
	}
	return resultado, nil
}

func validarPrecioPorItems(tipoPrecio string, servicios []servicioMaterializado) error {
	if tipoPrecio != "POR_ITEMS" {
		return nil
	}
	for _, servicio := range servicios {
		if servicio.Costo == nil {
			return fmt.Errorf("%w: cada servicio debe tener costo para POR_ITEMS", repository.ErrComboDatosInvalidos)
		}
	}
	return nil
}

func resumenServicios(servicios []servicioMaterializado) (float64, int) {
	var precio float64
	var sesiones int
	for _, servicio := range servicios {
		if servicio.Costo != nil {
			precio += *servicio.Costo * float64(servicio.Sesiones)
		}
		sesiones += servicio.Sesiones
	}
	return precio, sesiones
}

func insertarServiciosTx(tx *sqlx.Tx, comboID int, servicios []servicioMaterializado) error {
	if len(servicios) == 0 {
		return nil
	}
	args := make([]interface{}, 0, len(servicios)*8)
	for _, servicio := range servicios {
		args = append(args, comboID, servicio.ServicioID, servicio.ServicioTexto, pointerString(servicio.Tiempo), servicio.Costo, servicio.Sesiones, servicio.SesionNumero, servicio.Orden)
	}
	q := `INSERT INTO combo_servicios (combo_id,servicio_id,servicio_texto,tiempo,costo,sesiones,sesion_numero,orden) VALUES ` + batchValuesPlaceholders(len(servicios), 8)
	if _, err := tx.Exec(q, args...); err != nil {
		return fmt.Errorf("error al insertar servicios de combo: %w", err)
	}
	return nil
}

func actualizarResumenTx(tx *sqlx.Tx, comboID int) error {
	var resumen struct {
		TipoPrecio    string   `db:"tipo_precio"`
		PrecioPaquete *float64 `db:"precio_paquete"`
		PrecioItems   float64  `db:"precio_items"`
		Sesiones      int      `db:"sesiones"`
	}
	if err := tx.Get(&resumen, `
		SELECT cb.tipo_precio, cb.precio_paquete,
			COALESCE(SUM(cs.costo * cs.sesiones) FILTER (WHERE cs.activo = TRUE), 0) AS precio_items,
			COALESCE((SELECT COUNT(DISTINCT cs.sesion_numero) FROM combo_servicios cs WHERE cs.combo_id = cb.id AND cs.activo = TRUE), 1) AS sesiones
		FROM combos cb LEFT JOIN combo_servicios cs ON cs.combo_id = cb.id
		WHERE cb.id = $1 GROUP BY cb.id
	`, comboID); err != nil {
		return fmt.Errorf("error al calcular resumen de combo: %w", err)
	}
	precioFinal := resumen.PrecioItems
	if resumen.TipoPrecio == "PRECIO_PAQUETE" {
		if resumen.PrecioPaquete == nil {
			return fmt.Errorf("precio_paquete es requerido para PRECIO_PAQUETE")
		}
		precioFinal = *resumen.PrecioPaquete
	}
	// sesiones_totales se deriva de la cantidad de sesiones distintas en las líneas activas.
	if _, err := tx.Exec(`UPDATE combos SET costo_total = $1, sesiones_totales = $2, actualizado_en = NOW() WHERE id = $3`, precioFinal, resumen.Sesiones, comboID); err != nil {
		return fmt.Errorf("error al actualizar resumen de combo: %w", err)
	}
	return nil
}

func validarComboExisteTx(tx *sqlx.Tx, comboID int) error {
	var existe bool
	if err := tx.Get(&existe, `SELECT EXISTS(SELECT 1 FROM combos WHERE id = $1)`, comboID); err != nil {
		return err
	}
	if !existe {
		return fmt.Errorf("%w: combo con id %d", repository.ErrComboNoEncontrado, comboID)
	}
	return nil
}

func validarCategoriaTx(tx *sqlx.Tx, categoriaID *int) error {
	if categoriaID == nil {
		return nil
	}
	var existe bool
	if err := tx.Get(&existe, `SELECT EXISTS(SELECT 1 FROM categorias WHERE id = $1)`, *categoriaID); err != nil {
		return err
	}
	if !existe {
		return fmt.Errorf("%w: categoria con id %d", repository.ErrComboNoEncontrado, *categoriaID)
	}
	return nil
}

func validarLocalesTx(tx *sqlx.Tx, localIDs []int) error {
	if len(localIDs) == 0 {
		return nil
	}
	var encontrados []int
	if err := tx.Select(&encontrados, `SELECT id FROM locales WHERE id=ANY($1) AND activo=TRUE`, localIDs); err != nil {
		return fmt.Errorf("error al validar locales: %w", err)
	}
	set := map[int]bool{}
	for _, id := range encontrados {
		set[id] = true
	}
	for _, id := range localIDs {
		if !set[id] {
			return fmt.Errorf("%w: local con id %d no encontrado o inactivo", repository.ErrComboNoEncontrado, id)
		}
	}
	return nil
}

func insertarLocalesTx(tx *sqlx.Tx, comboID int, localIDs []int) error {
	if len(localIDs) == 0 {
		return nil
	}
	args := make([]interface{}, 0, len(localIDs)*2)
	for _, id := range localIDs {
		args = append(args, comboID, id)
	}
	if _, err := tx.Exec(`INSERT INTO combo_local (combo_id,local_id) VALUES `+batchValuesPlaceholders(len(localIDs), 2), args...); err != nil {
		return fmt.Errorf("error al asociar locales con combo: %w", err)
	}
	return nil
}

func pointerString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// contarSesiones cuenta las sesiones distintas presentes en las líneas del combo.
func contarSesiones(servicios []servicioMaterializado) int {
	set := map[int]bool{}
	for _, s := range servicios {
		n := s.SesionNumero
		if n < 1 {
			n = 1
		}
		set[n] = true
	}
	if len(set) == 0 {
		return 1
	}
	return len(set)
}
