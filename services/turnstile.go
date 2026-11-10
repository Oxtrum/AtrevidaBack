package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	turnstileSiteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	turnstileMaxTokenLen   = 2048
)

var (
	// ErrTurnstileTokenRequerido: la peticion publica no trae el token del widget.
	ErrTurnstileTokenRequerido = errors.New("turnstile: falta el token de verificacion")
	// ErrTurnstileRechazado: Cloudflare no valida el token, o su accion u hostname no son los esperados.
	ErrTurnstileRechazado = errors.New("turnstile: verificacion fallida")
	// ErrTurnstileNoConfigurado: falta TURNSTILE_SECRET_KEY o TURNSTILE_ALLOWED_HOSTNAMES; se rechaza todo (falla cerrado).
	ErrTurnstileNoConfigurado = errors.New("turnstile: no esta configurado en el servidor")
)

// TurnstileVerifier valida con Cloudflare (siteverify) el token del widget del formulario publico.
// Un token solo sirve una vez: debe validarse en un unico lugar de la cadena (aqui, no tambien en el proxy del front).
type TurnstileVerifier struct {
	// Secret es la secret key del mismo widget cuya site key usa el front.
	Secret string
	// AllowedHostnames son los dominios desde los que se acepta el token (el hostname que informa Cloudflare).
	AllowedHostnames []string
	// Action es la accion con la que el front renderiza el widget.
	Action string
	// Disabled omite la verificacion; solo para desarrollo local (TURNSTILE_DISABLED=true).
	Disabled bool
	// Endpoint reemplaza la URL de siteverify (pruebas). Vacio = la de Cloudflare.
	Endpoint string
	// Client reemplaza el cliente HTTP (pruebas). Nil = uno con timeout corto.
	Client *http.Client
}

type turnstileRespuesta struct {
	Success    bool     `json:"success"`
	Action     string   `json:"action"`
	Hostname   string   `json:"hostname"`
	ErrorCodes []string `json:"error-codes"`
}

// Verify devuelve nil solo si Cloudflare confirma el token. remoteIP es opcional.
// Si Cloudflare no responde 2xx devuelve un error de servicio (no ErrTurnstileRechazado) y no deja pasar.
func (v *TurnstileVerifier) Verify(ctx context.Context, token, remoteIP string) error {
	if v != nil && v.Disabled {
		return nil
	}
	if v == nil || strings.TrimSpace(v.Secret) == "" || len(v.AllowedHostnames) == 0 {
		return ErrTurnstileNoConfigurado
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrTurnstileTokenRequerido
	}
	// Cloudflare documenta tokens de hasta 2048 caracteres: uno mas largo no puede ser valido.
	if len(token) > turnstileMaxTokenLen {
		return ErrTurnstileRechazado
	}

	form := url.Values{"secret": {v.Secret}, "response": {token}}
	if remoteIP = strings.TrimSpace(remoteIP); remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	endpoint := v.Endpoint
	if endpoint == "" {
		endpoint = turnstileSiteverifyURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("turnstile: no se pudo armar la consulta: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := v.Client
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("turnstile: no se pudo consultar a Cloudflare: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("turnstile: Cloudflare respondio %d", resp.StatusCode)
	}

	var r turnstileRespuesta
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return ErrTurnstileRechazado
	}
	if !r.Success || r.Action != v.Action || !v.hostnamePermitido(r.Hostname) {
		return ErrTurnstileRechazado
	}
	return nil
}

func (v *TurnstileVerifier) hostnamePermitido(hostname string) bool {
	if hostname == "" {
		return false
	}
	for _, permitido := range v.AllowedHostnames {
		if permitido == hostname {
			return true
		}
	}
	return false
}
