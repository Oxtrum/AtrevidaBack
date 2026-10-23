package pgsql

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"atrevida-agenda-api/models"

	"github.com/jmoiron/sqlx"
)

var (
	ErrEmpleadoNoEnc = errors.New("empleado no encontrado")
	ErrPlanillaNoEnc = errors.New("planilla no encontrada")
	ErrAdelantoNoEnc = errors.New("adelanto no encontrado")
	ErrTipoBonoNoEnc = errors.New("tipo de bono no encontrado")
)

type RRHHRepo struct {
	db *sqlx.DB
}

func NewRRHHRepo(db *sqlx.DB) *RRHHRepo {
	return &RRHHRepo{db: db}
}

// ─── Empleados ────────────────────────────────────────────────────────────────

const empleadoSelect = `
	SELECT e.id, e.nombre, e.apellido, e.ci, e.cargo, e.telefono, e.local_id, l.nombre AS local_nombre, e.usuario_id,
	       e.sueldo_base, e.dia_pago::int AS dia_pago, e.aplica_aportes, e.metodo_pago,
	       e.fecha_ingreso::text AS fecha_ingreso, e.fecha_baja::text AS fecha_baja, e.activo, e.notas,
	       COALESCE((SELECT SUM(a.monto) FROM adelantos_empleado a
	                 WHERE a.empleado_id = e.id AND a.planilla_id IS NULL AND NOT a.anulado), 0) AS adelantos_pendientes
	FROM empleados e
	LEFT JOIN locales l ON l.id = e.local_id`

func (r *RRHHRepo) ListEmpleados(soloActivos bool) ([]models.Empleado, error) {
	cond := ""
	if soloActivos {
		cond = " WHERE e.activo = TRUE"
	}
	out := []models.Empleado{}
	if err := r.db.Select(&out, empleadoSelect+cond+" ORDER BY e.activo DESC, e.apellido, e.nombre"); err != nil {
		return nil, fmt.Errorf("error al listar empleados: %w", err)
	}
	return out, nil
}

func (r *RRHHRepo) GetEmpleado(id int) (*models.Empleado, error) {
	var e models.Empleado
	err := r.db.Get(&e, empleadoSelect+" WHERE e.id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEmpleadoNoEnc
	}
	if err != nil {
		return nil, fmt.Errorf("error al leer empleado: %w", err)
	}
	return &e, nil
}

type EmpleadoInput struct {
	Nombre        string
	Apellido      string
	CI            *string
	Cargo         *string
	Telefono      *string
	LocalID       *int
	UsuarioID     *int
	SueldoBase    float64
	DiaPago       *int
	AplicaAportes bool
	MetodoPago    *string
	FechaIngreso  time.Time
	FechaBaja     *time.Time
	Activo        bool
	Notas         *string
}

func (r *RRHHRepo) CreateEmpleado(in EmpleadoInput) (int, error) {
	var id int
	err := r.db.QueryRowx(`
		INSERT INTO empleados
			(nombre, apellido, ci, cargo, telefono, local_id, usuario_id, sueldo_base, dia_pago, aplica_aportes,
			 metodo_pago, fecha_ingreso, fecha_baja, activo, notas)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING id`,
		in.Nombre, in.Apellido, in.CI, in.Cargo, in.Telefono, in.LocalID, in.UsuarioID, in.SueldoBase, in.DiaPago,
		in.AplicaAportes, in.MetodoPago, in.FechaIngreso, in.FechaBaja, in.Activo, in.Notas).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("error al crear empleado: %w", err)
	}
	return id, nil
}

// UpdateEmpleado reemplaza todos los campos (el servicio arma el estado final). No toca planillas ya generadas.
func (r *RRHHRepo) UpdateEmpleado(id int, in EmpleadoInput) error {
	res, err := r.db.Exec(`
		UPDATE empleados SET nombre = $1, apellido = $2, ci = $3, cargo = $4, telefono = $5, local_id = $6,
			usuario_id = $7, sueldo_base = $8, dia_pago = $9, aplica_aportes = $10, metodo_pago = $11,
			fecha_ingreso = $12, fecha_baja = $13, activo = $14, notas = $15, fecha_modificacion = NOW()
		WHERE id = $16`,
		in.Nombre, in.Apellido, in.CI, in.Cargo, in.Telefono, in.LocalID, in.UsuarioID, in.SueldoBase, in.DiaPago,
		in.AplicaAportes, in.MetodoPago, in.FechaIngreso, in.FechaBaja, in.Activo, in.Notas, id)
	if err != nil {
		return fmt.Errorf("error al actualizar empleado: %w", err)
	}
	n, err := affectedRows(res, "actualizar empleado")
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrEmpleadoNoEnc
	}
	return nil
}

// ─── Adelantos ────────────────────────────────────────────────────────────────

const adelantoSelect = `
	SELECT a.id, a.empleado_id, e.nombre || ' ' || e.apellido AS empleado_nombre, a.fecha::text AS fecha, a.monto,
	       a.motivo, a.planilla_id, a.anulado
	FROM adelantos_empleado a
	JOIN empleados e ON e.id = a.empleado_id`

func (r *RRHHRepo) ListAdelantos(empleadoID *int, soloPendientes bool) ([]models.Adelanto, error) {
	conds := "TRUE"
	args := []interface{}{}
	if empleadoID != nil {
		args = append(args, *empleadoID)
		conds += fmt.Sprintf(" AND a.empleado_id = $%d", len(args))
	}
	if soloPendientes {
		conds += " AND a.planilla_id IS NULL AND NOT a.anulado"
	}
	out := []models.Adelanto{}
	if err := r.db.Select(&out, adelantoSelect+" WHERE "+conds+" ORDER BY a.fecha DESC, a.id DESC", args...); err != nil {
		return nil, fmt.Errorf("error al listar adelantos: %w", err)
	}
	return out, nil
}

func (r *RRHHRepo) CreateAdelanto(empleadoID int, fecha time.Time, monto float64, motivo *string) (int, error) {
	var id int
	err := r.db.QueryRowx(`
		INSERT INTO adelantos_empleado (empleado_id, fecha, monto, motivo) VALUES ($1,$2,$3,$4) RETURNING id`,
		empleadoID, fecha, monto, motivo).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("error al registrar adelanto: %w", err)
	}
	return id, nil
}

// AnularAdelanto solo aplica a adelantos que aun no se descontaron en una planilla.
func (r *RRHHRepo) AnularAdelanto(id int) error {
	res, err := r.db.Exec(
		`UPDATE adelantos_empleado SET anulado = TRUE WHERE id = $1 AND planilla_id IS NULL AND NOT anulado`, id)
	if err != nil {
		return fmt.Errorf("error al anular adelanto: %w", err)
	}
	n, err := affectedRows(res, "anular adelanto")
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	var existe bool
	if err := r.db.Get(&existe, `SELECT EXISTS(SELECT 1 FROM adelantos_empleado WHERE id = $1)`, id); err != nil {
		return fmt.Errorf("error al anular adelanto: %w", err)
	}
	if !existe {
		return ErrAdelantoNoEnc
	}
	return errors.New("el adelanto ya fue descontado en una planilla o esta anulado y no puede editarse")
}

// ─── Planillas ────────────────────────────────────────────────────────────────

const planillaSelect = `
	SELECT p.id, p.empleado_id, e.nombre || ' ' || e.apellido AS empleado_nombre, e.cargo, e.local_id,
	       l.nombre AS local_nombre, p.tipo, p.periodo::text AS periodo, p.fecha_pago::text AS fecha_pago,
	       p.sueldo_base, p.bonos, p.dias_falta, p.descuento_faltas, p.total_ganado, p.aporte_laboral, p.adelantos,
	       p.otros_descuentos, p.neto_pagar, p.aporte_patronal, p.costo_empresa, p.estado, p.motivo_anulacion,
	       p.notas, m.codigo AS movimiento_codigo
	FROM planillas p
	JOIN empleados e ON e.id = p.empleado_id
	LEFT JOIN locales l ON l.id = e.local_id
	LEFT JOIN movimientos_financieros m ON m.planilla_id = p.id`

type FiltroPlanillas struct {
	Periodo    *time.Time // primer dia del mes
	EmpleadoID *int
	LocalID    *int
	Estado     string
	Tipo       string
}

func (r *RRHHRepo) ListPlanillas(f FiltroPlanillas) ([]models.Planilla, error) {
	conds := "TRUE"
	args := []interface{}{}
	add := func(cond string, v interface{}) {
		args = append(args, v)
		conds += fmt.Sprintf(" AND "+cond, len(args))
	}
	if f.Periodo != nil {
		add("p.periodo = $%d", *f.Periodo)
	}
	if f.EmpleadoID != nil {
		add("p.empleado_id = $%d", *f.EmpleadoID)
	}
	if f.LocalID != nil {
		add("e.local_id = $%d", *f.LocalID)
	}
	if f.Estado != "" {
		add("p.estado = $%d", f.Estado)
	}
	if f.Tipo != "" {
		add("p.tipo = $%d", f.Tipo)
	}
	out := []models.Planilla{}
	if err := r.db.Select(&out, planillaSelect+" WHERE "+conds+" ORDER BY p.periodo DESC, e.apellido, e.nombre", args...); err != nil {
		return nil, fmt.Errorf("error al listar planillas: %w", err)
	}
	if err := r.adjuntarBonos(out); err != nil {
		return nil, err
	}
	return out, nil
}

// adjuntarBonos carga las lineas de bono de todas las planillas con una sola consulta.
func (r *RRHHRepo) adjuntarBonos(planillas []models.Planilla) error {
	if len(planillas) == 0 {
		return nil
	}
	ids := make([]int, len(planillas))
	for i := range planillas {
		ids[i] = planillas[i].ID
		planillas[i].BonosDetalle = []models.PlanillaBono{}
	}
	query, args, err := sqlx.In(`
		SELECT b.id, b.planilla_id, b.tipo_bono_id, t.nombre AS tipo_nombre, b.monto, b.nota
		FROM planilla_bonos b JOIN tipos_bono t ON t.id = b.tipo_bono_id
		WHERE b.planilla_id IN (?) ORDER BY b.id`, ids)
	if err != nil {
		return fmt.Errorf("error al leer bonos de planillas: %w", err)
	}
	var lineas []models.PlanillaBono
	if err := r.db.Select(&lineas, r.db.Rebind(query), args...); err != nil {
		return fmt.Errorf("error al leer bonos de planillas: %w", err)
	}
	porID := make(map[int]*models.Planilla, len(planillas))
	for i := range planillas {
		porID[planillas[i].ID] = &planillas[i]
	}
	for _, l := range lineas {
		p := porID[l.PlanillaID]
		p.BonosDetalle = append(p.BonosDetalle, l)
	}
	return nil
}

func (r *RRHHRepo) GetPlanilla(id int) (*models.Planilla, error) {
	var p models.Planilla
	err := r.db.Get(&p, planillaSelect+" WHERE p.id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPlanillaNoEnc
	}
	if err != nil {
		return nil, fmt.Errorf("error al leer planilla: %w", err)
	}
	one := []models.Planilla{p}
	if err := r.adjuntarBonos(one); err != nil {
		return nil, err
	}
	return &one[0], nil
}

// UltimoPeriodoSueldo devuelve el primer dia del ultimo mes con planilla de SUELDO (de cualquier estado), o nil.
func (r *RRHHRepo) UltimoPeriodoSueldo(empleadoID int) (*time.Time, error) {
	var periodo sql.NullTime
	if err := r.db.Get(&periodo,
		`SELECT MAX(periodo) FROM planillas WHERE empleado_id = $1 AND tipo = 'SUELDO'`, empleadoID); err != nil {
		return nil, fmt.Errorf("error al leer ultimo periodo de planilla: %w", err)
	}
	if !periodo.Valid {
		return nil, nil
	}
	return &periodo.Time, nil
}

// PlanillaValores es el desglose calculado por el servicio.
type PlanillaValores struct {
	SueldoBase      float64
	Bonos           float64
	DiasFalta       float64
	DescuentoFaltas float64
	TotalGanado     float64
	AporteLaboral   float64
	Adelantos       float64
	OtrosDescuentos float64
	NetoPagar       float64
	AportePatronal  float64
	CostoEmpresa    float64
}

// CalcularPlanilla recibe los montos de adelantos pendientes (mas antiguos primero) y devuelve el desglose
// junto con cuantos adelantos consume, contando desde el primero.
type CalcularPlanilla func(adelantos []float64) (PlanillaValores, int, error)

type GenerarPlanillaInput struct {
	EmpleadoID  int
	Tipo        string
	Periodo     time.Time
	FechaPago   time.Time
	EstadoMov   string
	Descripcion string
	Calcular    CalcularPlanilla
}

// GenerarPlanilla crea, en una sola transaccion, la planilla, el egreso FIJO "Sueldos y personal" por su costo
// para la empresa y el descuento de los adelantos pendientes. Devuelve false si ya existia (idempotente).
func (r *RRHHRepo) GenerarPlanilla(in GenerarPlanillaInput) (bool, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return false, fmt.Errorf("error al generar planilla: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var existe bool
	if err := tx.Get(&existe, `
		SELECT EXISTS(SELECT 1 FROM planillas WHERE empleado_id = $1 AND periodo = $2 AND tipo = $3 AND estado <> 'ANULADO')`,
		in.EmpleadoID, in.Periodo, in.Tipo); err != nil {
		return false, fmt.Errorf("error al generar planilla: %w", err)
	}
	if existe {
		return false, nil
	}

	var pend []struct {
		ID    int     `db:"id"`
		Monto float64 `db:"monto"`
	}
	if err := tx.Select(&pend, `
		SELECT id, monto FROM adelantos_empleado
		WHERE empleado_id = $1 AND planilla_id IS NULL AND NOT anulado AND fecha <= $2
		ORDER BY fecha, id FOR UPDATE`, in.EmpleadoID, in.FechaPago); err != nil {
		return false, fmt.Errorf("error al leer adelantos pendientes: %w", err)
	}
	montos := make([]float64, len(pend))
	for i, a := range pend {
		montos[i] = a.Monto
	}
	v, usados, err := in.Calcular(montos)
	if err != nil {
		return false, err
	}

	var planillaID int
	err = tx.QueryRowx(`
		INSERT INTO planillas
			(empleado_id, tipo, periodo, fecha_pago, sueldo_base, bonos, dias_falta, descuento_faltas, total_ganado,
			 aporte_laboral, adelantos, otros_descuentos, neto_pagar, aporte_patronal, costo_empresa)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (empleado_id, periodo, tipo) WHERE estado <> 'ANULADO' DO NOTHING
		RETURNING id`,
		in.EmpleadoID, in.Tipo, in.Periodo, in.FechaPago, v.SueldoBase, v.Bonos, v.DiasFalta, v.DescuentoFaltas,
		v.TotalGanado, v.AporteLaboral, v.Adelantos, v.OtrosDescuentos, v.NetoPagar, v.AportePatronal, v.CostoEmpresa,
	).Scan(&planillaID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("error al generar planilla: %w", err)
	}

	for i := 0; i < usados; i++ {
		if _, err := tx.Exec(`UPDATE adelantos_empleado SET planilla_id = $1 WHERE id = $2`, planillaID, pend[i].ID); err != nil {
			return false, fmt.Errorf("error al descontar adelanto: %w", err)
		}
	}

	if _, err := tx.Exec(`
		INSERT INTO movimientos_financieros
			(naturaleza, clase, categoria_id, planilla_id, periodo, local_id, local_nombre, fecha, monto, descripcion,
			 proveedor, metodo_pago, estado, nombre_usuario)
		SELECT 'EGRESO', 'FIJO', (SELECT id FROM categorias_financieras
		                          WHERE nombre = 'Sueldos y personal' AND activo AND naturaleza IN ('EGRESO','AMBOS') LIMIT 1),
		       $1, $2, e.local_id, l.nombre, $3, $4, $5, e.nombre || ' ' || e.apellido, e.metodo_pago, $6, 'sistema'
		FROM empleados e LEFT JOIN locales l ON l.id = e.local_id
		WHERE e.id = $7`,
		planillaID, in.Periodo, in.FechaPago, v.CostoEmpresa, in.Descripcion, in.EstadoMov, in.EmpleadoID); err != nil {
		return false, fmt.Errorf("error al generar el egreso de la planilla (verifica la categoria 'Sueldos y personal'): %w", err)
	}
	return true, tx.Commit()
}

// AdelantosDePlanilla devuelve los montos de adelantos ya descontados en la planilla.
func (r *RRHHRepo) AdelantosDePlanilla(planillaID int) ([]float64, error) {
	out := []float64{}
	if err := r.db.Select(&out,
		`SELECT monto FROM adelantos_empleado WHERE planilla_id = $1 ORDER BY fecha, id`, planillaID); err != nil {
		return nil, fmt.Errorf("error al leer adelantos de la planilla: %w", err)
	}
	return out, nil
}

// BonoLinea es una linea de bono a guardar en una planilla.
type BonoLinea struct {
	TipoBonoID int
	Monto      float64
	Nota       *string
}

// ActualizarPlanilla guarda el desglose recalculado de una planilla PENDIENTE y sincroniza el monto del egreso.
// Si bonos no es nil, reemplaza las lineas de bono de la planilla (nil las deja como estan).
func (r *RRHHRepo) ActualizarPlanilla(id int, v PlanillaValores, notas *string, bonos *[]BonoLinea) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("error al actualizar planilla: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(`
		UPDATE planillas SET bonos = $1, dias_falta = $2, descuento_faltas = $3, total_ganado = $4,
			aporte_laboral = $5, otros_descuentos = $6, neto_pagar = $7, aporte_patronal = $8, costo_empresa = $9,
			notas = $10, fecha_modificacion = NOW()
		WHERE id = $11 AND estado = 'PENDIENTE'`,
		v.Bonos, v.DiasFalta, v.DescuentoFaltas, v.TotalGanado, v.AporteLaboral, v.OtrosDescuentos, v.NetoPagar,
		v.AportePatronal, v.CostoEmpresa, notas, id)
	if err != nil {
		return fmt.Errorf("error al actualizar planilla: %w", err)
	}
	n, err := affectedRows(res, "actualizar planilla")
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("solo una planilla PENDIENTE puede editarse")
	}
	if bonos != nil {
		if _, err := tx.Exec(`DELETE FROM planilla_bonos WHERE planilla_id = $1`, id); err != nil {
			return fmt.Errorf("error al actualizar bonos de la planilla: %w", err)
		}
		for _, b := range *bonos {
			if _, err := tx.Exec(
				`INSERT INTO planilla_bonos (planilla_id, tipo_bono_id, monto, nota) VALUES ($1,$2,$3,$4)`,
				id, b.TipoBonoID, b.Monto, b.Nota); err != nil {
				return fmt.Errorf("error al guardar bono de la planilla: %w", err)
			}
		}
	}
	if _, err := tx.Exec(`
		UPDATE movimientos_financieros SET monto = $1, fecha_modificacion = NOW() WHERE planilla_id = $2`,
		v.CostoEmpresa, id); err != nil {
		return fmt.Errorf("error al sincronizar el egreso de la planilla: %w", err)
	}
	return tx.Commit()
}

// MarcarPlanillaPagada pasa la planilla y su egreso a PAGADO.
func (r *RRHHRepo) MarcarPlanillaPagada(id int, metodo *string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("error al pagar planilla: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(`
		UPDATE planillas SET estado = 'PAGADO', fecha_modificacion = NOW() WHERE id = $1 AND estado = 'PENDIENTE'`, id)
	if err != nil {
		return fmt.Errorf("error al pagar planilla: %w", err)
	}
	n, err := affectedRows(res, "pagar planilla")
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("solo una planilla PENDIENTE puede marcarse como pagada")
	}
	if _, err := tx.Exec(`
		UPDATE movimientos_financieros SET estado = 'PAGADO', metodo_pago = COALESCE($2, metodo_pago),
			fecha_modificacion = NOW()
		WHERE planilla_id = $1`, id, metodo); err != nil {
		return fmt.Errorf("error al pagar el egreso de la planilla: %w", err)
	}
	return tx.Commit()
}

// AnularPlanilla anula planilla y egreso, y libera los adelantos que descontaba para que vuelvan a quedar pendientes.
func (r *RRHHRepo) AnularPlanilla(id int, motivo string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("error al anular planilla: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(`
		UPDATE planillas SET estado = 'ANULADO', motivo_anulacion = $2, fecha_modificacion = NOW()
		WHERE id = $1 AND estado <> 'ANULADO'`, id, motivo)
	if err != nil {
		return fmt.Errorf("error al anular planilla: %w", err)
	}
	n, err := affectedRows(res, "anular planilla")
	if err != nil {
		return err
	}
	if n == 0 {
		var existe bool
		if err := tx.Get(&existe, `SELECT EXISTS(SELECT 1 FROM planillas WHERE id = $1)`, id); err != nil {
			return fmt.Errorf("error al anular planilla: %w", err)
		}
		if !existe {
			return ErrPlanillaNoEnc
		}
		return errors.New("la planilla ya esta anulada")
	}
	if _, err := tx.Exec(`
		UPDATE movimientos_financieros SET estado = 'ANULADO', motivo_anulacion = $2, fecha_modificacion = NOW()
		WHERE planilla_id = $1`, id, motivo); err != nil {
		return fmt.Errorf("error al anular el egreso de la planilla: %w", err)
	}
	if _, err := tx.Exec(`UPDATE adelantos_empleado SET planilla_id = NULL WHERE planilla_id = $1`, id); err != nil {
		return fmt.Errorf("error al liberar los adelantos de la planilla: %w", err)
	}
	return tx.Commit()
}

// PlanillaDeMovimiento indica si el movimiento pertenece a una planilla (se gestiona desde Recursos Humanos).
func (r *RRHHRepo) PlanillaDeMovimiento(movimientoID int) (bool, error) {
	var es bool
	if err := r.db.Get(&es,
		`SELECT EXISTS(SELECT 1 FROM movimientos_financieros WHERE id = $1 AND planilla_id IS NOT NULL)`,
		movimientoID); err != nil {
		return false, fmt.Errorf("error al leer movimiento: %w", err)
	}
	return es, nil
}

// ─── Tipos de bono ────────────────────────────────────────────────────────────

const tipoBonoSelect = `SELECT id, nombre, descripcion, activo FROM tipos_bono`

func (r *RRHHRepo) ListTiposBono(soloActivos bool) ([]models.TipoBono, error) {
	cond := ""
	if soloActivos {
		cond = " WHERE activo = TRUE"
	}
	out := []models.TipoBono{}
	if err := r.db.Select(&out, tipoBonoSelect+cond+" ORDER BY activo DESC, nombre"); err != nil {
		return nil, fmt.Errorf("error al listar tipos de bono: %w", err)
	}
	return out, nil
}

func (r *RRHHRepo) GetTipoBono(id int) (*models.TipoBono, error) {
	var t models.TipoBono
	err := r.db.Get(&t, tipoBonoSelect+" WHERE id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTipoBonoNoEnc
	}
	if err != nil {
		return nil, fmt.Errorf("error al leer tipo de bono: %w", err)
	}
	return &t, nil
}

// NombreTipoBonoEnUso indica si otro tipo (distinto de excluirID) ya usa ese nombre, sin distinguir mayusculas.
func (r *RRHHRepo) NombreTipoBonoEnUso(nombre string, excluirID int) (bool, error) {
	var en bool
	if err := r.db.Get(&en,
		`SELECT EXISTS(SELECT 1 FROM tipos_bono WHERE LOWER(nombre) = LOWER($1) AND id <> $2)`, nombre, excluirID); err != nil {
		return false, fmt.Errorf("error al validar nombre del tipo de bono: %w", err)
	}
	return en, nil
}

func (r *RRHHRepo) CreateTipoBono(nombre string, descripcion *string) (int, error) {
	var id int
	if err := r.db.QueryRowx(
		`INSERT INTO tipos_bono (nombre, descripcion) VALUES ($1,$2) RETURNING id`, nombre, descripcion).Scan(&id); err != nil {
		return 0, fmt.Errorf("error al crear tipo de bono: %w", err)
	}
	return id, nil
}

func (r *RRHHRepo) UpdateTipoBono(id int, nombre string, descripcion *string, activo bool) error {
	res, err := r.db.Exec(`
		UPDATE tipos_bono SET nombre = $1, descripcion = $2, activo = $3, fecha_modificacion = NOW() WHERE id = $4`,
		nombre, descripcion, activo, id)
	if err != nil {
		return fmt.Errorf("error al actualizar tipo de bono: %w", err)
	}
	n, err := affectedRows(res, "actualizar tipo de bono")
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTipoBonoNoEnc
	}
	return nil
}

// TiposBonoDePlanilla devuelve los ids de tipo ya usados en las lineas de una planilla.
func (r *RRHHRepo) TiposBonoDePlanilla(planillaID int) (map[int]bool, error) {
	ids := []int{}
	if err := r.db.Select(&ids, `SELECT DISTINCT tipo_bono_id FROM planilla_bonos WHERE planilla_id = $1`, planillaID); err != nil {
		return nil, fmt.Errorf("error al leer bonos de la planilla: %w", err)
	}
	out := make(map[int]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}
