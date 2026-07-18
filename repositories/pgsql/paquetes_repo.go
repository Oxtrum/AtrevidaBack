package pgsql

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

var _ repository.PaquetesRepository = (*PaquetesRepo)(nil)

type PaquetesRepo struct {
	db *sqlx.DB
}

func NewPaquetesRepo(db *sqlx.DB) *PaquetesRepo {
	return &PaquetesRepo{db: db}
}

func (r *PaquetesRepo) ListPaquetes(f repository.FiltroPaquetes) ([]models.PaqueteDetalle, error) {
	conditions, args := paqueteConditions(f)
	where := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT DISTINCT p.id, p.nombre, p.descripcion, p.categoria_id,
			c.nombre AS categoria,
			p.imagen_path, p.moneda, p.activo
		FROM paquetes p
		LEFT JOIN categorias c ON c.id = p.categoria_id
		LEFT JOIN paquete_local pl ON pl.paquete_id = p.id
		LEFT JOIN locales l ON l.id = pl.local_id
		WHERE %s
		ORDER BY p.nombre, p.id
	`, where)

	var paquetes []models.PaquetePG
	if err := r.db.Select(&paquetes, query, args...); err != nil {
		return nil, fmt.Errorf("error al listar paquetes: %w", err)
	}

	resultado := make([]models.PaqueteDetalle, 0, len(paquetes))
	for _, p := range paquetes {
		detalle, err := r.cargarDetallePaquete(p)
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, detalle)
	}
	return resultado, nil
}

func (r *PaquetesRepo) GetPaqueteByID(id int, incluirInactivo bool) (*models.PaqueteDetalle, error) {
	condition := "p.id = $1"
	if !incluirInactivo {
		condition += " AND p.activo = TRUE"
	}
	var paquete models.PaquetePG
	err := r.db.Get(&paquete, fmt.Sprintf(`
		SELECT p.id, p.nombre, p.descripcion, p.categoria_id,
			c.nombre AS categoria,
			p.imagen_path, p.moneda, p.activo
		FROM paquetes p
		LEFT JOIN categorias c ON c.id = p.categoria_id
		WHERE %s
	`, condition), id)
	if err != nil {
		return nil, fmt.Errorf("%w: paquete con id %d", repository.ErrPaqueteNoEncontrado, id)
	}
	detalle, err := r.cargarDetallePaquete(paquete)
	if err != nil {
		return nil, err
	}
	return &detalle, nil
}

func (r *PaquetesRepo) CrearPaquete(in repository.CrearPaqueteInput) (int, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, fmt.Errorf("error al iniciar transaccion de paquete: %w", err)
	}
	defer tx.Rollback()

	if err := validarCategoriaTx(tx, in.CategoriaID); err != nil {
		return 0, err
	}
	if err := validarLocalesTx(tx, in.LocalIDs); err != nil {
		return 0, err
	}

	var paqueteID int
	err = tx.QueryRowx(`
		INSERT INTO paquetes (nombre, descripcion, categoria_id, imagen_path, moneda, activo)
		VALUES ($1,$2,$3,$4,$5,TRUE)
		RETURNING id
	`, in.Nombre, nullStr(pointerString(in.Descripcion)), in.CategoriaID, nil, in.Moneda).Scan(&paqueteID)
	if err != nil {
		return 0, fmt.Errorf("error al crear paquete: %w", err)
	}

	if err := insertarServiciosBaseTx(tx, paqueteID, in.ServiciosBase); err != nil {
		return 0, err
	}
	if err := insertarLocalesPaqueteTx(tx, paqueteID, in.LocalIDs); err != nil {
		return 0, err
	}
	if err := materializarTiers(tx, paqueteID, in.Tiers); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("error al confirmar paquete: %w", err)
	}
	return paqueteID, nil
}

func (r *PaquetesRepo) ActualizarPaquete(in repository.ActualizarPaqueteInput) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var existe bool
	if err := tx.Get(&existe, `SELECT EXISTS(SELECT 1 FROM paquetes WHERE id = $1)`, in.ID); err != nil {
		return err
	}
	if !existe {
		return fmt.Errorf("%w: paquete con id %d", repository.ErrPaqueteNoEncontrado, in.ID)
	}

	if err := validarCategoriaTx(tx, in.CategoriaID); err != nil {
		return err
	}
	if err := validarLocalesTx(tx, in.LocalIDs); err != nil {
		return err
	}

	res, err := tx.Exec(`
		UPDATE paquetes SET nombre = $1, descripcion = $2, categoria_id = $3, moneda = $4, actualizado_en = NOW()
		WHERE id = $5
	`, in.Nombre, nullStr(pointerString(in.Descripcion)), in.CategoriaID, in.Moneda, in.ID)
	if err != nil {
		return fmt.Errorf("error al actualizar paquete: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("%w: paquete con id %d", repository.ErrPaqueteNoEncontrado, in.ID)
	}

	if _, err := tx.Exec(`UPDATE paquete_servicios SET activo = FALSE WHERE paquete_id = $1 AND activo = TRUE`, in.ID); err != nil {
		return fmt.Errorf("error al desactivar servicios base anteriores: %w", err)
	}
	if err := insertarServiciosBaseTx(tx, in.ID, in.ServiciosBase); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM paquete_local WHERE paquete_id = $1`, in.ID); err != nil {
		return fmt.Errorf("error al reemplazar locales de paquete: %w", err)
	}
	if err := insertarLocalesPaqueteTx(tx, in.ID, in.LocalIDs); err != nil {
		return err
	}

	if err := materializarTiers(tx, in.ID, in.Tiers); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PaquetesRepo) EliminarPaquete(id int) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`UPDATE paquetes SET activo = FALSE, actualizado_en = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("error al eliminar paquete: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("%w: paquete con id %d", repository.ErrPaqueteNoEncontrado, id)
	}
	if _, err := tx.Exec(`UPDATE combos SET activo = FALSE, actualizado_en = NOW() WHERE paquete_id = $1`, id); err != nil {
		return fmt.Errorf("error al eliminar tiers de paquete: %w", err)
	}
	return tx.Commit()
}

func (r *PaquetesRepo) SetPaqueteImagen(id int, path *string) error {
	result, err := r.db.Exec(`UPDATE paquetes SET imagen_path = $1, actualizado_en = NOW() WHERE id = $2`, path, id)
	if err != nil {
		return fmt.Errorf("error al actualizar imagen de paquete: %w", err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("%w: paquete con id %d", repository.ErrPaqueteNoEncontrado, id)
	}
	return nil
}

// cargarDetallePaquete arma el detalle de un paquete ya cargado: su catalogo
// base de servicios, sus locales y sus tiers (combos materializados activos).
func (r *PaquetesRepo) cargarDetallePaquete(paquete models.PaquetePG) (models.PaqueteDetalle, error) {
	detalle := models.PaqueteDetalle{Paquete: paquete}

	var servicios []models.PaqueteServicioPG
	if err := r.db.Select(&servicios, `
		SELECT id, paquete_id, servicio_id, servicio_texto, costo, orden
		FROM paquete_servicios
		WHERE paquete_id = $1 AND activo = TRUE
		ORDER BY orden, id
	`, paquete.ID); err != nil {
		return detalle, fmt.Errorf("error al cargar servicios base de paquete: %w", err)
	}
	if servicios == nil {
		servicios = []models.PaqueteServicioPG{}
	}
	detalle.ServiciosBase = servicios

	var locales []models.LocalPG
	if err := r.db.Select(&locales, `
		SELECT l.id, l.nombre, l.activo
		FROM locales l JOIN paquete_local pl ON pl.local_id = l.id
		WHERE pl.paquete_id = $1 ORDER BY l.nombre
	`, paquete.ID); err != nil {
		return detalle, fmt.Errorf("error al cargar locales de paquete: %w", err)
	}
	if locales == nil {
		locales = []models.LocalPG{}
	}
	detalle.Locales = locales

	var tiers []models.ComboCatalogoPG
	if err := r.db.Select(&tiers, `
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
		WHERE cb.paquete_id = $1 AND cb.activo = TRUE
		ORDER BY cb.sesiones_totales
	`, paquete.ID); err != nil {
		return detalle, fmt.Errorf("error al cargar tiers de paquete: %w", err)
	}
	for i := range tiers {
		// Locales/servicios del tier son un espejo materializado de los del
		// paquete y de servicios_base x sesiones; no se recargan aqui.
		tiers[i].Locales = []models.LocalPG{}
		tiers[i].Servicios = []models.ComboServicioDetallePG{}
	}
	if tiers == nil {
		tiers = []models.ComboCatalogoPG{}
	}
	detalle.Tiers = tiers

	return detalle, nil
}

func paqueteConditions(f repository.FiltroPaquetes) ([]string, []interface{}) {
	conditions := []string{"1 = 1"}
	args := []interface{}{}
	index := 1
	add := func(condition string, value interface{}) {
		conditions = append(conditions, fmt.Sprintf(condition, index))
		args = append(args, value)
		index++
	}
	if f.Activo != nil {
		add("p.activo = $%d", *f.Activo)
	}
	if f.Nombre != "" {
		add("p.nombre ILIKE '%%' || $%d || '%%'", f.Nombre)
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

// insertarServiciosBaseTx resuelve y valida cada linea del catalogo base,
// igual que combos.materializarServiciosTx: si trae servicio_id, lo busca en
// servicios (debe existir y estar activo) y copia nombre/costo cuando la
// linea no los trae; si no trae servicio_id, exige servicio_texto no vacio.
func insertarServiciosBaseTx(tx *sqlx.Tx, paqueteID int, servicios []repository.PaqueteServicioInput) error {
	for _, s := range servicios {
		servicioTexto := strings.TrimSpace(pointerString(s.ServicioTexto))
		costo := s.Costo
		if s.ServicioID != nil {
			var servicio struct {
				Nombre string   `db:"nombre"`
				Costo  *float64 `db:"costo"`
			}
			if err := tx.Get(&servicio, `SELECT nombre, costo FROM servicios WHERE id = $1 AND activo = TRUE`, *s.ServicioID); err != nil {
				return fmt.Errorf("%w: servicio con id %d no encontrado o inactivo", repository.ErrComboNoEncontrado, *s.ServicioID)
			}
			if servicioTexto == "" {
				servicioTexto = servicio.Nombre
			}
			if costo == 0 && servicio.Costo != nil {
				costo = *servicio.Costo
			}
		}
		if servicioTexto == "" {
			return fmt.Errorf("servicio_texto no puede quedar vacio")
		}
		if _, err := tx.Exec(`
			INSERT INTO paquete_servicios (paquete_id, servicio_id, servicio_texto, costo, orden, activo)
			VALUES ($1,$2,$3,$4,$5,TRUE)
		`, paqueteID, s.ServicioID, nullStr(servicioTexto), costo, s.Orden); err != nil {
			return fmt.Errorf("error al insertar servicio base de paquete: %w", err)
		}
	}
	return nil
}

func insertarLocalesPaqueteTx(tx *sqlx.Tx, paqueteID int, localIDs []int) error {
	for _, localID := range localIDs {
		if _, err := tx.Exec(`INSERT INTO paquete_local (paquete_id, local_id) VALUES ($1,$2)`, paqueteID, localID); err != nil {
			return fmt.Errorf("error al asociar local con paquete: %w", err)
		}
	}
	return nil
}

type paqueteServicioBase struct {
	ServicioID    *int    `db:"servicio_id"`
	ServicioTexto *string `db:"servicio_texto"`
	Costo         float64 `db:"costo"`
	Orden         int     `db:"orden"`
}

// materializarTiers es el corazon del modulo: por cada tier del input,
// upsertea la fila `combos` correspondiente (nombre derivado, precios,
// sesiones_totales, costo_total) y regenera su `combo_servicios` (una fila
// por servicio base x sesion) y su `combo_local` (copiado de paquete_local).
// Los tiers que existian en BD para el paquete pero ya no vienen en el input
// se desactivan (soft-delete), nunca se borran.
func materializarTiers(tx *sqlx.Tx, paqueteID int, tiers []models.PaqueteTierInput) error {
	var paquete struct {
		Nombre      string  `db:"nombre"`
		Moneda      string  `db:"moneda"`
		ImagenPath  *string `db:"imagen_path"`
		CategoriaID *int    `db:"categoria_id"`
	}
	if err := tx.Get(&paquete, `SELECT nombre, moneda, imagen_path, categoria_id FROM paquetes WHERE id = $1`, paqueteID); err != nil {
		return fmt.Errorf("%w: paquete con id %d", repository.ErrPaqueteNoEncontrado, paqueteID)
	}

	var base []paqueteServicioBase
	if err := tx.Select(&base, `
		SELECT servicio_id, servicio_texto, costo, orden
		FROM paquete_servicios
		WHERE paquete_id = $1 AND activo = TRUE
		ORDER BY orden, id
	`, paqueteID); err != nil {
		return fmt.Errorf("error al cargar servicios base para materializar tiers: %w", err)
	}
	var costoBase float64
	for _, b := range base {
		costoBase += b.Costo
	}

	vistos := map[int]bool{}
	for _, tier := range tiers {
		comboID, err := upsertTierComboTx(tx, paqueteID, paquete.Nombre, paquete.Moneda, paquete.ImagenPath, paquete.CategoriaID, costoBase, tier)
		if err != nil {
			return err
		}
		vistos[comboID] = true

		if err := regenerarServiciosTierTx(tx, comboID, base, tier.Sesiones); err != nil {
			return err
		}
		if err := regenerarLocalesTierTx(tx, comboID, paqueteID); err != nil {
			return err
		}
	}

	var existentes []int
	if err := tx.Select(&existentes, `SELECT id FROM combos WHERE paquete_id = $1 AND activo = TRUE`, paqueteID); err != nil {
		return fmt.Errorf("error al listar tiers existentes de paquete: %w", err)
	}
	for _, id := range existentes {
		if vistos[id] {
			continue
		}
		if _, err := tx.Exec(`UPDATE combos SET activo = FALSE, actualizado_en = NOW() WHERE id = $1`, id); err != nil {
			return fmt.Errorf("error al desactivar tier obsoleto: %w", err)
		}
	}
	return nil
}

func upsertTierComboTx(tx *sqlx.Tx, paqueteID int, paqueteNombre, moneda string, imagenPath *string, categoriaID *int, costoBase float64, tier models.PaqueteTierInput) (int, error) {
	label := "SESIONES"
	if tier.Sesiones == 1 {
		label = "SESIÓN"
	}
	nombre := fmt.Sprintf("%s - %d %s", paqueteNombre, tier.Sesiones, label)
	costoTotal := costoBase * float64(tier.Sesiones)

	if tier.ID != nil {
		res, err := tx.Exec(`
			UPDATE combos SET
				nombre = $1, tipo_precio = 'PRECIO_PAQUETE', precio_paquete = $2,
				precio_regular = $3, nota = $4, moneda = $5, sesiones_totales = $6, imagen_path = $7,
				categoria_id = $8, costo_total = $9, activo = TRUE, actualizado_en = NOW()
			WHERE id = $10 AND paquete_id = $11
		`, nombre, tier.PrecioContado, tier.PrecioRegular, nullStr(pointerString(tier.Nota)),
			moneda, tier.Sesiones, imagenPath, categoriaID, costoTotal, *tier.ID, paqueteID)
		if err != nil {
			return 0, fmt.Errorf("error al actualizar tier de paquete: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return 0, fmt.Errorf("%w: tier con id %d para paquete %d", repository.ErrPaqueteNoEncontrado, *tier.ID, paqueteID)
		}
		return *tier.ID, nil
	}

	var comboID int
	err := tx.QueryRowx(`
		INSERT INTO combos (
			nombre, paquete_id, tipo_precio, precio_paquete, precio_regular, nota,
			moneda, sesiones_totales, imagen_path, categoria_id, costo_total, activo
		) VALUES ($1,$2,'PRECIO_PAQUETE',$3,$4,$5,$6,$7,$8,$9,$10,TRUE)
		RETURNING id
	`, nombre, paqueteID, tier.PrecioContado, tier.PrecioRegular, nullStr(pointerString(tier.Nota)),
		moneda, tier.Sesiones, imagenPath, categoriaID, costoTotal).Scan(&comboID)
	if err != nil {
		return 0, fmt.Errorf("error al crear tier de paquete: %w", err)
	}
	return comboID, nil
}

func regenerarServiciosTierTx(tx *sqlx.Tx, comboID int, base []paqueteServicioBase, sesiones int) error {
	if _, err := tx.Exec(`DELETE FROM combo_servicios WHERE combo_id = $1`, comboID); err != nil {
		return fmt.Errorf("error al limpiar servicios de tier: %w", err)
	}
	orden := 1
	for n := 1; n <= sesiones; n++ {
		for _, b := range base {
			if _, err := tx.Exec(`
				INSERT INTO combo_servicios (combo_id, servicio_id, servicio_texto, costo, sesiones, sesion_numero, orden, activo)
				VALUES ($1,$2,$3,$4,1,$5,$6,TRUE)
			`, comboID, b.ServicioID, b.ServicioTexto, b.Costo, n, orden); err != nil {
				return fmt.Errorf("error al insertar servicio de tier: %w", err)
			}
			orden++
		}
	}
	return nil
}

func regenerarLocalesTierTx(tx *sqlx.Tx, comboID, paqueteID int) error {
	if _, err := tx.Exec(`DELETE FROM combo_local WHERE combo_id = $1`, comboID); err != nil {
		return fmt.Errorf("error al limpiar locales de tier: %w", err)
	}
	if _, err := tx.Exec(`
		INSERT INTO combo_local (combo_id, local_id)
		SELECT $1, local_id FROM paquete_local WHERE paquete_id = $2
	`, comboID, paqueteID); err != nil {
		return fmt.Errorf("error al copiar locales de tier: %w", err)
	}
	return nil
}
