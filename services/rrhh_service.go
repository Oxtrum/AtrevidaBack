package services

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

const (
	planillaSueldo    = "SUELDO"
	planillaAguinaldo = "AGUINALDO"
	// diasMinAguinaldo es la antiguedad minima dentro del anio para tener derecho a aguinaldo.
	diasMinAguinaldo = 90
)

type RRHHService struct {
	repo   *pgsqlrepo.RRHHRepo
	config *ConfiguracionService
}

func NewRRHHService(repo *pgsqlrepo.RRHHRepo, config *ConfiguracionService) *RRHHService {
	return &RRHHService{repo: repo, config: config}
}

func diaPagoEfectivo(e models.Empleado, cfg ConfigTrastienda) int {
	if e.DiaPago != nil {
		return *e.DiaPago
	}
	return cfg.DiaPagoDefecto
}

// ─── Empleados ────────────────────────────────────────────────────────────────

func (s *RRHHService) ListarEmpleados(soloActivos bool) ([]models.Empleado, error) {
	emps, err := s.repo.ListEmpleados(soloActivos)
	if err != nil {
		return nil, err
	}
	cfg, err := s.config.Snapshot()
	if err != nil {
		return nil, err
	}
	for i := range emps {
		emps[i].DiaPagoEfectivo = diaPagoEfectivo(emps[i], cfg)
	}
	return emps, nil
}

func (s *RRHHService) ObtenerEmpleado(id int) (*models.Empleado, error) {
	e, err := s.repo.GetEmpleado(id)
	if err != nil {
		return nil, err
	}
	cfg, err := s.config.Snapshot()
	if err != nil {
		return nil, err
	}
	e.DiaPagoEfectivo = diaPagoEfectivo(*e, cfg)
	return e, nil
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
	AplicaAportes *bool
	MetodoPago    *string
	FechaIngreso  string
	FechaBaja     *string
	Activo        *bool
	Notas         *string
}

func (s *RRHHService) validarEmpleado(in EmpleadoInput) (pgsqlrepo.EmpleadoInput, error) {
	out := pgsqlrepo.EmpleadoInput{}
	nombre, apellido := strings.TrimSpace(in.Nombre), strings.TrimSpace(in.Apellido)
	if nombre == "" || len(nombre) > 100 {
		return out, errors.New("nombre es requerido y no puede superar 100 caracteres")
	}
	if apellido == "" || len(apellido) > 100 {
		return out, errors.New("apellido es requerido y no puede superar 100 caracteres")
	}
	if in.SueldoBase <= 0 {
		return out, errors.New("sueldo_base debe ser mayor a 0")
	}
	if in.DiaPago != nil && (*in.DiaPago < 1 || *in.DiaPago > 31) {
		return out, errors.New("dia_pago debe estar entre 1 y 31")
	}
	ingreso, err := parseFecha(in.FechaIngreso)
	if err != nil {
		return out, errors.New("fecha_ingreso invalida, use formato YYYY-MM-DD")
	}
	var baja *time.Time
	if in.FechaBaja != nil && strings.TrimSpace(*in.FechaBaja) != "" {
		b, err := parseFecha(*in.FechaBaja)
		if err != nil {
			return out, errors.New("fecha_baja invalida, use formato YYYY-MM-DD")
		}
		if b.Before(ingreso) {
			return out, errors.New("fecha_baja no puede ser anterior a fecha_ingreso")
		}
		baja = &b
	}
	metodo, err := validarMetodoPago(in.MetodoPago)
	if err != nil {
		return out, err
	}
	aplica, activo := true, true
	if in.AplicaAportes != nil {
		aplica = *in.AplicaAportes
	}
	if in.Activo != nil {
		activo = *in.Activo
	}
	return pgsqlrepo.EmpleadoInput{
		Nombre: nombre, Apellido: apellido, CI: trimPtr(in.CI), Cargo: trimPtr(in.Cargo), Telefono: trimPtr(in.Telefono),
		LocalID: in.LocalID, UsuarioID: in.UsuarioID, SueldoBase: redondearMonto(in.SueldoBase), DiaPago: in.DiaPago,
		AplicaAportes: aplica, MetodoPago: metodo, FechaIngreso: ingreso, FechaBaja: baja, Activo: activo,
		Notas: trimPtr(in.Notas),
	}, nil
}

func (s *RRHHService) CrearEmpleado(in EmpleadoInput) (*models.Empleado, error) {
	val, err := s.validarEmpleado(in)
	if err != nil {
		return nil, err
	}
	id, err := s.repo.CreateEmpleado(val)
	if err != nil {
		return nil, err
	}
	return s.ObtenerEmpleado(id)
}

// EmpleadoPatch: los campos nullable usan *Set para distinguir "no vino" de "vino null".
type EmpleadoPatch struct {
	Nombre        *string
	Apellido      *string
	CI            *string
	CISet         bool
	Cargo         *string
	CargoSet      bool
	Telefono      *string
	TelefonoSet   bool
	LocalID       *int
	LocalSet      bool
	UsuarioID     *int
	UsuarioSet    bool
	SueldoBase    *float64
	DiaPago       *int
	DiaPagoSet    bool
	AplicaAportes *bool
	MetodoPago    *string
	MetodoSet     bool
	FechaIngreso  *string
	FechaBaja     *string
	FechaBajaSet  bool
	Activo        *bool
	Notas         *string
	NotasSet      bool
}

// ActualizarEmpleado edita la ficha. Solo afecta planillas futuras aun no generadas.
func (s *RRHHService) ActualizarEmpleado(id int, p EmpleadoPatch) (*models.Empleado, error) {
	actual, err := s.repo.GetEmpleado(id)
	if err != nil {
		return nil, err
	}
	in := EmpleadoInput{
		Nombre: actual.Nombre, Apellido: actual.Apellido, CI: actual.CI, Cargo: actual.Cargo, Telefono: actual.Telefono,
		LocalID: actual.LocalID, UsuarioID: actual.UsuarioID, SueldoBase: actual.SueldoBase, DiaPago: actual.DiaPago,
		AplicaAportes: &actual.AplicaAportes, MetodoPago: actual.MetodoPago, FechaIngreso: actual.FechaIngreso,
		FechaBaja: actual.FechaBaja, Activo: &actual.Activo, Notas: actual.Notas,
	}
	if p.Nombre != nil {
		in.Nombre = *p.Nombre
	}
	if p.Apellido != nil {
		in.Apellido = *p.Apellido
	}
	if p.CISet {
		in.CI = p.CI
	}
	if p.CargoSet {
		in.Cargo = p.Cargo
	}
	if p.TelefonoSet {
		in.Telefono = p.Telefono
	}
	if p.LocalSet {
		in.LocalID = p.LocalID
	}
	if p.UsuarioSet {
		in.UsuarioID = p.UsuarioID
	}
	if p.SueldoBase != nil {
		in.SueldoBase = *p.SueldoBase
	}
	if p.DiaPagoSet {
		in.DiaPago = p.DiaPago
	}
	if p.AplicaAportes != nil {
		in.AplicaAportes = p.AplicaAportes
	}
	if p.MetodoSet {
		in.MetodoPago = p.MetodoPago
	}
	if p.FechaIngreso != nil {
		in.FechaIngreso = *p.FechaIngreso
	}
	if p.FechaBajaSet {
		in.FechaBaja = p.FechaBaja
	}
	if p.Activo != nil {
		in.Activo = p.Activo
	}
	if p.NotasSet {
		in.Notas = p.Notas
	}
	val, err := s.validarEmpleado(in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateEmpleado(id, val); err != nil {
		return nil, err
	}
	return s.ObtenerEmpleado(id)
}

// ─── Adelantos ────────────────────────────────────────────────────────────────

func (s *RRHHService) ListarAdelantos(empleadoID *int, soloPendientes bool) ([]models.Adelanto, error) {
	return s.repo.ListAdelantos(empleadoID, soloPendientes)
}

// RegistrarAdelanto anota un anticipo que se descontara automaticamente en la siguiente planilla.
func (s *RRHHService) RegistrarAdelanto(empleadoID int, fecha string, monto float64, motivo *string) error {
	if monto <= 0 {
		return errors.New("monto debe ser mayor a 0")
	}
	f, err := parseFecha(fecha)
	if err != nil {
		return errors.New("fecha invalida, use formato YYYY-MM-DD")
	}
	emp, err := s.repo.GetEmpleado(empleadoID)
	if err != nil {
		return err
	}
	if !emp.Activo {
		return errors.New("el empleado esta inactivo")
	}
	_, err = s.repo.CreateAdelanto(empleadoID, f, redondearMonto(monto), trimPtr(motivo))
	return err
}

func (s *RRHHService) AnularAdelanto(id int) error { return s.repo.AnularAdelanto(id) }

// ─── Calculo ──────────────────────────────────────────────────────────────────

type novedadesPlanilla struct {
	Bonos           float64
	DiasFalta       float64
	OtrosDescuentos float64
}

// calcularSueldo arma el desglose de una planilla de sueldo. Los adelantos se consumen en orden mientras
// quepan en lo que queda tras aportes y descuentos, para que el neto nunca sea negativo.
func calcularSueldo(
	base float64, aplicaAportes bool, nov novedadesPlanilla, cfg ConfigTrastienda, adelantos []float64, exigirTodos bool,
) (pgsqlrepo.PlanillaValores, int, error) {
	v := pgsqlrepo.PlanillaValores{SueldoBase: base}
	if nov.Bonos < 0 || nov.DiasFalta < 0 || nov.OtrosDescuentos < 0 {
		return v, 0, errors.New("bonos, dias_falta y otros_descuentos no pueden ser negativos")
	}
	if nov.DiasFalta > float64(cfg.DiasMesFaltas) {
		return v, 0, fmt.Errorf("dias_falta no puede superar %d", cfg.DiasMesFaltas)
	}
	v.Bonos = redondearMonto(nov.Bonos)
	v.DiasFalta = math.Round(nov.DiasFalta*100) / 100
	v.OtrosDescuentos = redondearMonto(nov.OtrosDescuentos)
	v.DescuentoFaltas = redondearMonto(base / float64(cfg.DiasMesFaltas) * v.DiasFalta)
	v.TotalGanado = redondearMonto(base + v.Bonos - v.DescuentoFaltas)
	if v.TotalGanado <= 0 {
		return v, 0, errors.New("las faltas superan lo ganado: el total ganado debe ser mayor a 0")
	}
	if aplicaAportes {
		v.AporteLaboral = redondearMonto(v.TotalGanado * cfg.AporteLaboralPct / 100)
		v.AportePatronal = redondearMonto(v.TotalGanado * cfg.AportePatronalPct / 100)
	}
	capacidad := redondearMonto(v.TotalGanado - v.AporteLaboral - v.OtrosDescuentos)
	if capacidad < 0 {
		return v, 0, errors.New("los descuentos superan lo ganado: el neto a pagar no puede ser negativo")
	}
	usados := 0
	for _, m := range adelantos {
		if redondearMonto(v.Adelantos+m) > capacidad {
			break
		}
		v.Adelantos = redondearMonto(v.Adelantos + m)
		usados++
	}
	if exigirTodos && usados < len(adelantos) {
		return v, 0, errors.New("los descuentos superan lo ganado: el neto a pagar no puede ser negativo")
	}
	v.NetoPagar = redondearMonto(v.TotalGanado - v.AporteLaboral - v.Adelantos - v.OtrosDescuentos)
	v.CostoEmpresa = redondearMonto(v.TotalGanado + v.AportePatronal)
	return v, usados, nil
}

// ─── Planillas ────────────────────────────────────────────────────────────────

func parsePeriodo(valor string) (time.Time, error) {
	valor = strings.TrimSpace(valor)
	if t, err := time.Parse("2006-01", valor); err == nil {
		return t, nil
	}
	t, err := time.Parse("2006-01-02", valor)
	if err != nil {
		return time.Time{}, errors.New("periodo invalido, use formato YYYY-MM")
	}
	return primerDiaMes(t), nil
}

type FiltroPlanillas struct {
	Periodo    string
	EmpleadoID *int
	LocalID    *int
	Estado     string
	Tipo       string
}

func (s *RRHHService) ListarPlanillas(f FiltroPlanillas) ([]models.Planilla, error) {
	repoF := pgsqlrepo.FiltroPlanillas{EmpleadoID: f.EmpleadoID, LocalID: f.LocalID}
	if strings.TrimSpace(f.Periodo) != "" {
		p, err := parsePeriodo(f.Periodo)
		if err != nil {
			return nil, err
		}
		repoF.Periodo = &p
	}
	if f.Estado != "" {
		e := strings.ToUpper(strings.TrimSpace(f.Estado))
		if e != "PENDIENTE" && e != "PAGADO" && e != "ANULADO" {
			return nil, errors.New("estado invalido, valores permitidos: PENDIENTE, PAGADO, ANULADO")
		}
		repoF.Estado = e
	}
	if f.Tipo != "" {
		t := strings.ToUpper(strings.TrimSpace(f.Tipo))
		if t != planillaSueldo && t != planillaAguinaldo {
			return nil, errors.New("tipo invalido, valores permitidos: SUELDO, AGUINALDO")
		}
		repoF.Tipo = t
	}
	return s.repo.ListPlanillas(repoF)
}

func (s *RRHHService) ObtenerPlanilla(id int) (*models.Planilla, error) { return s.repo.GetPlanilla(id) }

// ResumenPeriodo suma las planillas no anuladas de un periodo.
func (s *RRHHService) ResumenPeriodo(planillas []models.Planilla, periodo string) models.PlanillaResumen {
	r := models.PlanillaResumen{Periodo: periodo}
	for _, p := range planillas {
		if p.Estado == "ANULADO" {
			continue
		}
		r.Empleados++
		r.TotalGanado += p.TotalGanado
		r.AporteLaboral += p.AporteLaboral
		r.Adelantos += p.Adelantos
		r.Descuentos += p.DescuentoFaltas + p.OtrosDescuentos
		r.NetoPagar += p.NetoPagar
		r.AportePatronal += p.AportePatronal
		r.CostoEmpresa += p.CostoEmpresa
		if p.Estado == "PAGADO" {
			r.Pagado += p.CostoEmpresa
		} else {
			r.Pendiente += p.CostoEmpresa
		}
	}
	r.TotalGanado, r.AporteLaboral, r.Adelantos = redondearMonto(r.TotalGanado), redondearMonto(r.AporteLaboral), redondearMonto(r.Adelantos)
	r.Descuentos, r.NetoPagar, r.AportePatronal = redondearMonto(r.Descuentos), redondearMonto(r.NetoPagar), redondearMonto(r.AportePatronal)
	r.CostoEmpresa, r.Pagado, r.Pendiente = redondearMonto(r.CostoEmpresa), redondearMonto(r.Pagado), redondearMonto(r.Pendiente)
	return r
}

type PlanillaPatch struct {
	Bonos           *float64
	DiasFalta       *float64
	OtrosDescuentos *float64
	Notas           *string
	NotasSet        bool
}

// ActualizarPlanilla cambia bonos, faltas u otros descuentos de una planilla PENDIENTE y recalcula todo,
// incluido el egreso asociado. Los adelantos ya descontados se mantienen.
func (s *RRHHService) ActualizarPlanilla(id int, p PlanillaPatch) (*models.Planilla, error) {
	actual, err := s.repo.GetPlanilla(id)
	if err != nil {
		return nil, err
	}
	if actual.Estado != "PENDIENTE" {
		return nil, errors.New("solo una planilla PENDIENTE puede editarse")
	}
	if actual.Tipo != planillaSueldo {
		return nil, errors.New("el aguinaldo no admite bonos ni descuentos: anula y regenera si hay un error")
	}
	emp, err := s.repo.GetEmpleado(actual.EmpleadoID)
	if err != nil {
		return nil, err
	}
	cfg, err := s.config.Snapshot()
	if err != nil {
		return nil, err
	}
	nov := novedadesPlanilla{Bonos: actual.Bonos, DiasFalta: actual.DiasFalta, OtrosDescuentos: actual.OtrosDescuentos}
	if p.Bonos != nil {
		nov.Bonos = *p.Bonos
	}
	if p.DiasFalta != nil {
		nov.DiasFalta = *p.DiasFalta
	}
	if p.OtrosDescuentos != nil {
		nov.OtrosDescuentos = *p.OtrosDescuentos
	}
	adelantos, err := s.repo.AdelantosDePlanilla(id)
	if err != nil {
		return nil, err
	}
	// Se respeta el sueldo con que nacio la planilla: editar la ficha del empleado no la altera.
	v, _, err := calcularSueldo(actual.SueldoBase, emp.AplicaAportes, nov, cfg, adelantos, true)
	if err != nil {
		return nil, err
	}
	notas := actual.Notas
	if p.NotasSet {
		notas = trimPtr(p.Notas)
	}
	if err := s.repo.ActualizarPlanilla(id, v, notas); err != nil {
		return nil, err
	}
	return s.repo.GetPlanilla(id)
}

// Pagar marca la planilla y su egreso como PAGADO.
func (s *RRHHService) Pagar(id int, metodo *string) (*models.Planilla, error) {
	m, err := validarMetodoPago(metodo)
	if err != nil {
		return nil, err
	}
	if err := s.repo.MarcarPlanillaPagada(id, m); err != nil {
		return nil, err
	}
	return s.repo.GetPlanilla(id)
}

// Anular exige motivo, libera los adelantos descontados y anula el egreso.
func (s *RRHHService) Anular(id int, motivo string) (*models.Planilla, error) {
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return nil, errors.New("motivo es requerido para anular")
	}
	if err := s.repo.AnularPlanilla(id, motivo); err != nil {
		return nil, err
	}
	return s.repo.GetPlanilla(id)
}

// ─── Generacion mensual ───────────────────────────────────────────────────────

func descripcionPlanilla(tipo string, periodo time.Time, e models.Empleado) string {
	if tipo == planillaAguinaldo {
		return fmt.Sprintf("Aguinaldo %d - %s %s", periodo.Year(), e.Nombre, e.Apellido)
	}
	return fmt.Sprintf("Sueldo %s - %s %s", periodo.Format("2006-01"), e.Nombre, e.Apellido)
}

// generarSueldo crea la planilla de sueldo del periodo para un empleado. Devuelve false si ya existia.
func (s *RRHHService) generarSueldo(e models.Empleado, periodo, fechaPago time.Time, cfg ConfigTrastienda) (bool, error) {
	return s.repo.GenerarPlanilla(pgsqlrepo.GenerarPlanillaInput{
		EmpleadoID: e.ID, Tipo: planillaSueldo, Periodo: periodo, FechaPago: fechaPago, EstadoMov: "PENDIENTE",
		Descripcion: descripcionPlanilla(planillaSueldo, periodo, e),
		Calcular: func(adelantos []float64) (pgsqlrepo.PlanillaValores, int, error) {
			return calcularSueldo(e.SueldoBase, e.AplicaAportes, novedadesPlanilla{}, cfg, adelantos, false)
		},
	})
}

// recorrerVencidas llama a fn por cada sueldo cuya fecha de pago ya llego (hasta `hoy`) y aun no tiene planilla,
// incluyendo meses atrasados. Lo comparten la generacion y su vista previa.
func (s *RRHHService) recorrerVencidas(hoy time.Time, cfg ConfigTrastienda, fn func(e models.Empleado, periodo, fechaPago time.Time) error) error {
	emps, err := s.repo.ListEmpleados(true)
	if err != nil {
		return err
	}
	for _, e := range emps {
		ingreso, err := parseFecha(e.FechaIngreso)
		if err != nil {
			continue
		}
		var baja *time.Time
		if e.FechaBaja != nil {
			if b, err := parseFecha(*e.FechaBaja); err == nil {
				baja = &b
			}
		}
		desde := primerDiaMes(ingreso)
		ultimo, err := s.repo.UltimoPeriodoSueldo(e.ID)
		if err != nil {
			return err
		}
		if ultimo != nil {
			if siguiente := primerDiaMes(*ultimo).AddDate(0, 1, 0); siguiente.After(desde) {
				desde = siguiente
			}
		}
		dia := diaPagoEfectivo(e, cfg)
		for periodo := desde; !periodo.After(hoy); periodo = periodo.AddDate(0, 1, 0) {
			fechaPago := fechaCobro(periodo, dia)
			if fechaPago.Before(ingreso) {
				continue // el primer mes puede pagarse antes de que el empleado ingrese
			}
			if fechaPago.After(hoy) || (baja != nil && fechaPago.After(*baja)) {
				break
			}
			if err := fn(e, periodo, fechaPago); err != nil {
				return err
			}
		}
	}
	return nil
}

// GenerarHasta crea las planillas de sueldo vencidas hasta `hoy` (catch-up). Es idempotente.
func (s *RRHHService) GenerarHasta(hoy time.Time) (int, error) {
	cfg, err := s.config.Snapshot()
	if err != nil {
		return 0, err
	}
	generadas := 0
	err = s.recorrerVencidas(hoy, cfg, func(e models.Empleado, periodo, fechaPago time.Time) error {
		creada, err := s.generarSueldo(e, periodo, fechaPago, cfg)
		if err != nil {
			return fmt.Errorf("planilla de %s %s: %w", e.Nombre, e.Apellido, err)
		}
		if creada {
			generadas++
		}
		return nil
	})
	return generadas, err
}

// GenerarHoy ejecuta GenerarHasta con la fecha actual en la zona horaria configurada.
func (s *RRHHService) GenerarHoy() (int, error) {
	cfg, err := s.config.Snapshot()
	if err != nil {
		return 0, err
	}
	return s.GenerarHasta(cfg.Hoy())
}

// PendientesDeGenerar devuelve, sin crear nada, las planillas que GenerarHoy generaria ahora.
func (s *RRHHService) PendientesDeGenerar() ([]models.PlanillaPendiente, error) {
	cfg, err := s.config.Snapshot()
	if err != nil {
		return nil, err
	}
	out := []models.PlanillaPendiente{}
	err = s.recorrerVencidas(cfg.Hoy(), cfg, func(e models.Empleado, periodo, fechaPago time.Time) error {
		out = append(out, models.PlanillaPendiente{
			EmpleadoID: e.ID, EmpleadoNombre: e.Nombre + " " + e.Apellido, Periodo: periodo.Format("2006-01"),
			FechaPago: fechaPago.Format("2006-01-02"), SueldoBase: e.SueldoBase,
		})
		return nil
	})
	return out, err
}

// GenerarPlanillaManual crea ya la planilla de un empleado para un periodo (p. ej. para cargar bonos o faltas antes
// del dia de pago, o regenerar una anulada). Devuelve false si ya existe una vigente.
func (s *RRHHService) GenerarPlanillaManual(empleadoID int, periodo string) (*models.Planilla, error) {
	p, err := parsePeriodo(periodo)
	if err != nil {
		return nil, err
	}
	e, err := s.repo.GetEmpleado(empleadoID)
	if err != nil {
		return nil, err
	}
	if !e.Activo {
		return nil, errors.New("el empleado esta inactivo")
	}
	cfg, err := s.config.Snapshot()
	if err != nil {
		return nil, err
	}
	if p.After(primerDiaMes(cfg.Hoy())) {
		return nil, errors.New("no se puede generar la planilla de un mes futuro")
	}
	if ingreso, err := parseFecha(e.FechaIngreso); err == nil && p.Before(primerDiaMes(ingreso)) {
		return nil, errors.New("el periodo es anterior al ingreso del empleado")
	}
	creada, err := s.generarSueldo(*e, p, fechaCobro(p, diaPagoEfectivo(*e, cfg)), cfg)
	if err != nil {
		return nil, err
	}
	if !creada {
		return nil, errors.New("ya existe una planilla vigente de ese empleado para ese periodo")
	}
	lista, err := s.repo.ListPlanillas(pgsqlrepo.FiltroPlanillas{Periodo: &p, EmpleadoID: &empleadoID, Tipo: planillaSueldo})
	if err != nil {
		return nil, err
	}
	for _, pl := range lista {
		if pl.Estado != "ANULADO" {
			return &pl, nil
		}
	}
	return nil, errPlanillaNoCreada
}

// errPlanillaNoCreada indica un estado inconsistente tras generar; no deberia ocurrir.
var errPlanillaNoCreada = errors.New("error al leer la planilla generada")

// GenerarAguinaldos crea el aguinaldo del anio para los empleados con al menos 90 dias en el anio:
// sueldo_base proporcional a los dias trabajados. No lleva aportes. Vence el 20 de diciembre.
func (s *RRHHService) GenerarAguinaldos(anio int) (int, error) {
	if anio < 2000 || anio > 2100 {
		return 0, errors.New("anio invalido")
	}
	cfg, err := s.config.Snapshot()
	if err != nil {
		return 0, err
	}
	if anio > cfg.Hoy().Year() {
		return 0, errors.New("no se puede generar el aguinaldo de un anio futuro")
	}
	emps, err := s.repo.ListEmpleados(false)
	if err != nil {
		return 0, err
	}
	inicioAnio := time.Date(anio, 1, 1, 0, 0, 0, 0, time.UTC)
	finAnio := time.Date(anio, 12, 31, 0, 0, 0, 0, time.UTC)
	diasAnio := float64(finAnio.YearDay())
	periodo := time.Date(anio, 12, 1, 0, 0, 0, 0, time.UTC)
	fechaPago := time.Date(anio, 12, 20, 0, 0, 0, 0, time.UTC)
	generadas := 0
	for _, e := range emps {
		desde, err := parseFecha(e.FechaIngreso)
		if err != nil {
			continue
		}
		if desde.Before(inicioAnio) {
			desde = inicioAnio
		}
		hasta := finAnio
		if e.FechaBaja != nil {
			if b, err := parseFecha(*e.FechaBaja); err == nil && b.Before(hasta) {
				hasta = b
			}
		}
		dias := int(hasta.Sub(desde).Hours()/24) + 1
		if dias < diasMinAguinaldo {
			continue
		}
		monto := redondearMonto(e.SueldoBase * math.Min(float64(dias), diasAnio) / diasAnio)
		emp := e
		creada, err := s.repo.GenerarPlanilla(pgsqlrepo.GenerarPlanillaInput{
			EmpleadoID: e.ID, Tipo: planillaAguinaldo, Periodo: periodo, FechaPago: fechaPago, EstadoMov: "PENDIENTE",
			Descripcion: descripcionPlanilla(planillaAguinaldo, periodo, emp),
			Calcular: func(_ []float64) (pgsqlrepo.PlanillaValores, int, error) {
				return pgsqlrepo.PlanillaValores{
					SueldoBase: emp.SueldoBase, TotalGanado: monto, NetoPagar: monto, CostoEmpresa: monto,
				}, 0, nil
			},
		})
		if err != nil {
			return generadas, fmt.Errorf("aguinaldo de %s %s: %w", e.Nombre, e.Apellido, err)
		}
		if creada {
			generadas++
		}
	}
	return generadas, nil
}
