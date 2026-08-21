package models

type ReservaItem struct {
	// ID de la reserva; permite abrir el detalle desde el calendario.
	ID      int    `json:"id" example:"100"`
	Tipo    string `json:"tipo" example:"M"`
	Cliente string `json:"cliente,omitempty" example:"Maria Lopez"`
	// Local, fecha y horario para reconstruir la reserva en el detalle.
	Local     string `json:"local,omitempty" example:"SAN MARTIN"`
	Fecha     string `json:"fecha,omitempty" example:"2026-05-23"`
	HoraDesde string `json:"hora_desde,omitempty" example:"09:00"`
	HoraHasta string `json:"hora_hasta,omitempty" example:"10:00"`
	// HoraHastaReal es la hora_hasta verdadera de la reserva cuando se parte
	// en slots de 30 min para la rejilla. HoraHasta contiene el fin del slot
	// (p.ej. "16:30"), HoraHastaReal el fin real de la reserva (p.ej. "17:00").
	HoraHastaReal string `json:"reserva_hora_hasta,omitempty" example:"17:00"`
	// ID del plan asociado, si la reserva consume un paquete.
	PlanID             *int   `json:"plan_id,omitempty" example:"5"`
	Servicio           string `json:"servicio,omitempty" example:"Depilacion Laser"`
	ServicioSolicitado string `json:"servicio_solicitado,omitempty" example:"Piernas completas"`
	ServicioConfirmado string `json:"servicio_confirmado,omitempty" example:"Depilacion Laser Piernas"`
	Estado             string `json:"estado,omitempty" example:"AGENDADO"`
	NumeroTelefono     string `json:"numero_telefono,omitempty" example:"+59170011223"`
	TelefonoE164       string `json:"telefono_e164,omitempty" example:"+59170011223"`
	Notificado         bool   `json:"notificado" example:"false"`
	CreadoEn           string `json:"creado_en,omitempty" example:"2026-05-23T15:04:05Z"`
	ActualizadoEn      string `json:"actualizado_en,omitempty" example:"2026-05-23T16:04:05Z"`
}

type ReservaSlot struct {
	Hora string                   `json:"hora" example:"09:00"`
	Dias map[string][]ReservaItem `json:"dias"`
}

type Semana struct {
	Titulo   string        `json:"titulo" example:"Semana 22"`
	Reservas []ReservaSlot `json:"reservas"`
}

type LocalReservas struct {
	Local   string   `json:"local" example:"SAN MARTIN"`
	Semanas []Semana `json:"semanas"`
}
