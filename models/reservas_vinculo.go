package models

import "time"

// ReservaPorVincularPG es una reserva vigente que todavia no tiene cliente registrado.
type ReservaPorVincularPG struct {
	ID             int       `db:"id" json:"id" example:"41"`
	Fecha          time.Time `db:"fecha" json:"fecha"`
	HoraDesde      string    `db:"hora_desde" json:"hora_desde" example:"10:00:00"`
	HoraHasta      string    `db:"hora_hasta" json:"hora_hasta" example:"11:00:00"`
	LocalNombre    string    `db:"local_nombre" json:"local" example:"SAN MARTIN"`
	Cliente        string    `db:"cliente" json:"cliente" example:"Lucia Paredes"`
	NumeroTelefono *string   `db:"numero_telefono" json:"numero_telefono,omitempty" example:"+59171235566"`
	TelefonoE164   *string   `db:"telefono_e164" json:"telefono_e164,omitempty" example:"+59171235566"`
	Servicio       *string   `db:"servicio_nombre" json:"servicio,omitempty" example:"Depilacion laser"`
	Estado         *string   `db:"estado" json:"estado,omitempty" example:"PENDIENTE"`
}

// ClienteSugeridoPG es un cliente que podria ser el dueño de una reserva por vincular. Es solo una pista:
// el cajero decide.
type ClienteSugeridoPG struct {
	ClientePG
	// PorTelefono: el telefono coincide (comparando los ultimos 8 digitos).
	PorTelefono bool `db:"por_telefono" json:"por_telefono"`
	// PorNombre: el nombre completo coincide, sin distinguir mayusculas ni acentos.
	PorNombre bool `db:"por_nombre" json:"por_nombre"`
}

// ResultadoVinculo cuenta cuantas reservas quedaron enlazadas con el cliente.
type ResultadoVinculo struct {
	Vinculadas int `json:"vinculadas" example:"3"`
}
