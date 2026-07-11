package pgsql

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

var _ repository.PlanesRepository = (*PlanesRepo)(nil)

type PlanesRepo struct {
	db *sqlx.DB
}

func NewPlanesRepo(db *sqlx.DB) *PlanesRepo {
	return &PlanesRepo{db: db}
}

func (r *PlanesRepo) ListPlanes(f repository.FiltroPlanes) ([]models.PlanPG, error) {
	conditions := []string{"1 = 1"}
	args := []interface{}{}
	idx := 1

	if f.Cliente != "" {
		conditions = append(conditions, fmt.Sprintf("p.cliente_nombre_snapshot ILIKE $%d", idx))
		args = append(args, "%"+f.Cliente+"%")
		idx++
	}
	if f.ClienteID != nil {
		conditions = append(conditions, fmt.Sprintf("p.cliente_id = $%d", idx))
		args = append(args, *f.ClienteID)
		idx++
	}
	if f.LocalID != nil {
		conditions = append(conditions, fmt.Sprintf("p.local_id = $%d", idx))
		args = append(args, *f.LocalID)
		idx++
	}
	if f.Local != "" {
		conditions = append(conditions, fmt.Sprintf("p.local_nombre_snapshot ILIKE $%d", idx))
		args = append(args, "%"+f.Local+"%")
		idx++
	}
	if f.Estado != "" {
		conditions = append(conditions, fmt.Sprintf("p.estado = $%d", idx))
		args = append(args, f.Estado)
		idx++
	}
	if f.EstadoCobranza != "" {
		conditions = append(conditions, fmt.Sprintf("p.estado_cobranza = $%d", idx))
		args = append(args, f.EstadoCobranza)
		idx++
	}
	if f.FechaDesde != nil {
		conditions = append(conditions, fmt.Sprintf("p.creado_en >= $%d", idx))
		args = append(args, *f.FechaDesde)
		idx++
	}
	if f.FechaHasta != nil {
		conditions = append(conditions, fmt.Sprintf("p.creado_en <= $%d", idx))
		args = append(args, *f.FechaHasta)
		idx++
	}

	query := fmt.Sprintf(`
		SELECT
			p.id, p.codigo, p.cliente, p.local_id, p.combo_id, p.combo_nombre,
			p.sesiones_totales, p.sesiones_usadas, p.costo_total, p.notas,
			p.activo, p.creado_en,
			p.cliente_id, p.cliente_nombre_snapshot, p.local_nombre_snapshot,
			p.combo_id_origen, p.combo_nombre_snapshot,
			p.fecha_inicio, p.fecha_fin,
			p.estado, p.estado_cobranza, p.tipo_pago,
			p.subtotal, p.descuento, p.precio_total, p.moneda,
			p.creado_por, p.actualizado_por, p.actualizado_en
		FROM planes p
		WHERE %s
		ORDER BY p.creado_en DESC, p.id DESC
	`, strings.Join(conditions, " AND "))

	var planes []models.PlanPG
	if err := r.db.Select(&planes, query, args...); err != nil {
		return nil, fmt.Errorf("error al listar planes: %w", err)
	}
	if planes == nil {
		planes = []models.PlanPG{}
	}
	return planes, nil
}

func (r *PlanesRepo) GetPlanByID(id int) (*models.PlanCompletoPG, error) {
	var plan models.PlanCompletoPG
	err := r.db.Get(&plan, `
		SELECT
			p.id, p.codigo, p.cliente, p.local_id, p.combo_id, p.combo_nombre,
			p.sesiones_totales, p.sesiones_usadas, p.costo_total, p.notas,
			p.activo, p.creado_en,
			p.cliente_id, p.cliente_nombre_snapshot, p.local_nombre_snapshot,
			p.combo_id_origen, p.combo_nombre_snapshot,
			p.fecha_inicio, p.fecha_fin,
			p.estado, p.estado_cobranza, p.tipo_pago,
			p.subtotal, p.descuento, p.precio_total, p.moneda,
			p.creado_por, p.actualizado_por, p.actualizado_en
		FROM planes p
		WHERE p.id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("%w: plan con id %d", repository.ErrPlanNoEncontrado, id)
	}

	servicios, err := r.cargarServicios(id)
	if err != nil {
		return nil, err
	}
	plan.Servicios = servicios

	cuotas, err := r.cargarCuotas(id)
	if err != nil {
		return nil, err
	}
	plan.Cuotas = cuotas

	pagos, err := r.cargarAplicaciones(id)
	if err != nil {
		return nil, err
	}
	plan.Pagos = pagos

	return &plan, nil
}

func (r *PlanesRepo) CreatePlan(input repository.CrearPlanInput) (int, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, fmt.Errorf("error al iniciar transaccion de plan: %w", err)
	}
	defer tx.Rollback()

	var clienteNombre string
	if err := tx.Get(&clienteNombre, `
		SELECT COALESCE(NULLIF(BTRIM(CONCAT(nombre, ' ', COALESCE(apellido, ''))), ''), 'SIN NOMBRE')
		FROM clientes WHERE id = $1
	`, input.ClienteID); err != nil {
		return 0, fmt.Errorf("cliente con id %d no encontrado: %w", input.ClienteID, repository.ErrPlanDatosInvalidos)
	}

	var localNombre string
	if err := tx.Get(&localNombre, `SELECT nombre FROM locales WHERE id = $1 AND activo = TRUE`, input.LocalID); err != nil {
		return 0, fmt.Errorf("local con id %d no encontrado o inactivo: %w", input.LocalID, repository.ErrPlanDatosInvalidos)
	}

	var planID int
	err = tx.QueryRowx(`
		INSERT INTO planes (
			cliente, local_id, cliente_id, cliente_nombre_snapshot,
			local_nombre_snapshot, combo_id_origen, combo_nombre_snapshot,
			fecha_inicio, fecha_fin, estado, tipo_pago,
			subtotal, descuento, precio_total, moneda, notas, activo, creado_por
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,TRUE,$17
		) RETURNING id
	`,
		clienteNombre, input.LocalID, input.ClienteID, clienteNombre,
		localNombre, input.ComboIDOrigen, input.ComboNombreSnapshot,
		input.FechaInicio, input.FechaFin, input.Estado, input.TipoPago,
		input.Subtotal, input.Descuento, input.PrecioTotal, input.Moneda,
		nullStr(pointerString(input.Notas)), input.CreadoPor,
	).Scan(&planID)
	if err != nil {
		return 0, fmt.Errorf("error al crear plan: %w", err)
	}

	for _, s := range input.Servicios {
		if _, err := tx.Exec(`
			INSERT INTO plan_servicios (
				plan_id, servicio_id_origen, nombre_snapshot, tiempo_snapshot,
				precio_unitario_snapshot, sesiones_contratadas, orden
			) VALUES ($1,$2,$3,$4,$5,$6,$7)
		`, planID, s.ServicioIDOrigen, s.NombreSnapshot, pointerString(s.TiempoSnapshot),
			s.PrecioUnitarioSnapshot, s.SesionesContratadas, s.Orden); err != nil {
			return 0, fmt.Errorf("error al insertar servicio del plan: %w", err)
		}
	}

	for _, c := range input.Cuotas {
		if _, err := tx.Exec(`
			INSERT INTO plan_cuotas (plan_id, numero, vencimiento, monto, estado)
			VALUES ($1,$2,$3,$4,'PENDIENTE')
		`, planID, c.Numero, nullStr(pointerString(c.Vencimiento)), c.Monto); err != nil {
			return 0, fmt.Errorf("error al insertar cuota del plan: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("error al confirmar plan: %w", err)
	}
	return planID, nil
}

func (r *PlanesRepo) UpdatePlan(input repository.ActualizarPlanInput) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var estado string
	if err := tx.Get(&estado, `SELECT estado FROM planes WHERE id = $1`, input.ID); err != nil {
		return fmt.Errorf("%w: plan con id %d", repository.ErrPlanNoEncontrado, input.ID)
	}
	if estado != "BORRADOR" {
		return fmt.Errorf("%w: solo se puede modificar un plan en BORRADOR", repository.ErrPlanEstadoBloqueado)
	}

	sets := []string{"actualizado_en = NOW()"}
	args := []interface{}{}
	index := 1
	add := func(column string, value interface{}) {
		sets = append(sets, fmt.Sprintf("%s = $%d", column, index))
		args = append(args, value)
		index++
	}

	if input.Notas != nil {
		add("notas", nullStr(*input.Notas))
	}
	if input.FechaInicio != nil {
		add("fecha_inicio", *input.FechaInicio)
	}
	if input.FechaFin != nil {
		add("fecha_fin", *input.FechaFin)
	}

	args = append(args, input.ID)
	query := fmt.Sprintf("UPDATE planes SET %s WHERE id = $%d", strings.Join(sets, ", "), index)
	if _, err := tx.Exec(query, args...); err != nil {
		return fmt.Errorf("error al actualizar plan: %w", err)
	}
	return tx.Commit()
}

func (r *PlanesRepo) UpdatePlanEstado(id int, estado string, estadoCobranza string, usuarioID *int) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var actual struct {
		Estado string `db:"estado"`
	}
	if err := tx.Get(&actual, `SELECT estado FROM planes WHERE id = $1`, id); err != nil {
		return fmt.Errorf("%w: plan con id %d", repository.ErrPlanNoEncontrado, id)
	}

	if !esTransicionValida(actual.Estado, estado) {
		return fmt.Errorf("%w: de %s a %s", repository.ErrPlanTransicionInvalida, actual.Estado, estado)
	}

	if _, err := tx.Exec(`
		UPDATE planes SET estado = $1, estado_cobranza = $2, actualizado_por = $3, actualizado_en = NOW()
		WHERE id = $4
	`, estado, estadoCobranza, usuarioID, id); err != nil {
		return fmt.Errorf("error al actualizar estado del plan: %w", err)
	}

	return tx.Commit()
}

func esTransicionValida(actual, nuevo string) bool {
	switch actual {
	case "BORRADOR":
		return nuevo == "ACTIVO" || nuevo == "CANCELADO"
	case "ACTIVO":
		return nuevo == "COMPLETADO" || nuevo == "CANCELADO"
	case "COMPLETADO", "CANCELADO", "VENCIDO":
		return false
	default:
		return false
	}
}

func (r *PlanesRepo) cargarServicios(planID int) ([]models.PlanServicioPG, error) {
	var servicios []models.PlanServicioPG
	if err := r.db.Select(&servicios, `
		SELECT id, plan_id, servicio_id_origen, nombre_snapshot, tiempo_snapshot,
			precio_unitario_snapshot, sesiones_contratadas, orden
		FROM plan_servicios
		WHERE plan_id = $1
		ORDER BY orden, id
	`, planID); err != nil {
		return nil, fmt.Errorf("error al cargar servicios del plan: %w", err)
	}
	if servicios == nil {
		servicios = []models.PlanServicioPG{}
	}
	return servicios, nil
}

func (r *PlanesRepo) cargarCuotas(planID int) ([]models.PlanCuotaPG, error) {
	var cuotas []models.PlanCuotaPG
	if err := r.db.Select(&cuotas, `
		SELECT id, plan_id, numero, vencimiento::text, monto, estado, creado_en
		FROM plan_cuotas
		WHERE plan_id = $1
		ORDER BY numero
	`, planID); err != nil {
		return nil, fmt.Errorf("error al cargar cuotas del plan: %w", err)
	}
	if cuotas == nil {
		cuotas = []models.PlanCuotaPG{}
	}
	return cuotas, nil
}

func (r *PlanesRepo) cargarAplicaciones(planID int) ([]models.PlanPagoAplicacionPG, error) {
	var aplicaciones []models.PlanPagoAplicacionPG
	if err := r.db.Select(&aplicaciones, `
		SELECT ppa.id, ppa.plan_cuota_id, ppa.pago_id, ppa.monto_aplicado, ppa.creado_en
		FROM plan_pago_aplicaciones ppa
		JOIN plan_cuotas pc ON pc.id = ppa.plan_cuota_id
		WHERE pc.plan_id = $1
		ORDER BY ppa.creado_en
	`, planID); err != nil {
		return nil, fmt.Errorf("error al cargar aplicaciones de pago del plan: %w", err)
	}
	if aplicaciones == nil {
		aplicaciones = []models.PlanPagoAplicacionPG{}
	}
	return aplicaciones, nil
}
