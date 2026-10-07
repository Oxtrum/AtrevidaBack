package services

import (
	"testing"

	"atrevida-agenda-api/models"
)

func TestReservaOcupaCapacidad(t *testing.T) {
	estado := func(valor string) *string { return &valor }

	casos := []struct {
		nombre string
		valor  *string
		ocupa  bool
	}{
		{nombre: "pendiente", valor: estado("PENDIENTE"), ocupa: true},
		{nombre: "agendado", valor: estado("AGENDADO"), ocupa: true},
		{nombre: "rechazado", valor: estado("RECHAZADO"), ocupa: false},
		{nombre: "completado", valor: estado("COMPLETADO"), ocupa: false},
		{nombre: "estado final normalizado", valor: estado(" completado "), ocupa: false},
		{nombre: "legacy nulo", valor: nil, ocupa: true},
		{nombre: "legacy desconocido", valor: estado("OTRO"), ocupa: true},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			reserva := models.ReservaPGCompleta{ReservaPG: models.ReservaPG{Estado: caso.valor}}
			if obtenido := reservaOcupaCapacidad(reserva); obtenido != caso.ocupa {
				t.Fatalf("reservaOcupaCapacidad() = %v; se esperaba %v", obtenido, caso.ocupa)
			}
		})
	}
}

func TestFilterReservasQueOcupanCapacidad(t *testing.T) {
	pendiente, rechazado, completado := "PENDIENTE", "RECHAZADO", "COMPLETADO"
	reservas := []models.ReservaPGCompleta{
		{ReservaPG: models.ReservaPG{ID: 1, Estado: &pendiente}},
		{ReservaPG: models.ReservaPG{ID: 2, Estado: &rechazado}},
		{ReservaPG: models.ReservaPG{ID: 3, Estado: &completado}},
		{ReservaPG: models.ReservaPG{ID: 4, Estado: nil}},
	}

	filtradas := filterReservasQueOcupanCapacidad(reservas)
	if len(filtradas) != 2 || filtradas[0].ID != 1 || filtradas[1].ID != 4 {
		t.Fatalf("reservas ocupantes inesperadas: %+v", filtradas)
	}
}
