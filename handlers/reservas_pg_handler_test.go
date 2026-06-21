package handlers

import (
	"testing"
	"time"

	"atrevida-agenda-api/services"
)

func TestBuildReservaResumenSemanaResponseDomingoIncluyeSabado(t *testing.T) {
	fechaDomingo := time.Date(2026, time.May, 24, 0, 0, 0, 0, time.UTC)
	resp := buildReservaResumenSemanaResponse(fechaDomingo, services.ResumenReservasSemana{
		TotalReservas: 21,
		Lunes:         1,
		Martes:        2,
		Miercoles:     3,
		Jueves:        4,
		Viernes:       5,
		Sabado:        6,
	})

	if resp.Sabado == nil || *resp.Sabado != 6 {
		t.Fatalf("Sabado = %v, want 6", resp.Sabado)
	}
}
