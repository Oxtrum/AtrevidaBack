package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"atrevida-agenda-api/services"

	"github.com/gin-gonic/gin"
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

func TestQueryLocalNombreDecodificaEspacios(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		rawQuery string
		want     string
	}{
		{
			name:     "espacio codificado como percent encoding",
			rawQuery: "local=SAN%20MARTIN",
			want:     "SAN MARTIN",
		},
		{
			name:     "espacio codificado como plus",
			rawQuery: "local=SAN+MARTIN",
			want:     "SAN MARTIN",
		},
		{
			name:     "sin local",
			rawQuery: "fecha=2026-07-02",
			want:     "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/bd/reservas/resumen?"+tc.rawQuery, nil)

			if got := queryLocalNombre(c); got != tc.want {
				t.Fatalf("queryLocalNombre() = %q, want %q", got, tc.want)
			}
		})
	}
}
