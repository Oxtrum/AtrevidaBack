package services

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/pagination"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

// rachaMaximaClientes es cuantos periodos hacia atras se leen para medir la fidelidad de un cliente.
const rachaMaximaClientes = 12

// topClientesReporte es cuantos clientes trae cada ranking del resumen; la lista completa va paginada aparte.
const topClientesReporte = 10

type ReporteClientesService struct {
	repo   *pgsqlrepo.ReporteClientesRepo
	config *ConfiguracionService
}

func NewReporteClientesService(repo *pgsqlrepo.ReporteClientesRepo, config *ConfiguracionService) *ReporteClientesService {
	return &ReporteClientesService{repo: repo, config: config}
}

type rangoFechas struct{ desde, hasta time.Time }

// periodosHaciaAtras devuelve n periodos seguidos: [0] es el que contiene a fecha, [1] el anterior, etc.
func periodosHaciaAtras(periodo string, fecha time.Time, n int) ([]rangoFechas, error) {
	out := make([]rangoFechas, 0, n)
	ref := fecha
	for i := 0; i < n; i++ {
		desde, hasta, _, _, err := rangoPeriodoReporte(periodo, ref)
		if err != nil {
			return nil, err
		}
		out = append(out, rangoFechas{desde, hasta})
		ref = desde.AddDate(0, 0, -1)
	}
	return out, nil
}

func soloDia(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// indicePeriodo dice en cual de los periodos cae la fecha; -1 si en ninguno.
func indicePeriodo(periodos []rangoFechas, fecha time.Time) int {
	f := soloDia(fecha)
	for i, p := range periodos {
		if !f.Before(p.desde) && !f.After(p.hasta) {
			return i
		}
	}
	return -1
}

func variacionPctMonto(actual, anterior float64) *float64 {
	if anterior <= 0 {
		return nil
	}
	v := redondear1((actual - anterior) / anterior * 100)
	return &v
}

// datosReporteClientes es todo lo que se lee de la base para armar el reporte.
type datosReporteClientes struct {
	// periodos[0] es el periodo del reporte y periodos[1] el anterior; el resto solo sirve para la racha.
	periodos []rangoFechas
	pagos    []pgsqlrepo.PagoClienteFila
	reservas []pgsqlrepo.ReservaClienteFila
	clientes map[int]pgsqlrepo.ClienteBasico
	deudas   []pgsqlrepo.DeudaClienteFila
	// saldosAFavor es el dinero abonado sin aplicar de cada cliente; se descuenta de lo que debe.
	saldosAFavor map[int]float64
	hoy          time.Time
}

type acumCliente struct {
	ingresos, ingresosAnt     float64
	pagos, pagosAnt           int
	citas, citasAnt           int
	agendadas                 int
	canceladas, inasistencias int
	servicios                 map[string]int
	locales                   map[string]bool
	localesActivos            map[string]bool
	ultima                    string
	// activo[i] = pago o completo una cita en periodos[i].
	activo []bool
	// enPeriodo = tuvo algun movimiento (tambien reservas agendadas o canceladas) en el periodo del reporte.
	enPeriodo bool
}

func enAmbito(ambito, local *int) bool {
	return ambito == nil || (local != nil && *local == *ambito)
}

// bloqueClientes calcula las metricas de un ambito: todos los locales (localID nil) o uno solo.
// Devuelve el bloque del resumen (con rankings acotados) y la lista completa de clientes con movimiento, ordenada
// por dinero generado, que se entrega paginada aparte.
func bloqueClientes(localID *int, localNombre string, d datosReporteClientes) (models.ReporteClientesBloque, []models.ClienteReporte) {
	b := models.ReporteClientesBloque{
		LocalID: localID, LocalNombre: localNombre,
		TopIngresos: []models.ClienteReporte{}, TopCitas: []models.ClienteReporte{}, TopFieles: []models.ClienteReporte{},
		TopCanceladas: []models.ClienteReporte{}, Deudores: []models.ClienteDeuda{}, Serie: []models.ClientesDia{},
	}
	clientes := []models.ClienteReporte{}
	if len(d.periodos) == 0 {
		return b, clientes
	}

	acums := map[int]*acumCliente{}
	acum := func(id int) *acumCliente {
		a := acums[id]
		if a == nil {
			a = &acumCliente{
				servicios: map[string]int{}, locales: map[string]bool{}, localesActivos: map[string]bool{},
				activo: make([]bool, len(d.periodos)),
			}
			acums[id] = a
		}
		return a
	}
	diaIngresos := map[string]float64{}
	diaClientes := map[string]map[int]bool{}
	marcarDia := func(dia string, id int) {
		if diaClientes[dia] == nil {
			diaClientes[dia] = map[int]bool{}
		}
		diaClientes[dia][id] = true
	}
	var sinCliente float64
	citasSinCliente := 0

	for _, p := range d.pagos {
		idx := indicePeriodo(d.periodos, p.Fecha)
		if idx < 0 || !enAmbito(localID, p.LocalID) {
			continue
		}
		if p.ClienteID == nil {
			if idx == 0 {
				sinCliente += p.Monto
			}
			continue
		}
		a := acum(*p.ClienteID)
		a.activo[idx] = true
		switch idx {
		case 0:
			dia := p.Fecha.Format(fechaISO)
			a.ingresos += p.Monto
			a.pagos++
			a.enPeriodo = true
			a.locales[p.LocalNombre] = true
			a.localesActivos[p.LocalNombre] = true
			if dia > a.ultima {
				a.ultima = dia
			}
			diaIngresos[dia] += p.Monto
			marcarDia(dia, *p.ClienteID)
		case 1:
			a.ingresosAnt += p.Monto
			a.pagosAnt++
		}
	}

	for _, r := range d.reservas {
		idx := indicePeriodo(d.periodos, r.Fecha)
		if idx < 0 || !enAmbito(localID, r.LocalID) {
			continue
		}
		completada := r.Estado == "COMPLETADO"
		if r.ClienteID == nil {
			if idx == 0 && completada {
				citasSinCliente++
			}
			continue
		}
		if completada {
			a := acum(*r.ClienteID)
			a.activo[idx] = true
			switch idx {
			case 0:
				dia := r.Fecha.Format(fechaISO)
				a.citas++
				a.enPeriodo = true
				a.locales[r.LocalNombre] = true
				a.localesActivos[r.LocalNombre] = true
				if r.Servicio != "" {
					a.servicios[r.Servicio]++
				}
				if dia > a.ultima {
					a.ultima = dia
				}
				marcarDia(dia, *r.ClienteID)
			case 1:
				a.citasAnt++
			}
			continue
		}
		if idx != 0 {
			continue
		}
		switch r.Estado {
		case "PENDIENTE", "AGENDADO":
			a := acum(*r.ClienteID)
			a.agendadas++
			a.enPeriodo = true
			a.locales[r.LocalNombre] = true
		case "CANCELADO":
			a := acum(*r.ClienteID)
			a.canceladas++
			if r.MotivoCancelacion == "INASISTENCIA" {
				a.inasistencias++
			}
			a.enPeriodo = true
			a.locales[r.LocalNombre] = true
			if dia := r.Fecha.Format(fechaISO); dia > a.ultima {
				a.ultima = dia
			}
		}
	}

	// Deudas de hoy en el ambito, sumadas por cliente. Al final se descuenta el saldo a favor sin aplicar, para que
	// el total coincida con la lista "Con deuda" (cargos menos abonos). El saldo a favor no pertenece a un local,
	// asi que en un bloque por local se descuenta entero de lo que el cliente debe ahi.
	type deuda struct {
		pendiente float64
		cargos    int
		desde     time.Time
	}
	deudas := map[int]*deuda{}
	for _, f := range d.deudas {
		if !enAmbito(localID, f.LocalID) {
			continue
		}
		x := deudas[f.ClienteID]
		if x == nil {
			x = &deuda{desde: f.Desde}
			deudas[f.ClienteID] = x
		}
		x.pendiente += f.Pendiente
		x.cargos += f.Cargos
		if f.Desde.Before(x.desde) {
			x.desde = f.Desde
		}
	}
	for id, x := range deudas {
		if x.pendiente -= d.saldosAFavor[id]; x.pendiente < 0.01 {
			delete(deudas, id)
		}
	}

	nombreDe := func(id int) (string, string) {
		if c, ok := d.clientes[id]; ok && c.Nombre != "" {
			return c.Nombre, c.Telefono
		}
		return "Cliente #" + strconv.Itoa(id), ""
	}

	res := &b.Resumen
	pagaronAnt := 0
	for id, a := range acums {
		res.IngresosAnterior += a.ingresosAnt
		res.CitasAnterior += a.citasAnt
		if a.pagosAnt > 0 {
			pagaronAnt++
		}
		if len(a.activo) > 1 && a.activo[1] {
			res.ClientesActivosAnterior++
			if a.activo[0] {
				res.ClientesQueVolvieron++
			}
		}
		if !a.enPeriodo {
			continue
		}
		nombre, telefono := nombreDe(id)
		fila := models.ClienteReporte{
			ClienteID: id, Nombre: nombre, Telefono: telefono,
			Locales:          strings.Join(clavesOrdenadas(a.locales), ", "),
			Ingresos:         redondearMonto(a.ingresos),
			IngresosAnterior: redondearMonto(a.ingresosAnt),
			VariacionPct:     variacionPctMonto(a.ingresos, a.ingresosAnt),
			Pagos:            a.pagos,
			Citas:            a.citas, CitasAnterior: a.citasAnt, CitasAgendadas: a.agendadas,
			Canceladas: a.canceladas, Inasistencias: a.inasistencias,
			UltimaActividad: a.ultima,
		}
		if a.pagos > 0 {
			fila.TicketPromedio = redondearMonto(a.ingresos / float64(a.pagos))
			res.ClientesQuePagaron++
		}
		for servicio, n := range a.servicios {
			if n > fila.ServicioTopCantidad || (n == fila.ServicioTopCantidad && servicio < fila.ServicioTop) {
				fila.ServicioTop, fila.ServicioTopCantidad = servicio, n
			}
		}
		for _, activo := range a.activo {
			if !activo {
				break
			}
			fila.RachaPeriodos++
		}
		if a.activo[0] {
			res.ClientesActivos++
			primera := d.clientes[id].PrimeraActividad
			if primera != nil && !soloDia(*primera).Before(d.periodos[0].desde) {
				fila.EsNuevo = true
				res.ClientesNuevos++
			}
			if localID == nil && len(a.localesActivos) > 1 {
				res.ClientesCompartidos++
			}
		}
		if x := deudas[id]; x != nil {
			fila.SaldoPendiente = redondearMonto(x.pendiente)
		}
		res.Ingresos += a.ingresos
		res.Citas += a.citas
		res.Canceladas += a.canceladas
		res.Inasistencias += a.inasistencias
		clientes = append(clientes, fila)
	}

	for i := range clientes {
		if res.Ingresos > 0 {
			clientes[i].ParticipacionPct = redondear1(clientes[i].Ingresos / res.Ingresos * 100)
		}
	}
	sort.SliceStable(clientes, func(i, j int) bool {
		x, y := clientes[i], clientes[j]
		if x.Ingresos != y.Ingresos {
			return x.Ingresos > y.Ingresos
		}
		if x.Citas != y.Citas {
			return x.Citas > y.Citas
		}
		if lx, ly := strings.ToLower(x.Nombre), strings.ToLower(y.Nombre); lx != ly {
			return lx < ly
		}
		return x.ClienteID < y.ClienteID
	})
	for i := range clientes {
		clientes[i].Posicion = i + 1
	}

	for id, x := range deudas {
		nombre, telefono := nombreDe(id)
		dias := int(d.hoy.Sub(soloDia(x.desde)).Hours() / 24)
		if dias < 0 {
			dias = 0
		}
		b.Deudores = append(b.Deudores, models.ClienteDeuda{
			ClienteID: id, Nombre: nombre, Telefono: telefono, SaldoPendiente: redondearMonto(x.pendiente),
			Cargos: x.cargos, DeudaDesde: x.desde.Format(fechaISO), DiasDeuda: dias,
		})
		res.PorCobrar += x.pendiente
	}
	sort.SliceStable(b.Deudores, func(i, j int) bool {
		x, y := b.Deudores[i], b.Deudores[j]
		if x.SaldoPendiente != y.SaldoPendiente {
			return x.SaldoPendiente > y.SaldoPendiente
		}
		return x.ClienteID < y.ClienteID
	})

	dias := map[string]bool{}
	for dia := range diaClientes {
		dias[dia] = true
	}
	for _, dia := range clavesOrdenadas(dias) {
		b.Serie = append(b.Serie, models.ClientesDia{
			Fecha: dia, Clientes: len(diaClientes[dia]), Ingresos: redondearMonto(diaIngresos[dia]),
		})
	}

	res.ClientesRecurrentes = res.ClientesActivos - res.ClientesNuevos
	res.VariacionClientesPct = variacionPct(res.ClientesActivos, res.ClientesActivosAnterior)
	res.VariacionIngresosPct = variacionPctMonto(res.Ingresos, res.IngresosAnterior)
	if res.ClientesQuePagaron > 0 {
		res.TicketPromedioCliente = redondearMonto(res.Ingresos / float64(res.ClientesQuePagaron))
	}
	if pagaronAnt > 0 {
		res.TicketPromedioClienteAnterior = redondearMonto(res.IngresosAnterior / float64(pagaronAnt))
	}
	if total := res.Ingresos + sinCliente; total > 0 {
		res.IngresosSinClientePct = redondear1(sinCliente / total * 100)
	}
	if res.ClientesActivosAnterior > 0 {
		v := redondear1(float64(res.ClientesQueVolvieron) / float64(res.ClientesActivosAnterior) * 100)
		res.TasaRetornoPct = &v
	}
	res.Ingresos = redondearMonto(res.Ingresos)
	res.IngresosAnterior = redondearMonto(res.IngresosAnterior)
	res.IngresosSinCliente = redondearMonto(sinCliente)
	res.CitasSinCliente = citasSinCliente
	res.PorCobrar = redondearMonto(res.PorCobrar)
	res.Deudores = len(b.Deudores)
	if len(b.Deudores) > topClientesReporte {
		b.Deudores = b.Deudores[:topClientesReporte]
	}

	b.TopIngresos = topClientes(clientes, func(c models.ClienteReporte) float64 { return c.Ingresos }, 0.01)
	b.TopCitas = topClientes(clientes, func(c models.ClienteReporte) float64 { return float64(c.Citas) }, 1)
	b.TopFieles = topClientes(clientes, func(c models.ClienteReporte) float64 { return float64(c.RachaPeriodos) }, 2)
	b.TopCanceladas = topClientes(clientes, func(c models.ClienteReporte) float64 { return float64(c.Canceladas) }, 1)
	return b, clientes
}

// topClientes devuelve los primeros topClientesReporte clientes segun valor (de mayor a menor), solo los que
// llegan a minimo. Los empates conservan el orden por dinero generado.
func topClientes(filas []models.ClienteReporte, valor func(models.ClienteReporte) float64, minimo float64) []models.ClienteReporte {
	out := []models.ClienteReporte{}
	for _, f := range filas {
		if valor(f) >= minimo {
			out = append(out, f)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return valor(out[i]) > valor(out[j]) })
	if len(out) > topClientesReporte {
		out = out[:topClientesReporte]
	}
	return out
}

// construirReporteClientes arma el bloque general y uno por local con movimiento en el periodo o con deudas.
func construirReporteClientes(d datosReporteClientes) (models.ReporteClientesBloque, []models.ReporteClientesBloque) {
	general, _ := bloqueClientes(nil, "General", d)

	nombres := map[int]string{}
	orden := []int{}
	ver := func(id *int, nombre string) {
		if id == nil {
			return
		}
		if _, ok := nombres[*id]; !ok {
			nombres[*id] = nombre
			orden = append(orden, *id)
		}
	}
	for _, p := range d.pagos {
		if indicePeriodo(d.periodos, p.Fecha) == 0 {
			ver(p.LocalID, p.LocalNombre)
		}
	}
	for _, r := range d.reservas {
		if indicePeriodo(d.periodos, r.Fecha) == 0 {
			ver(r.LocalID, r.LocalNombre)
		}
	}
	for _, f := range d.deudas {
		ver(f.LocalID, f.LocalNombre)
	}

	locales := make([]models.ReporteClientesBloque, 0, len(orden))
	for _, id := range orden {
		localID := id
		bloque, _ := bloqueClientes(&localID, nombres[id], d)
		locales = append(locales, bloque)
	}
	sort.SliceStable(locales, func(i, j int) bool {
		if locales[i].Resumen.Ingresos != locales[j].Resumen.Ingresos {
			return locales[i].Resumen.Ingresos > locales[j].Resumen.Ingresos
		}
		return locales[i].LocalNombre < locales[j].LocalNombre
	})
	return general, locales
}

// cargarDatos valida periodo y fecha y lee todo lo necesario para armar el reporte. Devuelve el periodo
// normalizado. Sin fecha usa hoy en la zona horaria configurada.
func (s *ReporteClientesService) cargarDatos(periodo, fecha string) (string, datosReporteClientes, error) {
	var d datosReporteClientes
	periodo = strings.ToLower(strings.TrimSpace(periodo))
	if periodo == "" {
		periodo = PeriodoMensual
	}
	cfg, err := s.config.Snapshot()
	if err != nil {
		return "", d, err
	}
	hoy := cfg.Hoy()
	ref := hoy
	if fecha = strings.TrimSpace(fecha); fecha != "" {
		if ref, err = time.Parse(fechaISO, fecha); err != nil {
			return "", d, errors.New("fecha debe tener formato YYYY-MM-DD")
		}
	}
	periodos, err := periodosHaciaAtras(periodo, ref, rachaMaximaClientes)
	if err != nil {
		return "", d, err
	}
	desde, hasta := periodos[len(periodos)-1].desde, periodos[0].hasta

	pagos, err := s.repo.PagosParaReporte(desde, hasta)
	if err != nil {
		return "", d, err
	}
	reservas, err := s.repo.ReservasParaReporte(desde, hasta)
	if err != nil {
		return "", d, err
	}
	listaClientes, err := s.repo.ClientesConPrimeraActividad()
	if err != nil {
		return "", d, err
	}
	deudas, err := s.repo.DeudasVigentes()
	if err != nil {
		return "", d, err
	}
	aFavor, err := s.repo.SaldosAFavor()
	if err != nil {
		return "", d, err
	}
	saldosAFavor := make(map[int]float64, len(aFavor))
	for _, f := range aFavor {
		saldosAFavor[f.ClienteID] = f.Disponible
	}
	clientes := make(map[int]pgsqlrepo.ClienteBasico, len(listaClientes))
	for _, c := range listaClientes {
		clientes[c.ID] = c
	}
	return periodo, datosReporteClientes{
		periodos: periodos, pagos: pagos, reservas: reservas, clientes: clientes, deudas: deudas,
		saldosAFavor: saldosAFavor, hoy: hoy,
	}, nil
}

// Generar devuelve el resumen del reporte de clientes del periodo (semanal, quincenal o mensual) que contiene a
// fecha: totales, rankings acotados y actividad por dia. La lista completa de clientes se pide con Detalle.
func (s *ReporteClientesService) Generar(periodo, fecha string) (*models.ReporteClientes, error) {
	periodo, d, err := s.cargarDatos(periodo, fecha)
	if err != nil {
		return nil, err
	}
	general, locales := construirReporteClientes(d)
	return &models.ReporteClientes{
		Periodo: periodo,
		Desde:   d.periodos[0].desde.Format(fechaISO), Hasta: d.periodos[0].hasta.Format(fechaISO),
		DesdeAnterior: d.periodos[1].desde.Format(fechaISO), HastaAnterior: d.periodos[1].hasta.Format(fechaISO),
		RachaMaxima: rachaMaximaClientes,
		General:     general, Locales: locales,
	}, nil
}

// ClientesReporteOrdenables lista las columnas por las que se puede ordenar el detalle de clientes (ordenar_por).
// El orden se resuelve en memoria, asi que el valor del mapa no se usa: solo importa la clave.
var ClientesReporteOrdenables = pagination.Columns{
	"posicion": "", "nombre": "", "ingresos": "", "variacion_pct": "", "pagos": "", "citas": "", "canceladas": "",
	"racha_periodos": "", "servicio_top": "", "ultima_actividad": "", "saldo_pendiente": "",
}

// FiltroDetalleClientes filtra, ordena y pagina la lista de clientes del reporte.
type FiltroDetalleClientes struct {
	Periodo string
	Fecha   string
	// LocalID nil = todos los locales.
	LocalID    *int
	Busqueda   string
	OrdenarPor string
	OrdenDesc  bool
	// Limit es cuantas filas devolver desde Offset; 0 = todas.
	Limit  int
	Offset int
}

func compararNumeros(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// compararClientes devuelve <0, 0 o >0 segun el orden ascendente de la columna. El segundo valor es true cuando
// uno de los dos no tiene dato: ese va al final en cualquier sentido.
func compararClientes(key string, x, y models.ClienteReporte) (int, bool) {
	texto := func(a, b string) (int, bool) {
		if (a == "") != (b == "") {
			if a == "" {
				return 1, true
			}
			return -1, true
		}
		return strings.Compare(strings.ToLower(a), strings.ToLower(b)), false
	}
	switch key {
	case "nombre":
		return texto(x.Nombre, y.Nombre)
	case "servicio_top":
		return texto(x.ServicioTop, y.ServicioTop)
	case "ultima_actividad":
		return texto(x.UltimaActividad, y.UltimaActividad)
	case "variacion_pct":
		if (x.VariacionPct == nil) != (y.VariacionPct == nil) {
			if x.VariacionPct == nil {
				return 1, true
			}
			return -1, true
		}
		if x.VariacionPct == nil {
			return 0, false
		}
		return compararNumeros(*x.VariacionPct, *y.VariacionPct), false
	case "ingresos":
		return compararNumeros(x.Ingresos, y.Ingresos), false
	case "pagos":
		return compararNumeros(float64(x.Pagos), float64(y.Pagos)), false
	case "citas":
		return compararNumeros(float64(x.Citas), float64(y.Citas)), false
	case "canceladas":
		return compararNumeros(float64(x.Canceladas), float64(y.Canceladas)), false
	case "racha_periodos":
		return compararNumeros(float64(x.RachaPeriodos), float64(y.RachaPeriodos)), false
	case "saldo_pendiente":
		return compararNumeros(x.SaldoPendiente, y.SaldoPendiente), false
	}
	return compararNumeros(float64(x.Posicion), float64(y.Posicion)), false
}

// filtrarOrdenarClientes aplica la busqueda (nombre, telefono o servicio favorito) y el orden pedido. Sin orden
// queda por posicion (dinero generado). Los empates se resuelven por posicion, que es unica: asi las paginas por
// desplazamiento no repiten ni saltan filas.
func filtrarOrdenarClientes(filas []models.ClienteReporte, busqueda, ordenarPor string, desc bool) []models.ClienteReporte {
	out := filas
	if q := strings.ToLower(strings.TrimSpace(busqueda)); q != "" {
		out = []models.ClienteReporte{}
		for _, f := range filas {
			if strings.Contains(strings.ToLower(f.Nombre), q) || strings.Contains(f.Telefono, q) ||
				strings.Contains(strings.ToLower(f.ServicioTop), q) {
				out = append(out, f)
			}
		}
	}
	if ordenarPor == "" {
		return out
	}
	sort.SliceStable(out, func(i, j int) bool {
		c, sinDato := compararClientes(ordenarPor, out[i], out[j])
		if c == 0 {
			return out[i].Posicion < out[j].Posicion
		}
		if desc && !sinDato {
			return c > 0
		}
		return c < 0
	})
	return out
}

// Detalle devuelve una pagina de los clientes con movimiento en el periodo y cuantos hay en total con ese filtro.
func (s *ReporteClientesService) Detalle(f FiltroDetalleClientes) ([]models.ClienteReporte, int, error) {
	if f.LocalID != nil && *f.LocalID <= 0 {
		return nil, 0, errors.New("local_id invalido")
	}
	if f.Offset < 0 || f.Limit < 0 {
		return nil, 0, errors.New("paginacion invalida")
	}
	_, d, err := s.cargarDatos(f.Periodo, f.Fecha)
	if err != nil {
		return nil, 0, err
	}
	_, clientes := bloqueClientes(f.LocalID, "", d)
	clientes = filtrarOrdenarClientes(clientes, f.Busqueda, f.OrdenarPor, f.OrdenDesc)
	total := len(clientes)
	if f.Offset >= total {
		return []models.ClienteReporte{}, total, nil
	}
	clientes = clientes[f.Offset:]
	if f.Limit > 0 && len(clientes) > f.Limit {
		clientes = clientes[:f.Limit]
	}
	return clientes, total, nil
}
