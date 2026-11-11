package services

import (
	"errors"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type JornadaInput struct {
	Fecha     *time.Time
	LocalID   *int
	Estado    string
	Q         string
	PageLimit int
	Offset    int
}

var estadosJornada = map[string]bool{
	"PROGRAMADA":       true,
	"PENDIENTE_CIERRE": true,
	"ATENDIDA":         true,
	"CANCELADA":        true,
}

// EstadoJornadaValido normaliza y valida un estado operativo de la jornada.
func EstadoJornadaValido(estado string) bool {
	return estadosJornada[strings.ToUpper(strings.TrimSpace(estado))]
}

// Jornada arma la vista operativa del dia: una fila por reserva, estado
// derivado con la zona horaria configurada. El filtro de estado afecta las
// filas, pero el resumen conserva el total del día para operar como filtro.
func (s *AtencionesService) Jornada(in JornadaInput) (*models.JornadaResponse, error) {
	cfg, err := s.config.Snapshot()
	if err != nil {
		return nil, err
	}
	hoy := cfg.Hoy()
	fecha := hoy
	if in.Fecha != nil {
		fecha = *in.Fecha
	}
	estado := strings.ToUpper(strings.TrimSpace(in.Estado))
	if estado != "" && !estadosJornada[estado] {
		return nil, errors.New("estado invalido, use PROGRAMADA, PENDIENTE_CIERRE, ATENDIDA o CANCELADA")
	}
	loc := cfg.Ubicacion()
	ahora := time.Now().In(loc)
	filas, err := s.repo.Jornada(repository.FiltroJornada{
		Fecha: fecha, LocalID: in.LocalID, Estado: estado, Q: strings.TrimSpace(in.Q),
		Ahora: ahora, PageLimit: in.PageLimit, Offset: in.Offset,
	})
	if err != nil {
		return nil, err
	}
	items := make([]models.JornadaAtencion, 0, len(filas))
	for _, fila := range filas {
		estadoOperativo := derivarEstadoJornada(fila, ahora, loc)
		if estadoOperativo == "" {
			continue
		}
		items = append(items, models.JornadaAtencion{
			ReservaID:             fila.ReservaID,
			Fecha:                 fecha.Format("2006-01-02"),
			HoraDesde:             fila.HoraDesde,
			HoraHasta:             fila.HoraHasta,
			Cliente:               fila.Cliente,
			Servicio:              fila.Servicio,
			LocalID:               fila.LocalID,
			LocalNombre:           fila.LocalNombre,
			EstadoReserva:         fila.EstadoReserva,
			MotivoCancelacionTipo: fila.MotivoCancelacionTipo,
			EstadoOperativo:       estadoOperativo,
			AtencionID:            fila.AtencionID,
			EmpleadoNombre:        fila.EmpleadoNombre,
			InsumosRegistrados:    fila.InsumosRegistrados,
			InsumosResumen:        fila.InsumosResumen,
			InsumosPendientes:     fila.InsumosPendientes,
			PuedeAtender:          estadoOperativo == "PROGRAMADA" || estadoOperativo == "PENDIENTE_CIERRE",
			PuedeVerAtencion:      fila.AtencionID != nil,
			PuedeEditarAtencion:   fila.AtencionID != nil && fila.InsumosPendientes > 0,
		})
	}
	return &models.JornadaResponse{
		Fecha: fecha.Format("2006-01-02"),
		Hoy:   hoy.Format("2006-01-02"),
		Items: items,
	}, nil
}

// ResumenJornada calcula los contadores completos de una fecha/local. Es una
// consulta distinta de la lista: estado y búsqueda nunca lo reducen.
func (s *AtencionesService) ResumenJornada(fechaIn *time.Time, localID *int) (*models.JornadaResponse, error) {
	cfg, err := s.config.Snapshot()
	if err != nil {
		return nil, err
	}
	hoy := cfg.Hoy()
	fecha := hoy
	if fechaIn != nil {
		fecha = *fechaIn
	}
	loc := cfg.Ubicacion()
	ahora := time.Now().In(loc)
	filas, err := s.repo.Jornada(repository.FiltroJornada{Fecha: fecha, LocalID: localID, Ahora: ahora, PageLimit: -1})
	if err != nil {
		return nil, err
	}
	resumen := models.JornadaResumen{}
	for _, fila := range filas {
		switch derivarEstadoJornada(fila, ahora, loc) {
		case "PROGRAMADA":
			resumen.Programadas++
		case "ATENDIDA":
			resumen.Atendidas++
		case "PENDIENTE_CIERRE":
			resumen.PendientesCierre++
		case "CANCELADA":
			resumen.Canceladas++
			if fila.MotivoCancelacionTipo != nil && *fila.MotivoCancelacionTipo == "INASISTENCIA" {
				resumen.NoAsistieron++
			}
		}
		if fila.InsumosPendientes > 0 {
			resumen.DeduccionesPendientes++
		}
	}
	return &models.JornadaResponse{Fecha: fecha.Format("2006-01-02"), Hoy: hoy.Format("2006-01-02"), Resumen: resumen}, nil
}

func (s *AtencionesService) ContarJornada(in JornadaInput) (int, error) {
	cfg, err := s.config.Snapshot()
	if err != nil {
		return 0, err
	}
	fecha := cfg.Hoy()
	if in.Fecha != nil {
		fecha = *in.Fecha
	}
	estado := strings.ToUpper(strings.TrimSpace(in.Estado))
	if estado != "" && !estadosJornada[estado] {
		return 0, errors.New("estado invalido")
	}
	return s.repo.ContarJornada(repository.FiltroJornada{
		Fecha: fecha, LocalID: in.LocalID, Estado: estado, Q: strings.TrimSpace(in.Q), Ahora: time.Now().In(cfg.Ubicacion()),
	})
}

func derivarEstadoJornada(fila repository.JornadaRow, ahora time.Time, loc *time.Location) string {
	if fila.AtencionID != nil {
		return "ATENDIDA"
	}
	switch strings.ToUpper(strings.TrimSpace(fila.EstadoReserva)) {
	case "AGENDADO":
		if jornadaFinalizada(fila, ahora, loc) {
			return "PENDIENTE_CIERRE"
		}
		return "PROGRAMADA"
	case "CANCELADO":
		return "CANCELADA"
	default:
		return ""
	}
}

// jornadaFinalizada compara el fin del bloque con la hora actual en la zona
// horaria configurada, sin depender de la zona del servidor.
func jornadaFinalizada(fila repository.JornadaRow, ahora time.Time, loc *time.Location) bool {
	hora, err := time.Parse("15:04", strings.TrimSpace(fila.HoraHasta))
	if err != nil {
		return false
	}
	fin := time.Date(fila.Fecha.Year(), fila.Fecha.Month(), fila.Fecha.Day(), hora.Hour(), hora.Minute(), 0, 0, loc)
	return !ahora.Before(fin)
}
