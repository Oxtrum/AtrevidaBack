package models

type ReservaItem struct {
	Tipo               string `json:"tipo" example:"M"`
	Cliente            string `json:"cliente,omitempty" example:"Maria Lopez"`
	Servicio           string `json:"servicio,omitempty" example:"Depilacion Laser"`
	ServicioSolicitado string `json:"servicio_solicitado,omitempty" example:"Piernas completas"`
	ServicioConfirmado string `json:"servicio_confirmado,omitempty" example:"Depilacion Laser Piernas"`
	Estado             string `json:"estado,omitempty" example:"AGENDADO"`
	NumeroTelefono     string `json:"numero_telefono,omitempty" example:"+59170011223"`
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
