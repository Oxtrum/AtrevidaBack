package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"atrevida-agenda-api/services"

	"github.com/gin-gonic/gin"
)

func TestRespondReservaDuplicadaDevuelve409ConLasReservasExistentes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	respondReservaDuplicada(c, &services.ReservaDuplicadaError{Reservas: []services.ReservaDuplicadaItem{
		{ID: 3, Local: "SAN MARTIN", HoraDesde: "10:00", HoraHasta: "11:00", Servicio: "Limpieza facial", Estado: "AGENDADO"},
	}})

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
	var body struct {
		Error bool `json:"error"`
		Data  struct {
			Codigo   string `json:"codigo"`
			Reservas []struct {
				HoraDesde string `json:"hora_desde"`
				Servicio  string `json:"servicio"`
			} `json:"reservas"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Error || body.Data.Codigo != "RESERVA_DUPLICADA" {
		t.Fatalf("body = %s, want error=true y codigo RESERVA_DUPLICADA", w.Body.String())
	}
	if len(body.Data.Reservas) != 1 || body.Data.Reservas[0].HoraDesde != "10:00" || body.Data.Reservas[0].Servicio != "Limpieza facial" {
		t.Fatalf("reservas = %+v", body.Data.Reservas)
	}
}
