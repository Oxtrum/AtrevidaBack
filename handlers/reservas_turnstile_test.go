package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"atrevida-agenda-api/services"

	"github.com/gin-gonic/gin"
)

const reservaPublicaBody = `{"local":"SAN MARTIN","fecha":"2026-05-23","hora_desde":"15:00","cliente":"Maria","numero_telefono":"70011223"%s}`

func postReservaPublica(t *testing.T, h *Container, extra string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/bd/reservas", strings.NewReader(strings.Replace(reservaPublicaBody, "%s", extra, 1)))
	c.Request.Header.Set("Content-Type", "application/json")
	h.PostReservaPG(c)
	return w
}

func cloudflareQueResponde(t *testing.T, respuesta string) *services.TurnstileVerifier {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(respuesta))
	}))
	t.Cleanup(srv.Close)
	return &services.TurnstileVerifier{
		Secret: "s", AllowedHostnames: []string{"atrevida.test"}, Action: "reservation_create", Endpoint: srv.URL,
	}
}

// La reserva publica (sin token de usuario) debe traer un token Turnstile valido; con el token invalido
// no se llega a crear nada (el servicio de reservas ni siquiera esta configurado en estas pruebas).
func TestPostReservaPublicaSinTokenTurnstileDa400(t *testing.T) {
	h := &Container{Turnstile: cloudflareQueResponde(t, `{"success":true,"action":"reservation_create","hostname":"atrevida.test"}`)}
	w := postReservaPublica(t, h, "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, quiero 400; body = %s", w.Code, w.Body.String())
	}
}

func TestPostReservaPublicaConTokenRechazadoDa403(t *testing.T) {
	h := &Container{Turnstile: cloudflareQueResponde(t, `{"success":false,"error-codes":["invalid-input-response"]}`)}
	w := postReservaPublica(t, h, `,"cf-turnstile-response":"tok"`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, quiero 403; body = %s", w.Code, w.Body.String())
	}
}

func TestPostReservaPublicaSinTurnstileConfiguradoDa503(t *testing.T) {
	h := &Container{}
	w := postReservaPublica(t, h, `,"cf-turnstile-response":"tok"`)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, quiero 503; body = %s", w.Code, w.Body.String())
	}
}
