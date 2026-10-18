package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	repository "atrevida-agenda-api/repositories"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

// optionalAuth valida el token Bearer si viene, sin abortar la peticion cuando falta. Sirve para endpoints
// publicos que muestran mas informacion a usuarios del panel (ej. crear reserva).
func (h *Container) optionalAuth(c *gin.Context) bool {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	tokenData, err := h.Auth.ValidarToken(strings.TrimSpace(strings.TrimPrefix(header, prefix)))
	if err != nil {
		return false
	}
	c.Set("auth_user_id", tokenData.UserID)
	c.Set("auth_username", tokenData.Username)
	c.Set("auth_rol_codigo", tokenData.RolCodigo)
	if tokenData.LocalID != nil {
		c.Set("auth_local_id", *tokenData.LocalID)
	}
	if tokenData.NombreLocal != nil {
		c.Set("auth_nombre_local", *tokenData.NombreLocal)
	}
	return true
}

// consumoActor identifica al usuario autenticado para el kardex y la auditoria.
func consumoActor(c *gin.Context) repository.ConsumoActor {
	actor := repository.ConsumoActor{}
	if id, ok := authenticatedUserID(c); ok {
		actor.UsuarioID = &id
	}
	if name, ok := authenticatedUsername(c); ok {
		actor.UsuarioNombre = name
	}
	return actor
}

// trastiendaLocalScope devuelve el local al que se limita al usuario. admin_sys ve todo (nil); el resto queda
// restringido a su local (PLAN §7). Si un usuario no admin no tiene local asignado responde 403.
func trastiendaLocalScope(c *gin.Context) (*int, bool) {
	scope, ok := authenticatedLocalScopeFromToken(c)
	if !ok {
		utils.RespondError(c, http.StatusForbidden, "el usuario no tiene un local asignado")
		return nil, false
	}
	return scope.LocalID, true
}

// resolverLocalID combina el local pedido con el alcance del usuario: un usuario limitado a un local no puede
// consultar ni operar en otro.
func resolverLocalID(c *gin.Context, pedido *int) (*int, bool) {
	scope, ok := trastiendaLocalScope(c)
	if !ok {
		return nil, false
	}
	if scope == nil {
		return pedido, true
	}
	if pedido != nil && *pedido != *scope {
		utils.RespondError(c, http.StatusForbidden, "no tienes acceso a ese local")
		return nil, false
	}
	return scope, true
}

// decodePatch lee el body en `dst` y devuelve las claves presentes, para distinguir "no vino" de "vino null".
func decodePatch(c *gin.Context, dst interface{}) (map[string]bool, bool) {
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "no se pudo leer el body")
		return nil, false
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido: "+err.Error())
		return nil, false
	}
	keys := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &keys); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido: "+err.Error())
		return nil, false
	}
	present := make(map[string]bool, len(keys))
	for k := range keys {
		present[k] = true
	}
	return present, true
}

func queryInt(c *gin.Context, name string) (*int, error) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 {
		return nil, errors.New(name + " debe ser un entero positivo")
	}
	return &v, nil
}

func queryBool(c *gin.Context, name string) (*bool, error) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, errors.New(name + " debe ser true o false")
	}
	return &v, nil
}

// statusTrastiendaError mapea errores del service al codigo HTTP por substring del mensaje (AGENTS.md).
// Los errores de infraestructura ("error al ...", SQLSTATE) son 500; el resto son validaciones o conflictos.
func statusTrastiendaError(err error) int {
	msg := strings.ToLower(err.Error())
	switch {
	case errors.Is(err, repository.ErrItemNoEncontrado), errors.Is(err, repository.ErrCompraNoEncontrada),
		errors.Is(err, repository.ErrMovimientoFinNoEnc), errors.Is(err, repository.ErrGastoFijoNoEnc),
		errors.Is(err, repository.ErrCategoriaNoEnc),
		strings.Contains(msg, "no encontrad"):
		return http.StatusNotFound
	case errors.Is(err, services.ErrAlmacenamientoNoConfigurado):
		return http.StatusServiceUnavailable
	case errors.Is(err, repository.ErrStockInsuficiente), strings.Contains(msg, "stock insuficiente"),
		strings.Contains(msg, "ya existe"), strings.Contains(msg, "ya esta anulada"),
		strings.Contains(msg, "no puede editarse"), strings.Contains(msg, "pertenece a una compra"):
		return http.StatusConflict
	case strings.HasPrefix(msg, "error al"), strings.Contains(msg, "sqlstate"):
		return http.StatusInternalServerError
	default:
		return http.StatusBadRequest
	}
}

func respondTrastiendaError(c *gin.Context, err error) {
	utils.RespondError(c, statusTrastiendaError(err), err.Error())
}

// AdminSysOrJobSecret deja pasar al job diario con el header X-Job-Secret (si hay secreto configurado) o a un
// usuario admin_sys con token Bearer. Cualquier otro caso responde 401/403.
func (h *Container) AdminSysOrJobSecret(c *gin.Context) {
	if secret := strings.TrimSpace(h.JobSecret); secret != "" &&
		subtle.ConstantTimeCompare([]byte(c.GetHeader("X-Job-Secret")), []byte(secret)) == 1 {
		c.Next()
		return
	}
	h.AuthRequired(c)
	if c.IsAborted() {
		return
	}
	h.AdminSysRequired(c)
}
