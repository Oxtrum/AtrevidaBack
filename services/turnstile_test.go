package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// cloudflareFalso responde como siteverify y deja ver lo que recibio.
func cloudflareFalso(t *testing.T, respuesta string, recibido *url.Values) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("form invalido: %v", err)
		}
		if recibido != nil {
			*recibido = r.PostForm
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(respuesta))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func verificadorDePrueba(endpoint string) *TurnstileVerifier {
	return &TurnstileVerifier{
		Secret:           "secreto",
		AllowedHostnames: []string{"atrevida.test"},
		Action:           "reservation_create",
		Endpoint:         endpoint,
	}
}

func TestTurnstileAceptaTokenValidoYEnviaSecretoTokenEIP(t *testing.T) {
	var recibido url.Values
	srv := cloudflareFalso(t, `{"success":true,"action":"reservation_create","hostname":"atrevida.test"}`, &recibido)

	if err := verificadorDePrueba(srv.URL).Verify(context.Background(), "tok-1", "203.0.113.9"); err != nil {
		t.Fatalf("Verify = %v, quiero nil", err)
	}
	if recibido.Get("secret") != "secreto" || recibido.Get("response") != "tok-1" || recibido.Get("remoteip") != "203.0.113.9" {
		t.Fatalf("form recibido = %v", recibido)
	}
}

func TestTurnstileRechazaTokenVacioSinLlamarACloudflare(t *testing.T) {
	llamado := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { llamado = true }))
	defer srv.Close()

	err := verificadorDePrueba(srv.URL).Verify(context.Background(), "  ", "")
	if !errors.Is(err, ErrTurnstileTokenRequerido) {
		t.Fatalf("Verify = %v, quiero ErrTurnstileTokenRequerido", err)
	}
	if llamado {
		t.Fatal("un token vacio no debe consultar a Cloudflare")
	}
}

func TestTurnstileRechazaTokenDemasiadoLargoSinLlamarACloudflare(t *testing.T) {
	llamado := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { llamado = true }))
	defer srv.Close()

	err := verificadorDePrueba(srv.URL).Verify(context.Background(), strings.Repeat("a", 2049), "")
	if !errors.Is(err, ErrTurnstileRechazado) {
		t.Fatalf("Verify = %v, quiero ErrTurnstileRechazado", err)
	}
	if llamado {
		t.Fatal("un token de mas de 2048 caracteres no debe consultar a Cloudflare")
	}
}

func TestTurnstileRechazaRespuestasInvalidas(t *testing.T) {
	casos := map[string]string{
		"success false":      `{"success":false,"error-codes":["invalid-input-response"]}`,
		"accion distinta":    `{"success":true,"action":"otra","hostname":"atrevida.test"}`,
		"hostname ajeno":     `{"success":true,"action":"reservation_create","hostname":"malo.test"}`,
		"hostname ausente":   `{"success":true,"action":"reservation_create"}`,
		"respuesta invalida": `no es json`,
	}
	for nombre, respuesta := range casos {
		t.Run(nombre, func(t *testing.T) {
			srv := cloudflareFalso(t, respuesta, nil)
			err := verificadorDePrueba(srv.URL).Verify(context.Background(), "tok", "")
			if !errors.Is(err, ErrTurnstileRechazado) {
				t.Fatalf("Verify = %v, quiero ErrTurnstileRechazado", err)
			}
		})
	}
}

func TestTurnstileRechazoIncluyeElMotivo(t *testing.T) {
	casos := map[string]struct {
		respuesta string
		esperado  []string
	}{
		"codigo de cloudflare": {`{"success":false,"error-codes":["invalid-input-secret"]}`, []string{"invalid-input-secret"}},
		"accion distinta":      {`{"success":true,"action":"otra","hostname":"atrevida.test"}`, []string{`action="otra"`, `reservation_create`}},
		"hostname ajeno":       {`{"success":true,"action":"reservation_create","hostname":"malo.test"}`, []string{`hostname="malo.test"`, `atrevida.test`}},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			srv := cloudflareFalso(t, caso.respuesta, nil)
			err := verificadorDePrueba(srv.URL).Verify(context.Background(), "tok", "")
			if !errors.Is(err, ErrTurnstileRechazado) {
				t.Fatalf("Verify = %v, quiero ErrTurnstileRechazado", err)
			}
			for _, trozo := range caso.esperado {
				if !strings.Contains(err.Error(), trozo) {
					t.Fatalf("error %q no menciona %q", err.Error(), trozo)
				}
			}
		})
	}
}

func TestTurnstileSinConfiguracionFallaCerrado(t *testing.T) {
	srv := cloudflareFalso(t, `{"success":true,"action":"reservation_create","hostname":"atrevida.test"}`, nil)

	sinSecreto := verificadorDePrueba(srv.URL)
	sinSecreto.Secret = ""
	if err := sinSecreto.Verify(context.Background(), "tok", ""); !errors.Is(err, ErrTurnstileNoConfigurado) {
		t.Fatalf("sin secreto: Verify = %v, quiero ErrTurnstileNoConfigurado", err)
	}

	sinHostnames := verificadorDePrueba(srv.URL)
	sinHostnames.AllowedHostnames = nil
	if err := sinHostnames.Verify(context.Background(), "tok", ""); !errors.Is(err, ErrTurnstileNoConfigurado) {
		t.Fatalf("sin hostnames: Verify = %v, quiero ErrTurnstileNoConfigurado", err)
	}

	var nulo *TurnstileVerifier
	if err := nulo.Verify(context.Background(), "tok", ""); !errors.Is(err, ErrTurnstileNoConfigurado) {
		t.Fatalf("verificador nil: Verify = %v, quiero ErrTurnstileNoConfigurado", err)
	}
}

func TestTurnstileDesactivadoPermiteSinConsultar(t *testing.T) {
	v := &TurnstileVerifier{Disabled: true}
	if err := v.Verify(context.Background(), "", ""); err != nil {
		t.Fatalf("Verify = %v, quiero nil con Disabled", err)
	}
}

func TestTurnstileCloudflareCaidoNoDejaPasar(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer srv.Close()

	err := verificadorDePrueba(srv.URL).Verify(context.Background(), "tok", "")
	if err == nil || errors.Is(err, ErrTurnstileRechazado) {
		t.Fatalf("Verify = %v, quiero un error de servicio distinto de rechazo", err)
	}
}
