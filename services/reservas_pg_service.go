package services

import (
	"fmt"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

type ReservasPGService struct {
	repo repository.ReservasPGRepository
}

func NewReservasPGService(repo repository.ReservasPGRepository) *ReservasPGService {
	return &ReservasPGService{repo: repo}
}

// GET

type FiltroReservasPG struct {
	Local      string
	Fecha      string
	FechaDesde string
	FechaHasta string
	Cliente    string
	Tipo       string
	Reservados *bool
}

func (s *ReservasPGService) GetReservasFiltradas(f FiltroReservasPG) ([]models.LocalReservas, error) {

	if f.Cliente != "" {
		soloOcupados := true
		f.Reservados = &soloOcupados
	}

	if f.Reservados != nil && !*f.Reservados {
		desde, hasta := getRangoTiempoDisp(f.FechaDesde, f.FechaHasta)
		return s.getEspaciosDisponibles(f, desde, hasta)
	}

	filtro := repository.FiltroReservasPG{
		LocalNombre: f.Local,
		Cliente:     f.Cliente,
		SoloActivas: true,
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
		var filtradas []models.ReservaPGCompleta
		for _, rv := range reservas {
			if strings.TrimSpace(rv.Cliente) != "" {
				filtradas = append(filtradas, rv)
			}
		}
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
				cantOcupados := len(ocupadosEnSlot)

				resultado = append(resultado, ocupadosEnSlot...)

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
		slots := partirEnSlots30(rv.HoraDesde, rv.HoraHasta)
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

// partirEnSlots30 divide un rango [desde, hasta] en bloques de 30 minutos.
func partirEnSlots30(desde, hasta string) [][2]string {
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
		siguiente := cur.Add(30 * time.Minute)
		if siguiente.After(tHasta) {
			siguiente = tHasta
		}
		slots = append(slots, [2]string{
			fmt.Sprintf("%d:%02d", cur.Hour(), cur.Minute()),
			fmt.Sprintf("%d:%02d", siguiente.Hour(), siguiente.Minute()),
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

// horarioLocal retorna los slots de 30 min disponibles para un día dado.
func horarioLocal(fecha time.Time) [][2]string {
	switch fecha.Weekday() {
	case time.Sunday:
		return nil
	case time.Saturday:
		return generarSlots("08:00", "15:30")
	default:
		return generarSlots("08:00", "20:00")
	}
}

// generarSlots produce pares [desde, hasta] de 30 min entre apertura y cierre.
func generarSlots(apertura, cierre string) [][2]string {
	t, _ := time.Parse("15:04", apertura)
	fin, _ := time.Parse("15:04", cierre)
	var slots [][2]string
	for t.Before(fin) {
		siguiente := t.Add(30 * time.Minute)
		if siguiente.After(fin) {
			break
		}
		slots = append(slots, [2]string{
			fmt.Sprintf("%d:%02d", t.Hour(), t.Minute()),
			fmt.Sprintf("%d:%02d", siguiente.Hour(), siguiente.Minute()),
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
	ID        int      `json:"id"`
	Local     string   `json:"local"`
	Tipo      string   `json:"tipo"`
	Fecha     string   `json:"fecha"`
	HoraDesde string   `json:"hora_desde"`
	HoraHasta string   `json:"hora_hasta"`
	Cliente   string   `json:"cliente"`
	Servicio  *string  `json:"servicio,omitempty"`
	Precio    *float64 `json:"precio,omitempty"`
	Notas     *string  `json:"notas,omitempty"`
}

type FiltroReservasSimple struct {
	Local      string
	Fecha      string
	FechaDesde string
	FechaHasta string
	Cliente    string
	Tipo       string
}

func (s *ReservasPGService) GetReservasSimple(f FiltroReservasSimple) ([]ReservaSimple, error) {
	filtro := repository.FiltroReservasPG{
		LocalNombre: f.Local,
		Cliente:     f.Cliente,
		SoloActivas: true,
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
	resultado := make([]ReservaSimple, 0, len(reservas))
	for _, rv := range reservas {
		resultado = append(resultado, ReservaSimple{
			ID:        rv.ID,
			Local:     rv.LocalNombre,
			Tipo:      tipoLetraANombreService(rv.TipoEspacio),
			Fecha:     rv.Fecha.Format("2006-01-02"),
			HoraDesde: formatHoraService(rv.HoraDesde),
			HoraHasta: formatHoraService(rv.HoraHasta),
			Cliente:   rv.Cliente,
			Servicio:  rv.ServicioNombre,
			Precio:    rv.Precio,
			Notas:     rv.Notas,
		})
	}
	return resultado, nil
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
	h = strings.TrimPrefix(h, "0")
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
	Servicio  string
	Precio    *float64
	Notas     string
	PlanID    *int
}

func (s *ReservasPGService) CrearReserva(input CrearReservaPGInput) error {
	fecha, err := time.Parse("2006-01-02", input.Fecha)
	if err != nil {
		return fmt.Errorf("formato de fecha inválido, use YYYY-MM-DD")
	}

	horaHasta := input.HoraHasta
	if horaHasta == "" {
		horaHasta = sumar30Min(input.HoraDesde)
	}

	_, err = s.repo.CreateReserva(repository.CreateReservaInput{
		LocalNombre:    input.Local,
		TipoEspacio:    strings.ToUpper(input.Tipo),
		Fecha:          fecha,
		HoraDesde:      input.HoraDesde,
		HoraHasta:      horaHasta,
		Cliente:        input.Cliente,
		ServicioNombre: input.Servicio,
		Precio:         input.Precio,
		Notas:          input.Notas,
		PlanID:         input.PlanID,
	})
	return err
}

// PATCH
type ActualizarReservaPGInput struct {
	Id        int
	Local     string
	Fecha     string
	HoraDesde string
	Tipo      string
	Cliente   string

	NuevaFecha     string
	NuevaHoraDesde string
	NuevaHoraHasta string
	NuevoTipo      string
	NuevoServicio  string
	NuevoPrecio    *float64
	NuevasNotas    string
}

func (s *ReservasPGService) ActualizarReserva(input ActualizarReservaPGInput) error {

	upd := repository.UpdateReservaInput{
		Id:          input.Id,
		LocalNombre: input.Local,
	}

	if input.NuevaFecha != "" {
		t, err := time.Parse("2006-01-02", input.NuevaFecha)
		if err != nil {
			return fmt.Errorf("formato de nueva_fecha inválido")
		}
		upd.NuevaFecha = &t
	}
	if input.NuevaHoraDesde != "" {
		upd.NuevaHoraDesde = &input.NuevaHoraDesde
	}
	if input.NuevaHoraHasta != "" {
		upd.NuevaHoraHasta = &input.NuevaHoraHasta
	}
	if input.NuevoTipo != "" {
		t := strings.ToUpper(input.NuevoTipo)
		upd.NuevoTipo = &t
	}
	if input.NuevoServicio != "" {
		upd.NuevoServicio = &input.NuevoServicio
	}
	if input.NuevoPrecio != nil {
		upd.NuevoPrecio = input.NuevoPrecio
	}
	if input.NuevasNotas != "" {
		upd.NuevasNotas = &input.NuevasNotas
	}

	return s.repo.UpdateReserva(upd)
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

func sumar30Min(hora string) string {
	t, err := time.Parse("15:04", hora)
	if err != nil {
		t, _ = time.Parse("15:4", hora)
	}
	t = t.Add(30 * time.Minute)
	return fmt.Sprintf("%d:%02d", t.Hour(), t.Minute())
}
