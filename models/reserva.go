package models

type ReservaItem struct {
	Tipo               string `json:"tipo" example:"M"`
	Cliente            string `json:"cliente,omitempty" example:"Maria Lopez"`
	Servicio           string `json:"servicio,omitempty" example:"Depilacion Laser"`
	ServicioSolicitado string `json:"servicio_solicitado,omitempty" example:"Piernas completas"`
	ServicioConfirmado string `json:"servicio_confirmado,omitempty" example:"Depilacion Laser Piernas"`
	Estado             string `json:"estado,omitempty" example:"AGENDADO"`
	NumeroTelefono     string `json:"numero_telefono,omitempty" example:"+59170011223"`
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
