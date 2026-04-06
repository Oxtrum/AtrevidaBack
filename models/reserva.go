package models

type ReservaItem struct {
	Tipo     string `json:"tipo"`
	Cliente  string `json:"cliente,omitempty"`
	Servicio string `json:"servicio,omitempty"`
}

type ReservaSlot struct {
	Hora string                   `json:"hora"`
	Dias map[string][]ReservaItem `json:"dias"`
}

type Semana struct {
	Titulo   string        `json:"titulo"`
	Reservas []ReservaSlot `json:"reservas"`
}

type LocalReservas struct {
	Local   string   `json:"local"`
	Semanas []Semana `json:"semanas"`
}
