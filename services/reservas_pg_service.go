package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

type ReservasPGService struct {
	repo          repository.ReservasPGRepository
	serviciosRepo repository.ServiciosRepository
}

func NewReservasPGService(repo repository.ReservasPGRepository, serviciosRepo *pgsqlrepo.ServiciosRepo) *ReservasPGService {
	return &ReservasPGService{
		repo:          repo,
		serviciosRepo: serviciosRepo,
	}
}

// GET

type FiltroReservasPG struct {
	Local          string
	Fecha          string
	FechaDesde     string
	FechaHasta     string
	Cliente        string
	NumeroTelefono string
	Estado         string
	Tipo           string
	Reservados     *bool
}

func (s *ReservasPGService) GetReservasFiltradas(f FiltroReservasPG) ([]models.LocalReservas, error) {

	if f.Cliente != "" {
		soloOcupados := true
		f.Reservados = &soloOcupados
	}
	if f.NumeroTelefono != "" {
		soloOcupados := true
		f.Reservados = &soloOcupados
	}

	if f.Reservados != nil && !*f.Reservados {
		desde, hasta := getRangoTiempoDisp(f.FechaDesde, f.FechaHasta)
		return s.getEspaciosDisponibles(f, desde, hasta)
	}

	filtro := repository.FiltroReservasPG{
		LocalNombre:    f.Local,
		Cliente:        f.Cliente,
		NumeroTelefono: f.NumeroTelefono,
		SoloActivas:    true,
	}

	if f.Tipo != "" {
		filtro.TipoEspacio = tipoNombreALetra(f.Tipo)
	}
	if f.Fecha != "" {
		t, err := time.Parse("2006-01-02", f.Fecha)
		if err != nil {
			return nil, fmt.Errorf("formato de fecha inválido, use YYYY-MM-DD")
		}
		filtro.Fecha = &t
	}
	if f.FechaDesde != "" {
		t, err := time.Parse("2006-01-02", f.FechaDesde)
		if err != nil {
			return nil, fmt.Errorf("formato de fecha_desde inválido, use YYYY-MM-DD")
		}
		filtro.FechaDesde = &t
	}
	if f.FechaHasta != "" {
		t, err := time.Parse("2006-01-02", f.FechaHasta)
		if err != nil {
			return nil, fmt.Errorf("formato de fecha_hasta inválido, use YYYY-MM-DD")
		}
		filtro.FechaHasta = &t
	}

	reservas, err := s.repo.GetReservas(filtro)
	if err != nil {
		return nil, err
	}

	if f.Reservados != nil && *f.Reservados {
		filtradas := filterReservasOcupadasPorEstado(reservas, f.Estado)
		expandidas := transformReservasEnSlots(filtradas)
		return pgsqlrepo.BuildJerarquia(expandidas), nil
	}

	desdeStr, hastaStr := f.FechaDesde, f.FechaHasta
	if f.Fecha != "" {
		desdeStr = f.Fecha
		hastaStr = f.Fecha
	}
	desde, hasta := getRangoTiempoDisp(desdeStr, hastaStr)

	return s.getCalendarioCompleto(f, desde, hasta, reservas)
}

// getCalendarioCompleto genera todos los slots libres del rango y luego agrega reservas
func (s *ReservasPGService) getCalendarioCompleto(
	f FiltroReservasPG,
	desde, hasta time.Time,
	reservasDB []models.ReservaPGCompleta,
) ([]models.LocalReservas, error) {

	capacidades, err := s.repo.GetCapacidades(f.Local)
	if err != nil {
		return nil, err
	}

	reservasExpandidas := transformReservasEnSlots(reservasDB)
	reservasExpandidasFiltradas := transformReservasEnSlots(filterReservasPorEstado(reservasDB, f.Estado))

	// Indexar ocupados
	type slotIdx struct {
		local string
		fecha string
		tipo  string
		hora  string // hora_desde del slot
	}
	ocupadosIdx := map[slotIdx][]models.ReservaPGCompleta{}
	for _, rv := range reservasExpandidas {
		key := slotIdx{
			local: rv.LocalNombre,
			fecha: rv.Fecha.Format("2006-01-02"),
			tipo:  strings.ToUpper(rv.TipoEspacio),
			hora:  rv.HoraDesde,
		}
		ocupadosIdx[key] = append(ocupadosIdx[key], rv)
	}
	ocupadosFiltradosIdx := map[slotIdx][]models.ReservaPGCompleta{}
	for _, rv := range reservasExpandidasFiltradas {
		key := slotIdx{
			local: rv.LocalNombre,
			fecha: rv.Fecha.Format("2006-01-02"),
			tipo:  strings.ToUpper(rv.TipoEspacio),
			hora:  rv.HoraDesde,
		}
		ocupadosFiltradosIdx[key] = append(ocupadosFiltradosIdx[key], rv)
	}

	// Construir calendario completo
	var resultado []models.ReservaPGCompleta

	for _, cap := range capacidades {
		if f.Tipo != "" && !strings.EqualFold(cap.TipoEspacio, tipoNombreALetra(f.Tipo)) {
			continue
		}
		tipo := strings.ToUpper(cap.TipoEspacio)

		for d := desde; !d.After(hasta); d = d.AddDate(0, 0, 1) {
			slots := horarioLocal(d)
			fecha := d.Format("2006-01-02")

			for _, slot := range slots {
				key := slotIdx{
					local: cap.LocalNombre,
					fecha: fecha,
					tipo:  tipo,
					hora:  slot[0],
				}

				ocupadosEnSlot := ocupadosIdx[key]
				ocupadosVisiblesEnSlot := ocupadosFiltradosIdx[key]
				cantOcupados := len(ocupadosEnSlot)

				resultado = append(resultado, ocupadosVisiblesEnSlot...)

				libres := cap.Capacidad - cantOcupados
				for i := 0; i < libres; i++ {
					resultado = append(resultado, models.ReservaPGCompleta{
						ReservaPG: models.ReservaPG{
							LocalNombre: cap.LocalNombre,
							TipoEspacio: tipo,
							Fecha:       d,
							HoraDesde:   slot[0],
							HoraHasta:   slot[1],
							Cliente:     "", // libre
						},
					})
				}
			}
		}
	}

	return pgsqlrepo.BuildJerarquia(resultado), nil
}

// transformReservasEnSlots toma reservas con rango de tiempo arbitrario
func transformReservasEnSlots(reservas []models.ReservaPGCompleta) []models.ReservaPGCompleta {
	var resultado []models.ReservaPGCompleta

	for _, rv := range reservas {
		slots := partirEnSlots60(rv.HoraDesde, rv.HoraHasta)
		if len(slots) == 0 {
			// Si no se puede partir (horario inválido), lo dejamos tal cual
			resultado = append(resultado, rv)
			continue
		}
		for _, slot := range slots {
			entrada := rv // copia valor
			entrada.HoraDesde = slot[0]
			entrada.HoraHasta = slot[1]
			resultado = append(resultado, entrada)
		}
	}

	return resultado
}

// partirEnSlots60 divide un rango [desde, hasta] en bloques de 60 minutos.
func partirEnSlots60(desde, hasta string) [][2]string {
	parseHora := func(h string) (time.Time, error) {
		if len(h) > 5 {
			h = h[:5]
		}
		t, err := time.Parse("15:04", h)
		if err != nil {
			t, err = time.Parse("15:4", h)
		}
		return t, err
	}

	tDesde, err1 := parseHora(desde)
	tHasta, err2 := parseHora(hasta)
	if err1 != nil || err2 != nil {
		return nil
	}

	var slots [][2]string
	cur := tDesde
	for cur.Before(tHasta) {
		siguiente := cur.Add(60 * time.Minute)
		if siguiente.After(tHasta) {
			siguiente = tHasta
		}
		slots = append(slots, [2]string{
			fmt.Sprintf("%02d:%02d", cur.Hour(), cur.Minute()),
			fmt.Sprintf("%02d:%02d", siguiente.Hour(), siguiente.Minute()),
		})
		cur = siguiente
	}
	return slots
}

// getEspaciosDisponibles retorna solo los espacios libres como jerarquía.
func (s *ReservasPGService) getEspaciosDisponibles(f FiltroReservasPG, desde, hasta time.Time) ([]models.LocalReservas, error) {
	raw, err := s.getEspaciosLibresRaw(f, desde, hasta)
	if err != nil {
		return nil, err
	}
	return pgsqlrepo.BuildJerarquia(raw), nil
}

// getEspaciosLibresRaw calcula los slots libres y los devuelve como []ReservaPGCompleta,
func (s *ReservasPGService) getEspaciosLibresRaw(f FiltroReservasPG, desde, hasta time.Time) ([]models.ReservaPGCompleta, error) {
	capacidades, err := s.repo.GetCapacidades(f.Local)
	if err != nil {
		return nil, err
	}

	// reservas ocupadas en el rango para restar
	filtro := repository.FiltroReservasPG{
		LocalNombre: f.Local,
		SoloActivas: true,
		FechaDesde:  &desde,
		FechaHasta:  &hasta,
	}
	if f.Tipo != "" {
		filtro.TipoEspacio = tipoNombreALetra(f.Tipo)
	}

	ocupadas, err := s.repo.GetReservas(filtro)
	if err != nil {
		return nil, err
	}

	// indexar ocupados: localNombre → fecha → tipo → []{ horaDesde, horaHasta }
	type rangoHora struct{ desde, hasta string }
	ocupadosIdx := map[string]map[string]map[string][]rangoHora{}
	for _, rv := range ocupadas {
		local := rv.LocalNombre
		fecha := rv.Fecha.Format("2006-01-02")
		tipo := strings.ToUpper(rv.TipoEspacio)
		if ocupadosIdx[local] == nil {
			ocupadosIdx[local] = map[string]map[string][]rangoHora{}
		}
		if ocupadosIdx[local][fecha] == nil {
			ocupadosIdx[local][fecha] = map[string][]rangoHora{}
		}
		ocupadosIdx[local][fecha][tipo] = append(
			ocupadosIdx[local][fecha][tipo],
			rangoHora{rv.HoraDesde, rv.HoraHasta},
		)
	}

	// generar items disponibles por local, fecha, slot y tipo
	var resultado []models.ReservaPGCompleta
	for _, cap := range capacidades {
		if f.Tipo != "" && !strings.EqualFold(cap.TipoEspacio, tipoNombreALetra(f.Tipo)) {
			continue
		}
		for d := desde; !d.After(hasta); d = d.AddDate(0, 0, 1) {
			slots := horarioLocal(d)
			fecha := d.Format("2006-01-02")
			tipo := strings.ToUpper(cap.TipoEspacio)
			rangosOcupados := ocupadosIdx[cap.LocalNombre][fecha][tipo]

			for _, slot := range slots {
				// solapamiento: reserva.desde < slot.hasta AND reserva.hasta > slot.desde
				ocupados := 0
				for _, r := range rangosOcupados {
					if r.desde < slot[1] && r.hasta > slot[0] {
						ocupados++
					}
				}
				libres := cap.Capacidad - ocupados
				for i := 0; i < libres; i++ {
					resultado = append(resultado, models.ReservaPGCompleta{
						ReservaPG: models.ReservaPG{
							LocalNombre: cap.LocalNombre,
							TipoEspacio: tipo,
							Fecha:       d,
							HoraDesde:   slot[0],
							HoraHasta:   slot[1],
							Cliente:     "", // libre
						},
					})
				}
			}
		}
	}

	return resultado, nil
}

// horarioLocal retorna los slots de 60 min disponibles para un día dado.
func horarioLocal(fecha time.Time) [][2]string {
	if fecha.Weekday() == time.Sunday {
		return nil
	}
	if fecha.Weekday() == time.Saturday {
		return generarSlots60("08:00", "15:00")
	}
	return generarSlots60("08:00", "20:00")
}

// generarSlots60 produce pares [desde, hasta] de 60 min entre apertura y cierre.
func generarSlots60(apertura, cierre string) [][2]string {
	t, _ := time.Parse("15:04", apertura)
	fin, _ := time.Parse("15:04", cierre)
	var slots [][2]string
	for t.Before(fin) {
		siguiente := t.Add(60 * time.Minute)
		if siguiente.After(fin) {
			break
		}
		slots = append(slots, [2]string{
			fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute()),
			fmt.Sprintf("%02d:%02d", siguiente.Hour(), siguiente.Minute()),
		})
		t = siguiente
	}
	return slots
}

// getRangoTiempoDisp aplica las reglas de defaulting de fechas para disponibles.
func getRangoTiempoDisp(desdeStr, hastaStr string) (time.Time, time.Time) {
	hoy := time.Now().Truncate(24 * time.Hour)

	if desdeStr == "" && hastaStr == "" {
		return hoy, hoy
	}

	if desdeStr != "" && hastaStr == "" {
		desde, err := time.Parse("2006-01-02", desdeStr)
		if err != nil {
			desde = hoy
		}
		return desde, desde.AddDate(0, 0, 2)
	}

	if desdeStr == "" && hastaStr != "" {
		hasta, err := time.Parse("2006-01-02", hastaStr)
		if err != nil {
			hasta = hoy
		}
		// max(hoy, hasta - 2 días)
		candidato := hasta.AddDate(0, 0, -2)
		if candidato.Before(hoy) {
			candidato = hoy
		}
		return candidato, hasta
	}

	// ambos presentes
	desde, err1 := time.Parse("2006-01-02", desdeStr)
	hasta, err2 := time.Parse("2006-01-02", hastaStr)
	if err1 != nil {
		desde = hoy
	}
	if err2 != nil {
		hasta = hoy
	}
	return desde, hasta
}

// ReservaSimple es la representación plana de una reserva para GET /bd/reservas.
type ReservaSimple struct {
	ID             int      `json:"id"`
	Local          string   `json:"local"`
	Tipo           string   `json:"tipo"`
	Fecha          string   `json:"fecha"`
	HoraDesde      string   `json:"hora_desde"`
	HoraHasta      string   `json:"hora_hasta"`
	Cliente        string   `json:"cliente"`
	Estado         *string  `json:"estado,omitempty"`
	NumeroTelefono *string  `json:"numero_telefono,omitempty"`
	Servicio       *string  `json:"servicio,omitempty"`
	Precio         *float64 `json:"precio,omitempty"`
	Notas          *string  `json:"notas,omitempty"`
}

type FiltroReservasSimple struct {
	Local          string
	Fecha          string
	FechaDesde     string
	FechaHasta     string
	Cliente        string
	NumeroTelefono string
	Estado         string
	Tipo           string
}

func (s *ReservasPGService) GetReservasSimple(f FiltroReservasSimple) ([]ReservaSimple, error) {
	filtro := repository.FiltroReservasPG{
		LocalNombre:    f.Local,
		Cliente:        f.Cliente,
		NumeroTelefono: f.NumeroTelefono,
		SoloActivas:    true,
	}
	if f.Tipo != "" {
		filtro.TipoEspacio = tipoNombreALetra(f.Tipo)
	}
	if f.Fecha != "" {
		t, err := time.Parse("2006-01-02", f.Fecha)
		if err != nil {
			return nil, fmt.Errorf("formato de fecha inválido, use YYYY-MM-DD")
		}
		filtro.Fecha = &t
	}
	if f.FechaDesde != "" {
		t, err := time.Parse("2006-01-02", f.FechaDesde)
		if err != nil {
			return nil, fmt.Errorf("formato de fecha_desde inválido, use YYYY-MM-DD")
		}
		filtro.FechaDesde = &t
	}
	if f.FechaHasta != "" {
		t, err := time.Parse("2006-01-02", f.FechaHasta)
		if err != nil {
			return nil, fmt.Errorf("formato de fecha_hasta inválido, use YYYY-MM-DD")
		}
		filtro.FechaHasta = &t
	}
	reservas, err := s.repo.GetReservas(filtro)
	if err != nil {
		return nil, err
	}
	reservas = filterReservasPorEstado(reservas, f.Estado)
	resultado := make([]ReservaSimple, 0, len(reservas))
	for _, rv := range reservas {
		resultado = append(resultado, ReservaSimple{
			ID:             rv.ID,
			Local:          rv.LocalNombre,
			Tipo:           tipoLetraANombreService(rv.TipoEspacio),
			Fecha:          rv.Fecha.Format("2006-01-02"),
			HoraDesde:      formatHoraService(rv.HoraDesde),
			HoraHasta:      formatHoraService(rv.HoraHasta),
			Cliente:        rv.Cliente,
			Estado:         rv.Estado,
			NumeroTelefono: rv.NumeroTelefono,
			Servicio:       rv.ServicioNombre,
			Precio:         rv.Precio,
			Notas:          rv.Notas,
		})
	}
	return resultado, nil
}

func (s *ReservasPGService) GetReservaByID(id int) (*ReservaSimple, error) {
	rv, err := s.repo.GetReservaByID(id)
	if err != nil {
		return nil, err
	}

	return &ReservaSimple{
		ID:             rv.ID,
		Local:          rv.LocalNombre,
		Tipo:           tipoLetraANombreService(rv.TipoEspacio),
		Fecha:          rv.Fecha.Format("2006-01-02"),
		HoraDesde:      formatHoraService(rv.HoraDesde),
		HoraHasta:      formatHoraService(rv.HoraHasta),
		Cliente:        rv.Cliente,
		Estado:         rv.Estado,
		NumeroTelefono: rv.NumeroTelefono,
		Servicio:       rv.ServicioNombre,
		Precio:         rv.Precio,
		Notas:          rv.Notas,
	}, nil
}

func tipoLetraANombreService(letra string) string {
	switch strings.ToUpper(letra) {
	case "M":
		return "mesa"
	case "B":
		return "bicicleta"
	}
	return strings.ToLower(letra)
}

func formatHoraService(h string) string {
	if len(h) > 5 {
		h = h[:5]
	}
	return h
}

// POST
type CrearReservaPGInput struct {
	Local     string
	Fecha     string
	HoraDesde string
	HoraHasta string
	Tipo      string
	Cliente   string
	Telefono  string
	Estado    string
	Servicio  string
	Precio    *float64
	Notas     string
	PlanID    *int
}

func (s *ReservasPGService) CrearReserva(input CrearReservaPGInput) (int, error) {
	fecha, err := time.Parse("2006-01-02", input.Fecha)
	if err != nil {
		return 0, fmt.Errorf("formato de fecha inválido, use YYYY-MM-DD")
	}

	horaHasta := input.HoraHasta
	if horaHasta == "" {
		horaHasta = sumar60Min(input.HoraDesde)
	}

	/* COMENTADO HASTA QUE FRONTEND RECUPERE SERVICIOS DE LA BD

	// 1. Validar servicio
	if strings.TrimSpace(input.Servicio) != "" {

		servicio, err := s.serviciosRepo.GetServicioByNombre(input.Servicio)
		if err != nil {
			return 0, err
		}

		if strings.TrimSpace(servicio.TipoEspacio) != "" {
			input.Tipo = strings.ToUpper(servicio.TipoEspacio)

		} else if strings.TrimSpace(input.Tipo) != "" {
			input.Tipo = strings.ToUpper(input.Tipo)
		}

		if input.Precio == nil {

			var precio float64

			_, err := fmt.Sscanf(servicio.Costo, "%f", &precio)
			if err != nil {
				return 0, fmt.Errorf(
					"Error al convertir precio del servicio '%s'",
					servicio.Nombre,
				)
			}

			input.Precio = &precio
		}
	}
	*/

	// Mantener comportamiento manual temporal
	input.Tipo = strings.ToUpper(strings.TrimSpace(input.Tipo))
	estadoFinal := "PENDIENTE"
	if strings.TrimSpace(input.Estado) != "" {
		estadoRecibido, err := NormalizarEstadoReserva(input.Estado)
		if err != nil {
			return 0, err
		}
		if estadoRecibido != "PENDIENTE" {
			return 0, errors.New("una reserva nueva siempre inicia en estado PENDIENTE")
		}
	}

	/*if err := s.validarDisponibilidad(input.Local, &fecha, input.HoraDesde, horaHasta, input.Tipo, nil); err != nil {
		return 0, err
	}*/

	// When no sabes

	id, err := s.repo.CreateReserva(repository.CreateReservaInput{
		LocalNombre:    input.Local,
		TipoEspacio:    strings.ToUpper(input.Tipo),
		Fecha:          fecha,
		HoraDesde:      input.HoraDesde,
		HoraHasta:      horaHasta,
		Cliente:        input.Cliente,
		Estado:         estadoFinal,
		NumeroTelefono: input.Telefono,
		ServicioNombre: input.Servicio,
		Precio:         input.Precio,
		Notas:          input.Notas,
		PlanID:         input.PlanID,
	})
	return id, err
}

type ActualizarEstadoReservaInput struct {
	Id     int
	Estado string
	Causa  string
}

func (s *ReservasPGService) ActualizarEstadoReserva(input ActualizarEstadoReservaInput) error {
	current, err := s.repo.GetReservaByID(input.Id)
	if err != nil {
		return fmt.Errorf("No se pudo recuperar la reserva actual: %v", err)
	}

	estadoActual, err := estadoReservaActual(current)
	if err != nil {
		return err
	}

	estado, err := NormalizarEstadoReserva(input.Estado)
	if err != nil {
		return err
	}

	if !canTransitionReservaEstado(estadoActual, estado) {
		return fmt.Errorf("transicion de estado invalida: %s -> %s", estadoActual, estado)
	}

	if estado == "RECHAZADO" && strings.TrimSpace(input.Causa) == "" {
		return errors.New("causa es requerida cuando el estado es RECHAZADO")
	}

	return s.repo.UpdateReservaEstado(input.Id, estado)
}

// PATCH
type ActualizarReservaPGInput struct {
	Id        int
	Local     string
	Fecha     string
	HoraDesde string
	Tipo      string
	Cliente   string

	NuevaFecha          string
	NuevaHoraDesde      string
	NuevaHoraHasta      string
	NuevoTipo           string
	NuevoNumeroTelefono string
	NuevoServicio       string
	NuevoPrecio         *float64
	NuevasNotas         string
}

func (s *ReservasPGService) ActualizarReserva(input ActualizarReservaPGInput) error {
	upd := repository.UpdateReservaInput{Id: input.Id}

	if input.NuevaFecha != "" {
		t, err := time.Parse("2006-01-02", input.NuevaFecha)
		if err != nil {
			return fmt.Errorf("formato de fecha inválido")
		}
		upd.NuevaFecha = &t
	}

	if input.NuevaHoraDesde != "" {
		upd.NuevaHoraDesde = &input.NuevaHoraDesde
		if input.NuevaHoraHasta == "" {
			hHasta := sumar60Min(input.NuevaHoraDesde)
			upd.NuevaHoraHasta = &hHasta
		}
	}
	if input.NuevaHoraHasta != "" {
		upd.NuevaHoraHasta = &input.NuevaHoraHasta
	}

	if input.NuevoServicio != "" {
		upd.NuevoServicio = &input.NuevoServicio
	}

	if input.NuevoNumeroTelefono != "" {
		upd.NuevoNumeroTelefono = &input.NuevoNumeroTelefono
	}

	if input.NuevoTipo != "" {

		t := strings.ToUpper(strings.TrimSpace(input.NuevoTipo))
		upd.NuevoTipo = &t
	}

	if input.NuevoPrecio != nil {
		upd.NuevoPrecio = input.NuevoPrecio
	}

	/* COMENTADO HASTA QUE FRONTEND RECUPERE SERVICIOS DE LA BD
	if strings.TrimSpace(input.NuevoServicio) != "" {

		servicio, err := s.serviciosRepo.GetServicioByNombre(input.NuevoServicio)
		if err != nil {
			return err
		}

		upd.NuevoServicio = &input.NuevoServicio

		tipoFinal := strings.TrimSpace(servicio.TipoEspacio)

		if tipoFinal == "" {
			tipoFinal = strings.TrimSpace(input.NuevoTipo)
		}

		if tipoFinal != "" {
			tipoFinal = strings.ToUpper(tipoFinal)
			upd.NuevoTipo = &tipoFinal
		}

		if input.NuevoPrecio == nil {

			var precio float64

			_, err := fmt.Sscanf(servicio.Costo, "%f", &precio)
			if err != nil {
				return fmt.Errorf("Error al convertir precio del servicio")
			}

			upd.NuevoPrecio = &precio

		} else {

			upd.NuevoPrecio = input.NuevoPrecio
		}

	} else {

		if input.NuevoTipo != "" {

			t := strings.ToUpper(input.NuevoTipo)
			upd.NuevoTipo = &t
		}

		if input.NuevoPrecio != nil {
			upd.NuevoPrecio = input.NuevoPrecio
		}
	}
	*/

	if input.NuevasNotas != "" {
		upd.NuevasNotas = &input.NuevasNotas
	}

	// 1. Obtener datos actuales para completar campos faltantes en la validación
	current, err := s.repo.GetReservaByID(input.Id)
	if err != nil {
		return fmt.Errorf("No se pudo recuperar la reserva actual: %v", err)
	}
	estadoActual, err := estadoReservaActual(current)
	if err != nil {
		return err
	}
	if estadoActual == "AGENDADO" {
		return errors.New("no se puede editar una reserva con estado AGENDADO")
	}

	// 1.1 coherencia de horas
	horaDesdeFinal := current.HoraDesde
	if upd.NuevaHoraDesde != nil {
		horaDesdeFinal = *upd.NuevaHoraDesde
	}

	horaHastaFinal := current.HoraHasta
	if upd.NuevaHoraHasta != nil {
		horaHastaFinal = *upd.NuevaHoraHasta
	}

	parseHora := func(h string) (time.Time, error) {

		if len(h) > 5 {
			h = h[:5]
		}

		t, err := time.Parse("15:04", h)
		if err != nil {
			t, err = time.Parse("15:4", h)
		}

		return t, err
	}

	hDesde, err := parseHora(horaDesdeFinal)
	if err != nil {
		return fmt.Errorf("Hora de inicio inválida")
	}

	hHasta, err := parseHora(horaHastaFinal)
	if err != nil {
		return fmt.Errorf("Hora de finalización inválida:")
	}

	if !hDesde.Before(hHasta) {
		return fmt.Errorf("La hora de inicio no puede ser igual o posterior a la hora de finalización")
	}

	// Campos para validación (prioridad al nuevo valor, fallback al actual)
	valLocal := input.Local
	if valLocal == "" {
		valLocal = current.LocalNombre
	}

	valFecha := upd.NuevaFecha
	if valFecha == nil {
		valFecha = &current.Fecha
	}

	valHoraDesde := input.NuevaHoraDesde
	if valHoraDesde == "" {
		valHoraDesde = current.HoraDesde
	}

	valHoraHasta := input.NuevaHoraHasta
	if valHoraHasta == "" {
		valHoraHasta = current.HoraHasta
	}

	valTipo := input.NuevoTipo
	if valTipo == "" {
		valTipo = current.TipoEspacio
	}

	// 2. Ejecutar actualización en BD
	return s.repo.UpdateReserva(upd)
}

func (s *ReservasPGService) validarDisponibilidad(local string, fecha *time.Time, horaDesde, horaHasta, tipo string, excludeID *int) error {
	if local == "" || fecha == nil || horaDesde == "" || horaHasta == "" || tipo == "" {
		return nil // No hay suficiente info para validar, asumo ok o se validará en repo
	}

	// Obtener capacidades del local
	caps, err := s.repo.GetCapacidades(local)
	if err != nil {
		return err
	}
	var capacidad int
	tipoL := tipoNombreALetra(tipo)
	for _, c := range caps {
		if strings.EqualFold(c.TipoEspacio, tipoL) {
			capacidad = c.Capacidad
			break
		}
	}
	if capacidad == 0 {
		capacidad = 3 // fallback
	}

	// Obtener reservas existentes para ese día
	f := repository.FiltroReservasPG{
		LocalNombre: local,
		FechaDesde:  fecha,
		FechaHasta:  fecha,
		TipoEspacio: tipoL,
		SoloActivas: true,
	}
	reservas, err := s.repo.GetReservas(f)
	if err != nil {
		return err
	}

	// Contar ocupación en el rango solicitado
	// Convertir horas a minutos para comparar rangos
	toMin := func(h string) int {
		var hh, mm int
		fmt.Sscanf(h, "%d:%d", &hh, &mm)
		return hh*60 + mm
	}
	reqInicio := toMin(horaDesde)
	reqFin := toMin(horaHasta)

	// Mapa de minutos para contar solapamientos
	// Como son bloques de 60 min, podemos simplificar o usar un contador por slot
	maxOcupados := 0
	// Revisamos cada bloque de 1 minuto en el rango solicitado (o simplemente slots de 60 si son fijos)
	// Pero para ser robustos ante cualquier solapamiento:
	for m := reqInicio; m < reqFin; m++ {
		ocupadosEnMinuto := 0
		for _, r := range reservas {
			if excludeID != nil && r.ID == *excludeID {
				continue
			}
			rInicio := toMin(r.HoraDesde)
			rFin := toMin(r.HoraHasta)
			if m >= rInicio && m < rFin {
				ocupadosEnMinuto++
			}
		}
		if ocupadosEnMinuto > maxOcupados {
			maxOcupados = ocupadosEnMinuto
		}
	}

	if maxOcupados >= capacidad {
		return fmt.Errorf("no hay espacios disponibles de tipo '%s' en ese horario (%d/%d ocupados)", tipoL, maxOcupados, capacidad)
	}

	return nil
}

// Helpers

func tipoNombreALetra(nombre string) string {
	switch strings.ToLower(nombre) {
	case "mesa":
		return "M"
	case "bicicleta":
		return "B"
	}
	return strings.ToUpper(nombre)
}

func sumar60Min(hora string) string {
	t, err := time.Parse("15:04", hora)
	if err != nil {
		t, _ = time.Parse("15:4", hora)
	}
	t = t.Add(60 * time.Minute)
	return fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
}

func NormalizarEstadoReserva(raw string) (string, error) {
	estado := strings.ToUpper(strings.TrimSpace(raw))
	if estado == "" {
		return "PENDIENTE", nil
	}

	switch estado {
	case "PENDIENTE", "RECHAZADO", "AGENDADO":
		return estado, nil
	default:
		return "", fmt.Errorf("estado invalido, valores permitidos: PENDIENTE, RECHAZADO, AGENDADO")
	}
}

func filterReservasPorEstado(reservas []models.ReservaPGCompleta, estado string) []models.ReservaPGCompleta {
	if strings.TrimSpace(estado) == "" {
		return reservas
	}

	resultado := make([]models.ReservaPGCompleta, 0, len(reservas))
	for _, rv := range reservas {
		if rv.Estado != nil && strings.EqualFold(strings.TrimSpace(*rv.Estado), estado) {
			resultado = append(resultado, rv)
		}
	}
	return resultado
}

func filterReservasOcupadasPorEstado(reservas []models.ReservaPGCompleta, estado string) []models.ReservaPGCompleta {
	resultado := make([]models.ReservaPGCompleta, 0, len(reservas))
	for _, rv := range filterReservasPorEstado(reservas, estado) {
		if strings.TrimSpace(rv.Cliente) != "" {
			resultado = append(resultado, rv)
		}
	}
	return resultado
}

func estadoReservaActual(rv *models.ReservaPGCompleta) (string, error) {
	if rv == nil || rv.Estado == nil {
		return "PENDIENTE", nil
	}
	return NormalizarEstadoReserva(*rv.Estado)
}

func canTransitionReservaEstado(actual, siguiente string) bool {
	if actual == siguiente {
		return true
	}

	switch actual {
	case "PENDIENTE":
		return siguiente == "RECHAZADO" || siguiente == "AGENDADO"
	case "RECHAZADO":
		return siguiente == "PENDIENTE"
	case "AGENDADO":
		return false
	default:
		return false
	}
}
