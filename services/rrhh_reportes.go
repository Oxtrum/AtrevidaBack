package services

import (
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

const (
	PeriodoSemanal   = "semanal"
	PeriodoQuincenal = "quincenal"
	PeriodoMensual   = "mensual"
)

const fechaISO = "2006-01-02"

// rangoPeriodoReporte devuelve el periodo que contiene a fecha y el periodo anterior equivalente.
// Semanal: lunes a domingo. Quincenal: del 1 al 15 y del 16 a fin de mes. Mensual: el mes calendario.
func rangoPeriodoReporte(periodo string, fecha time.Time) (desde, hasta, antDesde, antHasta time.Time, err error) {
	f := time.Date(fecha.Year(), fecha.Month(), fecha.Day(), 0, 0, 0, 0, time.UTC)
	finDeMes := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	}
	inicioDeMes := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	switch periodo {
	case PeriodoSemanal:
		// time.Weekday: domingo = 0; se corre para que la semana empiece el lunes.
		desde = f.AddDate(0, 0, -((int(f.Weekday()) + 6) % 7))
		hasta = desde.AddDate(0, 0, 6)
		antDesde, antHasta = desde.AddDate(0, 0, -7), hasta.AddDate(0, 0, -7)
	case PeriodoQuincenal:
		if f.Day() <= 15 {
			desde = inicioDeMes(f)
			hasta = desde.AddDate(0, 0, 14)
			antHasta = desde.AddDate(0, 0, -1)
			antDesde = time.Date(antHasta.Year(), antHasta.Month(), 16, 0, 0, 0, 0, time.UTC)
		} else {
			desde = time.Date(f.Year(), f.Month(), 16, 0, 0, 0, 0, time.UTC)
			hasta = finDeMes(f)
			antDesde = inicioDeMes(f)
			antHasta = antDesde.AddDate(0, 0, 14)
		}
	case PeriodoMensual:
		desde, hasta = inicioDeMes(f), finDeMes(f)
		antHasta = desde.AddDate(0, 0, -1)
		antDesde = inicioDeMes(antHasta)
	default:
		err = errors.New("periodo debe ser semanal, quincenal o mensual")
	}
	return
}

func variacionPct(actual, anterior int) *float64 {
	if anterior <= 0 {
		return nil
	}
	v := redondear1(float64(actual-anterior) / float64(anterior) * 100)
	return &v
}

// clavePersonal identifica a quien atendio: el empleado de RRHH o, si solo se escribio un nombre, ese nombre.
func clavePersonal(f pgsqlrepo.AtencionReporteFila) string {
	if f.EmpleadoID != nil {
		return "e" + strconv.Itoa(*f.EmpleadoID)
	}
	return "n" + strings.ToLower(strings.Join(strings.Fields(f.Nombre), " "))
}

func claveLocal(f pgsqlrepo.AtencionReporteFila) string {
	if f.LocalID != nil {
		return "l" + strconv.Itoa(*f.LocalID)
	}
	return "n" + strings.ToLower(f.LocalNombre)
}

type acumPersonal struct {
	fila        pgsqlrepo.AtencionReporteFila
	atenciones  int
	minutos     float64
	conHorario  int
	dias        map[string]bool
	clientes    map[string]bool
	servicios   map[string]int
	locales     map[string]bool
	ultimaFecha string
}

// bloqueAtenciones calcula las metricas de un ambito a partir de sus atenciones y las del periodo anterior.
func bloqueAtenciones(localID *int, localNombre string, actual, anterior []pgsqlrepo.AtencionReporteFila) models.ReporteAtencionesBloque {
	b := models.ReporteAtencionesBloque{
		LocalID: localID, LocalNombre: localNombre,
		Ranking: []models.PersonalAtenciones{}, Serie: []models.AtencionesDia{}, Registradores: []models.RegistradorAtenciones{},
	}

	antPorPersona := map[string]int{}
	for _, f := range anterior {
		antPorPersona[clavePersonal(f)]++
	}

	personas := map[string]*acumPersonal{}
	orden := []string{}
	porDia := map[string]int{}
	clientes := map[string]bool{}
	registradores := map[string]int{}
	var minutos float64
	conHorario := 0

	for _, f := range actual {
		k := clavePersonal(f)
		p := personas[k]
		if p == nil {
			p = &acumPersonal{fila: f, dias: map[string]bool{}, clientes: map[string]bool{}, servicios: map[string]int{}, locales: map[string]bool{}}
			personas[k] = p
			orden = append(orden, k)
		}
		dia := f.Fecha.Format(fechaISO)
		p.atenciones++
		p.dias[dia] = true
		p.clientes[f.Cliente] = true
		p.locales[f.LocalNombre] = true
		if f.Servicio != "" {
			p.servicios[f.Servicio]++
		}
		if dia > p.ultimaFecha {
			p.ultimaFecha = dia
		}
		if f.Minutos != nil && *f.Minutos > 0 {
			p.minutos += *f.Minutos
			p.conHorario++
			minutos += *f.Minutos
			conHorario++
		}
		porDia[dia]++
		clientes[f.Cliente] = true
		registradores[f.RegistradoPor]++
	}

	total := len(actual)
	for _, k := range orden {
		p := personas[k]
		fila := models.PersonalAtenciones{
			EmpleadoID: p.fila.EmpleadoID, Nombre: p.fila.Nombre, Cargo: p.fila.Cargo,
			Locales:            strings.Join(clavesOrdenadas(p.locales), ", "),
			Atenciones:         p.atenciones,
			ParticipacionPct:   redondear1(float64(p.atenciones) / float64(total) * 100),
			AtencionesAnterior: antPorPersona[k],
			VariacionPct:       variacionPct(p.atenciones, antPorPersona[k]),
			DiasTrabajados:     len(p.dias),
			PromedioPorDia:     redondear1(float64(p.atenciones) / float64(len(p.dias))),
			ClientesDistintos:  len(p.clientes),
			MinutosAtendidos:   int(math.Round(p.minutos)),
			UltimaAtencion:     p.ultimaFecha,
		}
		if p.conHorario > 0 {
			fila.DuracionPromedioMin = redondear1(p.minutos / float64(p.conHorario))
		}
		for servicio, n := range p.servicios {
			if n > fila.ServicioTopCantidad || (n == fila.ServicioTopCantidad && servicio < fila.ServicioTop) {
				fila.ServicioTop, fila.ServicioTopCantidad = servicio, n
			}
		}
		b.Ranking = append(b.Ranking, fila)
	}
	sort.SliceStable(b.Ranking, func(i, j int) bool {
		x, y := b.Ranking[i], b.Ranking[j]
		if x.Atenciones != y.Atenciones {
			return x.Atenciones > y.Atenciones
		}
		if x.MinutosAtendidos != y.MinutosAtendidos {
			return x.MinutosAtendidos > y.MinutosAtendidos
		}
		return strings.ToLower(x.Nombre) < strings.ToLower(y.Nombre)
	})
	for i := range b.Ranking {
		b.Ranking[i].Posicion = i + 1
	}

	for _, dia := range clavesOrdenadas(porDia) {
		b.Serie = append(b.Serie, models.AtencionesDia{Fecha: dia, Atenciones: porDia[dia]})
	}
	for _, usuario := range clavesOrdenadas(registradores) {
		b.Registradores = append(b.Registradores, models.RegistradorAtenciones{
			Usuario: usuario, Registros: registradores[usuario],
			ParticipacionPct: redondear1(float64(registradores[usuario]) / float64(total) * 100),
		})
	}
	sort.SliceStable(b.Registradores, func(i, j int) bool { return b.Registradores[i].Registros > b.Registradores[j].Registros })

	b.Resumen = models.ResumenAtencionesRRHH{
		Atenciones: total, AtencionesAnterior: len(anterior), VariacionPct: variacionPct(total, len(anterior)),
		Personal: len(personas), DiasConAtencion: len(porDia), ClientesDistintos: len(clientes),
		MinutosAtendidos: int(math.Round(minutos)), SinHorario: total - conHorario,
	}
	if len(porDia) > 0 {
		b.Resumen.PromedioDiario = redondear1(float64(total) / float64(len(porDia)))
	}
	if len(personas) > 0 {
		b.Resumen.PromedioPorPersona = redondear1(float64(total) / float64(len(personas)))
	}
	if conHorario > 0 {
		b.Resumen.DuracionPromedioMin = redondear1(minutos / float64(conHorario))
	}
	return b
}

func clavesOrdenadas[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// construirReporteAtenciones arma el bloque general y uno por local con atenciones en el periodo.
func construirReporteAtenciones(actual, anterior []pgsqlrepo.AtencionReporteFila) (models.ReporteAtencionesBloque, []models.ReporteAtencionesBloque) {
	general := bloqueAtenciones(nil, "General", actual, anterior)

	type grupo struct {
		id               *int
		nombre           string
		actual, anterior []pgsqlrepo.AtencionReporteFila
	}
	grupos := map[string]*grupo{}
	orden := []string{}
	for _, f := range actual {
		k := claveLocal(f)
		if grupos[k] == nil {
			grupos[k] = &grupo{id: f.LocalID, nombre: f.LocalNombre}
			orden = append(orden, k)
		}
		grupos[k].actual = append(grupos[k].actual, f)
	}
	for _, f := range anterior {
		if g := grupos[claveLocal(f)]; g != nil {
			g.anterior = append(g.anterior, f)
		}
	}
	locales := make([]models.ReporteAtencionesBloque, 0, len(orden))
	for _, k := range orden {
		g := grupos[k]
		locales = append(locales, bloqueAtenciones(g.id, g.nombre, g.actual, g.anterior))
	}
	sort.SliceStable(locales, func(i, j int) bool {
		if locales[i].Resumen.Atenciones != locales[j].Resumen.Atenciones {
			return locales[i].Resumen.Atenciones > locales[j].Resumen.Atenciones
		}
		return locales[i].LocalNombre < locales[j].LocalNombre
	})
	return general, locales
}

// ReporteAtenciones devuelve el reporte de personal del periodo (semanal, quincenal o mensual) que contiene a fecha.
// Sin fecha usa hoy en la zona horaria configurada.
func (s *RRHHService) ReporteAtenciones(periodo, fecha string) (*models.ReporteAtencionesRRHH, error) {
	periodo = strings.ToLower(strings.TrimSpace(periodo))
	if periodo == "" {
		periodo = PeriodoMensual
	}
	var ref time.Time
	if fecha = strings.TrimSpace(fecha); fecha == "" {
		cfg, err := s.config.Snapshot()
		if err != nil {
			return nil, err
		}
		ref = cfg.Hoy()
	} else {
		t, err := time.Parse(fechaISO, fecha)
		if err != nil {
			return nil, errors.New("fecha debe tener formato YYYY-MM-DD")
		}
		ref = t
	}
	desde, hasta, antDesde, antHasta, err := rangoPeriodoReporte(periodo, ref)
	if err != nil {
		return nil, err
	}
	actual, err := s.repo.AtencionesParaReporte(desde, hasta)
	if err != nil {
		return nil, err
	}
	anterior, err := s.repo.AtencionesParaReporte(antDesde, antHasta)
	if err != nil {
		return nil, err
	}
	general, locales := construirReporteAtenciones(actual, anterior)
	return &models.ReporteAtencionesRRHH{
		Periodo: periodo,
		Desde:   desde.Format(fechaISO), Hasta: hasta.Format(fechaISO),
		DesdeAnterior: antDesde.Format(fechaISO), HastaAnterior: antHasta.Format(fechaISO),
		General: general, Locales: locales,
	}, nil
}
