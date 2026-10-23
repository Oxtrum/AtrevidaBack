package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

type GastosFijosService struct {
	repo   *pgsqlrepo.FinanzasRepo
	config *ConfiguracionService
	fin    *FinanzasService
}

func NewGastosFijosService(repo *pgsqlrepo.FinanzasRepo, config *ConfiguracionService, fin *FinanzasService) *GastosFijosService {
	return &GastosFijosService{repo: repo, config: config, fin: fin}
}

func primerDiaMes(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func ultimoDiaMes(t time.Time) int {
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// fechaCobro es el dia de cobro dentro del mes del periodo; si el mes no tiene ese dia (31 en febrero)
// se usa el ultimo dia del mes (RN-31).
func fechaCobro(periodo time.Time, diaCobro int) time.Time {
	dia := diaCobro
	if ult := ultimoDiaMes(periodo); dia > ult {
		dia = ult
	}
	return time.Date(periodo.Year(), periodo.Month(), dia, 0, 0, 0, 0, time.UTC)
}

func parseFecha(valor string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(valor))
}

// Frecuencias de cobro de un gasto fijo.
const (
	FrecDiaria     = "DIARIA"
	FrecSemanal    = "SEMANAL"
	FrecQuincenal  = "QUINCENAL"
	FrecMensual    = "MENSUAL"
	FrecBimestral  = "BIMESTRAL"
	FrecTrimestral = "TRIMESTRAL"
	FrecSemestral  = "SEMESTRAL"
	FrecAnual      = "ANUAL"
)

// pasoFrecuencia devuelve cada cuanto se cobra: en meses (MENSUAL o mas) o en dias (DIARIA/SEMANAL/QUINCENAL).
func pasoFrecuencia(f string) (meses, dias int, ok bool) {
	switch f {
	case FrecDiaria:
		return 0, 1, true
	case FrecSemanal:
		return 0, 7, true
	case FrecQuincenal:
		return 0, 14, true
	case FrecMensual:
		return 1, 0, true
	case FrecBimestral:
		return 2, 0, true
	case FrecTrimestral:
		return 3, 0, true
	case FrecSemestral:
		return 6, 0, true
	case FrecAnual:
		return 12, 0, true
	}
	return 0, 0, false
}

// recorrerCobros recorre en orden las fechas de cobro de la plantilla desde `inicio` hasta que fn devuelva false.
// `periodo` es la clave unica del cobro: el primer dia del mes para frecuencias de mes en adelante y la propia
// fecha para las de dias. El llamador debe cortar (fn=false): la secuencia no termina sola.
func recorrerCobros(g models.GastoFijo, inicio time.Time, fn func(periodo, cobro time.Time) bool) {
	meses, dias, ok := pasoFrecuencia(g.Frecuencia)
	if !ok {
		meses = 1 // plantillas anteriores a la frecuencia
	}
	if dias > 0 {
		for c := inicio; ; c = c.AddDate(0, 0, dias) {
			if !fn(c, c) {
				return
			}
		}
	}
	base := primerDiaMes(inicio)
	for i := 0; ; i++ {
		periodo := base.AddDate(0, i*meses, 0)
		cobro := fechaCobro(periodo, g.DiaCobro)
		if cobro.Before(inicio) {
			continue // el primer periodo puede cobrarse antes de que la plantilla entre en vigencia
		}
		if !fn(periodo, cobro) {
			return
		}
	}
}

func vigencia(g models.GastoFijo) (inicio time.Time, fin *time.Time, ok bool) {
	inicio, err := parseFecha(g.FechaInicio)
	if err != nil {
		return inicio, nil, false
	}
	if g.FechaFin != nil {
		if f, err := parseFecha(*g.FechaFin); err == nil {
			fin = &f
		}
	}
	return inicio, fin, true
}

// cobroDesde devuelve la primera fecha de cobro dentro de la vigencia que es >= minimo (o > si estricto), o nil.
func cobroDesde(g models.GastoFijo, minimo time.Time, estricto bool) *time.Time {
	inicio, fin, ok := vigencia(g)
	if !ok {
		return nil
	}
	var out *time.Time
	recorrerCobros(g, inicio, func(_, cobro time.Time) bool {
		if fin != nil && cobro.After(*fin) {
			return false
		}
		if cobro.Before(minimo) || (estricto && cobro.Equal(minimo)) {
			return true
		}
		c := cobro
		out = &c
		return false
	})
	return out
}

// siguienteCobro es el cobro que toca pagar a continuacion: el primero desde hoy o, si ya se pago por adelantado
// mas alla de hoy, el primero despues de lo pagado. nil si no esta activo o la vigencia ya termino.
func siguienteCobro(g models.GastoFijo, hoy time.Time) *time.Time {
	if !g.Activo {
		return nil
	}
	if g.PagadoHasta != nil {
		if pagado, err := parseFecha(*g.PagadoHasta); err == nil && !pagado.Before(hoy) {
			return cobroDesde(g, pagado, true)
		}
	}
	return cobroDesde(g, hoy, false)
}

// ─── CRUD ─────────────────────────────────────────────────────────────────────

func (s *GastosFijosService) Listar() ([]models.GastoFijo, error) {
	cfg, err := s.config.Snapshot()
	if err != nil {
		return nil, err
	}
	hoy := cfg.Hoy()
	// Catch-up antes de leer: "pagado hasta" y "atrasados" son correctos aunque el job diario no haya corrido.
	if _, err := s.GenerarHasta(hoy); err != nil {
		return nil, err
	}
	gastos, err := s.repo.ListGastosFijos(false)
	if err != nil {
		return nil, err
	}
	atrasados, err := s.repo.AtrasadosPorGasto(hoy)
	if err != nil {
		return nil, err
	}
	for i := range gastos {
		gastos[i].Atrasados = atrasados[gastos[i].ID]
		if p := siguienteCobro(gastos[i], hoy); p != nil {
			v := p.Format("2006-01-02")
			gastos[i].ProximoCobro = &v
		}
	}
	return gastos, nil
}

type GastoFijoInput struct {
	Nombre      string
	CategoriaID int
	Monto       float64
	DiaCobro    int
	LocalID     *int
	MetodoPago  *string
	Proveedor   *string
	FechaInicio string
	FechaFin    *string
	Activo      *bool
	Notas       *string
	// Frecuencia: vacia = MENSUAL.
	Frecuencia string
}

func (s *GastosFijosService) validar(in GastoFijoInput) (pgsqlrepo.GastoFijoInput, error) {
	out := pgsqlrepo.GastoFijoInput{}
	nombre := strings.TrimSpace(in.Nombre)
	if nombre == "" || len(nombre) > 200 {
		return out, errors.New("nombre es requerido y no puede superar 200 caracteres")
	}
	if in.Monto <= 0 {
		return out, errors.New("monto debe ser mayor a 0")
	}
	frecuencia := strings.ToUpper(strings.TrimSpace(in.Frecuencia))
	if frecuencia == "" {
		frecuencia = FrecMensual
	}
	_, dias, okFrec := pasoFrecuencia(frecuencia)
	if !okFrec {
		return out, errors.New("frecuencia invalida, valores permitidos: DIARIA, SEMANAL, QUINCENAL, MENSUAL, BIMESTRAL, TRIMESTRAL, SEMESTRAL, ANUAL")
	}
	inicio, err := parseFecha(in.FechaInicio)
	if err != nil {
		return out, errors.New("fecha_inicio invalida, use formato YYYY-MM-DD")
	}
	diaCobro := in.DiaCobro
	if dias > 0 || diaCobro == 0 {
		// Sin dia de cobro explicito el dia sale de la fecha del primer pago; las frecuencias de dias cuentan
		// desde esa fecha y no usan dia.
		diaCobro = inicio.Day()
	} else if diaCobro < 1 || diaCobro > 31 {
		return out, errors.New("dia_cobro debe estar entre 1 y 31")
	}
	var fin *time.Time
	if in.FechaFin != nil && strings.TrimSpace(*in.FechaFin) != "" {
		f, err := parseFecha(*in.FechaFin)
		if err != nil {
			return out, errors.New("fecha_fin invalida, use formato YYYY-MM-DD")
		}
		if f.Before(inicio) {
			return out, errors.New("fecha_fin no puede ser anterior a fecha_inicio")
		}
		fin = &f
	}
	metodo, err := validarMetodoPago(in.MetodoPago)
	if err != nil {
		return out, err
	}
	if err := s.fin.categoriaCompatible(in.CategoriaID, "EGRESO"); err != nil {
		return out, err
	}
	activo := true
	if in.Activo != nil {
		activo = *in.Activo
	}
	return pgsqlrepo.GastoFijoInput{
		Nombre: nombre, CategoriaID: in.CategoriaID, Monto: redondearMonto(in.Monto), DiaCobro: diaCobro,
		LocalID: in.LocalID, MetodoPago: metodo, Proveedor: trimPtr(in.Proveedor), FechaInicio: inicio,
		FechaFin: fin, Activo: activo, Notas: trimPtr(in.Notas), Frecuencia: frecuencia,
	}, nil
}

func (s *GastosFijosService) Crear(in GastoFijoInput) (*models.GastoFijo, error) {
	val, err := s.validar(in)
	if err != nil {
		return nil, err
	}
	id, err := s.repo.CreateGastoFijo(val)
	if err != nil {
		return nil, err
	}
	return s.repo.GetGastoFijo(id)
}

// GastoFijoPatch: los campos nullable usan *Set para distinguir "no vino" de "vino null".
type GastoFijoPatch struct {
	Frecuencia   *string
	Nombre       *string
	CategoriaID  *int
	Monto        *float64
	DiaCobro     *int
	LocalID      *int
	LocalSet     bool
	MetodoPago   *string
	MetodoSet    bool
	Proveedor    *string
	ProveedorSet bool
	FechaInicio  *string
	FechaFin     *string
	FechaFinSet  bool
	Activo       *bool
	Notas        *string
	NotasSet     bool
}

// Actualizar edita la plantilla. Solo afecta periodos futuros aun no generados (RN-34).
func (s *GastosFijosService) Actualizar(id int, p GastoFijoPatch) error {
	actual, err := s.repo.GetGastoFijo(id)
	if err != nil {
		return err
	}
	in := GastoFijoInput{
		Nombre: actual.Nombre, CategoriaID: actual.CategoriaID, Monto: actual.Monto, DiaCobro: actual.DiaCobro,
		LocalID: actual.LocalID, MetodoPago: actual.MetodoPago, Proveedor: actual.Proveedor,
		FechaInicio: actual.FechaInicio, FechaFin: actual.FechaFin, Activo: &actual.Activo, Notas: actual.Notas,
		Frecuencia: actual.Frecuencia,
	}
	if p.Frecuencia != nil {
		in.Frecuencia = *p.Frecuencia
	}
	if p.Nombre != nil {
		in.Nombre = *p.Nombre
	}
	if p.CategoriaID != nil {
		in.CategoriaID = *p.CategoriaID
	}
	if p.Monto != nil {
		in.Monto = *p.Monto
	}
	if p.DiaCobro != nil {
		in.DiaCobro = *p.DiaCobro
	}
	if p.LocalSet {
		in.LocalID = p.LocalID
	}
	if p.MetodoSet {
		in.MetodoPago = p.MetodoPago
	}
	if p.ProveedorSet {
		in.Proveedor = p.Proveedor
	}
	if p.FechaInicio != nil {
		in.FechaInicio = *p.FechaInicio
	}
	if p.FechaFinSet {
		in.FechaFin = p.FechaFin
	}
	if p.Activo != nil {
		in.Activo = p.Activo
	}
	if p.NotasSet {
		in.Notas = p.Notas
	}
	val, err := s.validar(in)
	if err != nil {
		return err
	}
	return s.repo.UpdateGastoFijo(id, val)
}

// Desactivar da de baja la plantilla sin perder el historial generado.
func (s *GastosFijosService) Desactivar(id int) error {
	return s.repo.DesactivarGastoFijo(id)
}

// ─── Generacion mensual (RN-30..RN-35) ────────────────────────────────────────

// FiltroAdelanto acota la vista previa de pagos futuros: los proximos `Cobros` desde hoy y/o todos los que caen
// hasta la fecha `Hasta`; `GastoFijoID` limita a una sola plantilla. El valor cero no ofrece pagos futuros.
type FiltroAdelanto struct {
	Cobros      int
	Hasta       *time.Time
	GastoFijoID *int
}

// recorrerVencidos llama a fn por cada cobro de gasto fijo que ya vencio hasta `hoy` y aun no tiene egreso
// (incluye periodos atrasados) y, segun `adel`, tambien por los pagos futuros que todavia no existen
// (adelantado = true; los N proximos se cuentan sin los futuros que ya existen). Un cobro ya existente, pagado o anulado, nunca se vuelve a ofrecer.
// Lo comparten la generacion y su vista previa.
func (s *GastosFijosService) recorrerVencidos(hoy time.Time, adel FiltroAdelanto, fn func(g models.GastoFijo, periodo, cobro time.Time, adelantado bool) error) error {
	gastos, err := s.repo.ListGastosFijos(true)
	if err != nil {
		return err
	}
	for _, g := range gastos {
		if adel.GastoFijoID != nil && g.ID != *adel.GastoFijoID {
			continue
		}
		inicio, fin, ok := vigencia(g)
		if !ok {
			continue
		}
		existentes, err := s.repo.PeriodosGenerados(g.ID)
		if err != nil {
			return err
		}
		futuros := 0
		var errFn error
		recorrerCobros(g, inicio, func(periodo, cobro time.Time) bool {
			if fin != nil && cobro.After(*fin) {
				return false
			}
			adelantado := cobro.After(hoy)
			existe := existentes[periodo.Format("2006-01-02")]
			if adelantado {
				if adel.Cobros <= 0 && adel.Hasta == nil {
					return false
				}
				// Un futuro que ya existe (pagado o pendiente) no se ofrece ni cuenta para "proximos N":
				// los N se cuentan a partir del ultimo cobro cubierto.
				if existe {
					return true
				}
				futuros++
				dentro := futuros <= maxPagosFuturos && (futuros <= adel.Cobros || (adel.Hasta != nil && !cobro.After(*adel.Hasta)))
				if !dentro {
					return false
				}
			}
			if existe {
				return true
			}
			errFn = fn(g, periodo, cobro, adelantado)
			return errFn == nil
		})
		if errFn != nil {
			return errFn
		}
	}
	return nil
}

// GenerarHasta crea los egresos FIJO que ya vencieron hasta `hoy`, incluyendo periodos atrasados
// (catch-up). Es idempotente: el indice unico (gasto_fijo_id, periodo) impide duplicados (RN-32).
func (s *GastosFijosService) GenerarHasta(hoy time.Time) (int, error) {
	cfg, err := s.config.Snapshot()
	if err != nil {
		return 0, err
	}
	generados := 0
	err = s.recorrerVencidos(hoy, FiltroAdelanto{}, func(g models.GastoFijo, periodo, cobro time.Time, _ bool) error {
		creado, err := s.repo.GenerarMovimientoGastoFijo(g.ID, periodo, cobro, cfg.EstadoGastoFijoGen)
		if err != nil {
			return fmt.Errorf("gasto fijo %q: %w", g.Nombre, err)
		}
		if creado {
			generados++
		}
		return nil
	})
	return generados, err
}

// PendientesDeGenerar devuelve, sin crear nada, los egresos que GenerarHoy generaria ahora. Con `adel` incluye
// ademas los pagos futuros pedidos (proximos N y/o hasta una fecha), marcados como adelantados.
func (s *GastosFijosService) PendientesDeGenerar(adel FiltroAdelanto) ([]models.GastoFijoPendiente, error) {
	cfg, err := s.config.Snapshot()
	if err != nil {
		return nil, err
	}
	hoy := cfg.Hoy()
	if adel.Hasta != nil && adel.Hasta.Sub(hoy) > fechaMaxima {
		return nil, errors.New("la fecha esta demasiado lejos")
	}
	out := []models.GastoFijoPendiente{}
	err = s.recorrerVencidos(hoy, adel, func(g models.GastoFijo, periodo, cobro time.Time, adelantado bool) error {
		out = append(out, models.GastoFijoPendiente{
			GastoFijoID: g.ID, Nombre: g.Nombre, Categoria: g.CategoriaNombre,
			Periodo: periodo.Format("2006-01-02"), FechaCobro: cobro.Format("2006-01-02"),
			Monto: g.Monto, LocalNombre: g.LocalNombre, Adelantado: adelantado,
		})
		return nil
	})
	return out, err
}

// GenerarHoy ejecuta GenerarHasta con la fecha actual en la zona horaria configurada.
func (s *GastosFijosService) GenerarHoy() (int, error) {
	cfg, err := s.config.Snapshot()
	if err != nil {
		return 0, err
	}
	return s.GenerarHasta(cfg.Hoy())
}

// CobroPago identifica el cobro de un gasto fijo que se quiere pagar.
type CobroPago struct {
	GastoFijoID int
	FechaCobro  string
}

// Protecciones internas para que una fecha absurda (ej. el anio 9000) no deje al servidor iterando: no son un
// limite de negocio. Hasta 50 anios hacia adelante y 2000 pagos futuros por gasto en una sola vista previa.
const (
	fechaMaxima     = 50 * 366 * 24 * time.Hour
	maxPagosFuturos = 2000
)

// PagarCobros crea (si no existen) y paga en un solo paso los cobros indicados, vencidos o por adelantado. Cada
// fecha debe ser un cobro real de la plantilla; pagar dos veces el mismo cobro no duplica. Un cobro anulado no
// se revive.
func (s *GastosFijosService) PagarCobros(cobros []CobroPago, usuarioID *int, usuarioNombre string) (int, error) {
	if len(cobros) == 0 {
		return 0, errors.New("indica al menos un cobro para pagar")
	}
	cfg, err := s.config.Snapshot()
	if err != nil {
		return 0, err
	}
	hoy := cfg.Hoy()
	lista := make([]pgsqlrepo.CobroPagar, 0, len(cobros))
	for _, c := range cobros {
		g, err := s.repo.GetGastoFijo(c.GastoFijoID)
		if err != nil {
			return 0, err
		}
		if !g.Activo {
			return 0, fmt.Errorf("el gasto fijo %q esta inactivo", g.Nombre)
		}
		fecha, err := parseFecha(c.FechaCobro)
		if err != nil {
			return 0, errors.New("fecha_cobro invalida, use formato YYYY-MM-DD")
		}
		if fecha.Sub(hoy) > fechaMaxima {
			return 0, errors.New("la fecha esta demasiado lejos")
		}
		inicio, fin, ok := vigencia(*g)
		if !ok {
			return 0, fmt.Errorf("el gasto fijo %q tiene una fecha de inicio invalida", g.Nombre)
		}
		var periodo *time.Time
		recorrerCobros(*g, inicio, func(p, cobro time.Time) bool {
			if fin != nil && cobro.After(*fin) {
				return false
			}
			if cobro.Equal(fecha) {
				pp := p
				periodo = &pp
				return false
			}
			return cobro.Before(fecha)
		})
		if periodo == nil {
			return 0, fmt.Errorf("el %s no es un cobro valido de %q", c.FechaCobro, g.Nombre)
		}
		lista = append(lista, pgsqlrepo.CobroPagar{GastoFijoID: g.ID, Periodo: *periodo, Fecha: fecha})
	}
	if err := s.exigirAlDia(lista, hoy); err != nil {
		return 0, err
	}
	if err := s.repo.PagarCobrosGastoFijo(lista, usuarioID, usuarioNombre); err != nil {
		return 0, err
	}
	return len(lista), nil
}

// exigirAlDia impide adelantar un gasto que tiene pagos atrasados: para pagar algo futuro, todos los vencidos del
// gasto deben estar ya pagados o ir en la misma solicitud.
func (s *GastosFijosService) exigirAlDia(cobros []pgsqlrepo.CobroPagar, hoy time.Time) error {
	adelantan := map[int]bool{}
	pagan := map[int]map[string]bool{}
	for _, c := range cobros {
		if c.Fecha.After(hoy) {
			adelantan[c.GastoFijoID] = true
		}
		if pagan[c.GastoFijoID] == nil {
			pagan[c.GastoFijoID] = map[string]bool{}
		}
		pagan[c.GastoFijoID][c.Fecha.Format("2006-01-02")] = true
	}
	for id := range adelantan {
		registrados, err := s.repo.PendientesRegistradosHasta(id, hoy)
		if err != nil {
			return err
		}
		sinCrear := 0
		gid := id
		err = s.recorrerVencidos(hoy, FiltroAdelanto{GastoFijoID: &gid}, func(_ models.GastoFijo, _, cobro time.Time, adelantado bool) error {
			if !adelantado && !pagan[id][cobro.Format("2006-01-02")] {
				sinCrear++
			}
			return nil
		})
		if err != nil {
			return err
		}
		if registrados+sinCrear > 0 {
			g, err := s.repo.GetGastoFijo(id)
			if err != nil {
				return err
			}
			return fmt.Errorf("no se puede adelantar %q: primero paga sus pagos atrasados", g.Nombre)
		}
	}
	return nil
}

// Proximos lista los cobros que vencen dentro de dias_aviso_gasto_fijo (dashboard). Hace catch-up
// antes de calcular para que el aviso sea correcto aunque el job diario haya fallado.
func (s *GastosFijosService) Proximos() ([]models.GastoFijoProximo, error) {
	cfg, err := s.config.Snapshot()
	if err != nil {
		return nil, err
	}
	hoy := cfg.Hoy()
	if _, err := s.GenerarHasta(hoy); err != nil {
		return nil, err
	}
	gastos, err := s.repo.ListGastosFijos(true)
	if err != nil {
		return nil, err
	}
	out := []models.GastoFijoProximo{}
	for _, g := range gastos {
		p := siguienteCobro(g, hoy)
		if p == nil {
			continue
		}
		dias := int(p.Sub(hoy).Hours() / 24)
		if dias > cfg.DiasAvisoGastoFijo {
			continue
		}
		out = append(out, models.GastoFijoProximo{
			GastoFijoID: g.ID, Nombre: g.Nombre, Monto: g.Monto, FechaCobro: p.Format("2006-01-02"),
			DiasRestantes: dias, LocalNombre: g.LocalNombre,
		})
	}
	return out, nil
}
