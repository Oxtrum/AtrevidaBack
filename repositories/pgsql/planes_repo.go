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
