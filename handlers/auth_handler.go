package handlers

import (
	"errors"
	"net/http"
	"strings"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type authRequest struct {
	// Nombre de usuario para iniciar sesion.
	Username string `json:"username" binding:"required" example:"admin"`
	// Password en texto plano enviada por el cliente.
	Password string `json:"password" binding:"required" example:"Secreto123"`
}

type registrarUsuarioRequest struct {
	// Nombre de usuario a registrar.
	Username string `json:"username" binding:"required" example:"operador"`
	// Password en texto plano enviada por el cliente; se guarda encriptada con bcrypt.
	Password string `json:"password" binding:"required" example:"Secreto123"`
	// Codigo del rol a asignar al usuario.
	RolCodigo string `json:"rol_codigo" binding:"required" example:"gerencia"`
	// ID del local asignado al usuario. Requerido para roles no admin; se ignora para admin_sys.
	LocalID *int `json:"local_id,omitempty" example:"1"`
}

type loginResponse struct {
	// Token Bearer firmado para usar en endpoints protegidos.
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	// Tipo de token devuelto.
	TokenType string `json:"token_type" example:"Bearer"`
	// Nombre de usuario autenticado.
	Username string `json:"username" example:"admin"`
	// Codigo del rol del usuario autenticado.
	RolCodigo string `json:"rol_codigo" example:"admin_sys"`
	// ID del local asignado al usuario; se omite para admin_sys.
	LocalID *int `json:"local_id,omitempty" example:"1"`
	// Nombre del local asignado al usuario; se omite para admin_sys.
	NombreLocal *string `json:"nombre_local,omitempty" example:"SAN MARTIN"`
	// Duracion del token en segundos.
	ExpiresIn int `json:"expires_in" example:"3600"`
}

type usuariosListResponse struct {
	// Cantidad total de usuarios devueltos.
	Total int `json:"total" example:"2"`
	// Usuarios registrados con rol y local asignado cuando corresponda.
	Usuarios []models.UsuarioResumenPG `json:"usuarios"`
}

type cambiarPasswordRequest struct {
	// Nueva password del usuario autenticado.
	Password string `json:"password" binding:"required" example:"NuevoSecreto123"`
}

type actualizarUsuarioActivoRequest struct {
	// Nombre de usuario a activar o desactivar.
	Username string `json:"username" binding:"required" example:"operador"`
	// Estado activo del usuario a modificar.
	Activo *bool `json:"activo" binding:"required" example:"true"`
}

type actualizarUsuarioLocalRequest struct {
	// Nombre de usuario a modificar.
	Username string `json:"username" binding:"required" example:"operador"`
	// ID del local activo a asignar al usuario.
	LocalID *int `json:"local_id" binding:"required" example:"1"`
}

// GetUsuarios godoc
// @Summary Listar usuarios
// @Description Devuelve todos los usuarios registrados sin filtros. Requiere token Bearer con rol admin_sys. Response: total (int), usuarios ([]UsuarioResumenPG con username, activo, fecha_registro, rol_codigo, rol_nombre, local_id y nombre_local).
// @Tags Auth
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Success 200 {object} utils.APIResponse{data=usuariosListResponse}
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /auth/usuarios [get]
func (h *Container) GetUsuarios(c *gin.Context) {
	usuarios, err := h.Auth.GetUsuarios()
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, usuariosListResponse{
		Total:    len(usuarios),
		Usuarios: usuarios,
	})
}

// RegisterUsuario godoc
// @Summary Registrar usuario
// @Description Crea un usuario activo y le asigna un rol por codigo. Requiere token Bearer con rol admin_sys. Body: username, password y rol_codigo requeridos; local_id es requerido solo para roles no admin. nombre_local no se recibe del cliente: se obtiene desde locales por local_id y se guarda desnormalizado. Si el rol es admin_sys, local_id y nombre_local quedan vacios. Response: id (int ID del usuario creado).
// @Tags Auth
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body registrarUsuarioRequest true "Datos del usuario"
// @Success 200 {object} utils.APIResponse{data=idResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: JSON invalido, username/password obligatorios, rol_codigo obligatorio, local_id obligatorio para usuarios no admin o local_id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Rol o local no encontrado"
// @Failure 409 {object} utils.APIResponse "Conflicto: usuario ya existe"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /auth/register [post]
func (h *Container) RegisterUsuario(c *gin.Context) {
	var req registrarUsuarioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, registrarUsuarioValidationMessage(err))
		return
	}

	id, err := h.Auth.RegistrarUsuario(services.RegistrarUsuarioInput{
		Username:  req.Username,
		Password:  req.Password,
		RolCodigo: req.RolCodigo,
		LocalID:   req.LocalID,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrUsuarioYaExiste) {
			status = http.StatusConflict
		}
		if errors.Is(err, services.ErrCredencialesObligatorias) {
			status = http.StatusBadRequest
		}
		if errors.Is(err, services.ErrRolObligatorio) {
			status = http.StatusBadRequest
		}
		if errors.Is(err, services.ErrLocalObligatorio) {
			status = http.StatusBadRequest
		}
		if errors.Is(err, services.ErrLocalInvalido) {
			status = http.StatusBadRequest
		}
		if errors.Is(err, services.ErrRolNoEncontrado) {
			status = http.StatusNotFound
		}
		if errors.Is(err, services.ErrLocalNoEncontrado) {
			status = http.StatusNotFound
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, idResponse{ID: id})
}

func registrarUsuarioValidationMessage(err error) string {
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return "body JSON invalido"
	}

	for _, validationError := range validationErrors {
		if validationError.Field() == "Username" || validationError.Field() == "Password" {
			return services.ErrCredencialesObligatorias.Error()
		}
	}

	return services.ErrRolObligatorio.Error()
}

// CambiarPassword godoc
// @Summary Cambiar password propia
// @Description Cambia la password del usuario autenticado. Requiere token Bearer y toma el usuario exclusivamente del token. Body: password requerida.
// @Tags Auth
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body cambiarPasswordRequest true "Nueva password"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: body invalido, password obligatoria"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 404 {object} utils.APIResponse "Usuario no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /auth/change-password [patch]
func (h *Container) CambiarPassword(c *gin.Context) {
	var req cambiarPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}

	tokenUserID, ok := authenticatedUserID(c)
	if !ok {
		utils.RespondError(c, http.StatusUnauthorized, "token invalido")
		return
	}

	err := h.Auth.CambiarPassword(services.CambiarPasswordInput{
		TokenUserID: tokenUserID,
		Password:    req.Password,
	})
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, services.ErrCredencialesObligatorias):
			status = http.StatusBadRequest
		case errors.Is(err, services.ErrUsuarioNoEncontrado):
			status = http.StatusNotFound
		case err.Error() == "token invalido":
			status = http.StatusUnauthorized
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "password actualizada correctamente"})
}

// ActualizarUsuarioActivo godoc
// @Summary Activar o desactivar usuario
// @Description Actualiza el estado activo de otro usuario. Requiere token Bearer con rol admin_sys. No permite modificar el estado del usuario autenticado ni desactivar al unico usuario admin_sys activo. Body: username y activo requeridos; activo=false desactiva, activo=true reactiva.
// @Tags Auth
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body actualizarUsuarioActivoRequest true "Usuario y estado activo"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: body invalido, username requerido, activo requerido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Usuario no encontrado"
// @Failure 409 {object} utils.APIResponse "Conflicto: ya existe un usuario activo con ese nombre o se intenta desactivar al unico admin_sys activo"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /auth/deactivate [patch]
func (h *Container) ActualizarUsuarioActivo(c *gin.Context) {
	var req actualizarUsuarioActivoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}
	if strings.TrimSpace(req.Username) == "" {
		utils.RespondError(c, http.StatusBadRequest, "username es requerido")
		return
	}
	if req.Activo == nil {
		utils.RespondError(c, http.StatusBadRequest, "activo es requerido")
		return
	}

	tokenUsername, ok := authenticatedUsername(c)
	if !ok {
		utils.RespondError(c, http.StatusUnauthorized, "token invalido")
		return
	}

	err := h.Auth.ActualizarUsuarioActivo(services.ActualizarUsuarioActivoInput{
		Username:      req.Username,
		TokenUsername: tokenUsername,
		Activo:        *req.Activo,
	})
	if err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		switch {
		case err.Error() == "username es requerido":
			status = http.StatusBadRequest
		case errors.Is(err, services.ErrNoModificarPropioEstado):
			status = http.StatusForbidden
			message = services.ErrNoAutorizado.Error()
		case errors.Is(err, services.ErrUsuarioNoEncontrado):
			status = http.StatusNotFound
		case errors.Is(err, services.ErrUsuarioYaExiste):
			status = http.StatusConflict
		case errors.Is(err, services.ErrUltimoAdminSysActivo):
			status = http.StatusConflict
		}
		utils.RespondError(c, status, message)
		return
	}

	mensaje := "usuario desactivado correctamente"
	if *req.Activo {
		mensaje = "usuario restaurado correctamente"
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: mensaje})
}

// ActualizarUsuarioLocal godoc
// @Summary Cambiar local de usuario
// @Description Actualiza el local asignado a un usuario. Requiere token Bearer con rol admin_sys. El body recibe username y local_id; el local debe existir y estar activo. Un admin puede cambiar su propio local o el local de un usuario no admin, pero no puede cambiar el local de otro admin. Al actualizar se guardan usuarios.local_id y usuarios.nombre_local usando el nombre actual de la tabla locales.
// @Tags Auth
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body actualizarUsuarioLocalRequest true "Usuario y local a asignar"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: body invalido, username requerido o local_id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado o solo un administrador puede asignarse un local"
// @Failure 404 {object} utils.APIResponse "Usuario o local no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /auth/usuarios/local [patch]
func (h *Container) ActualizarUsuarioLocal(c *gin.Context) {
	var req actualizarUsuarioLocalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		utils.RespondError(c, http.StatusBadRequest, "username es requerido")
		return
	}
	if req.LocalID == nil {
		utils.RespondError(c, http.StatusBadRequest, "local_id es requerido")
		return
	}
	tokenUsername, ok := authenticatedUsername(c)
	if !ok {
		utils.RespondError(c, http.StatusUnauthorized, "token invalido")
		return
	}

	err := h.Auth.ActualizarUsuarioLocal(services.ActualizarUsuarioLocalInput{
		Username:      username,
		TokenUsername: tokenUsername,
		LocalID:       *req.LocalID,
	})
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case err.Error() == "username es requerido":
			status = http.StatusBadRequest
		case err.Error() == "token invalido":
			status = http.StatusUnauthorized
		case errors.Is(err, services.ErrLocalInvalido):
			status = http.StatusBadRequest
		case errors.Is(err, services.ErrSoloAdminAsignarseLocal):
			status = http.StatusForbidden
		case errors.Is(err, services.ErrUsuarioNoEncontrado):
			status = http.StatusNotFound
		case errors.Is(err, services.ErrLocalNoEncontrado):
			status = http.StatusNotFound
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "local de usuario actualizado correctamente"})
}

// Login godoc
// @Summary Iniciar sesion
// @Description Valida username y password contra un usuario activo. La password recibida se compara con el hash bcrypt guardado en BD. Ante credenciales validas responde un token Bearer con el codigo de rol, local_id y nombre_local cuando el usuario tiene local asignado. Los usuarios admin_sys no incluyen local para poder consultar todos los locales.
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body authRequest true "Credenciales de acceso"
// @Success 200 {object} utils.APIResponse{data=loginResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: body invalido, username y password son obligatorios"
// @Failure 401 {object} utils.APIResponse "Contrasena incorrecta"
// @Failure 404 {object} utils.APIResponse "Usuario no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /auth/login [post]
func (h *Container) Login(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}

	result, err := h.Auth.Login(services.LoginInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, services.ErrUsuarioNoEncontrado):
			status = http.StatusNotFound
		case errors.Is(err, services.ErrPasswordIncorrecta):
			status = http.StatusUnauthorized
		case errors.Is(err, services.ErrCredencialesObligatorias):
			status = http.StatusBadRequest
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, loginResponse{
		Token:       result.Token,
		TokenType:   "Bearer",
		Username:    result.Username,
		RolCodigo:   result.RolCodigo,
		LocalID:     result.LocalID,
		NombreLocal: result.NombreLocal,
		ExpiresIn:   result.ExpiresIn,
	})
}

func (h *Container) AuthRequired(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if strings.TrimSpace(authHeader) == "" {
		utils.RespondError(c, http.StatusUnauthorized, "token requerido")
		c.Abort()
		return
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		utils.RespondError(c, http.StatusUnauthorized, "token invalido")
		c.Abort()
		return
	}

	token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
	if token == "" {
		utils.RespondError(c, http.StatusUnauthorized, "token requerido")
		c.Abort()
		return
	}

	tokenData, err := h.Auth.ValidarToken(token)
	if err != nil {
		utils.RespondError(c, http.StatusUnauthorized, err.Error())
		c.Abort()
		return
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
	c.Next()
}

func (h *Container) AdminSysRequired(c *gin.Context) {
	rolCodigo, ok := authenticatedRolCodigo(c)
	if !ok || !strings.EqualFold(rolCodigo, "admin_sys") {
		utils.RespondError(c, http.StatusForbidden, services.ErrNoAutorizado.Error())
		c.Abort()
		return
	}

	c.Next()
}

func authenticatedUserID(c *gin.Context) (int, bool) {
	value, exists := c.Get("auth_user_id")
	if !exists {
		return 0, false
	}

	userID, ok := value.(int)
	if !ok || userID <= 0 {
		return 0, false
	}

	return userID, true
}

func authenticatedUsername(c *gin.Context) (string, bool) {
	value, exists := c.Get("auth_username")
	if !exists {
		return "", false
	}

	username, ok := value.(string)
	if !ok || strings.TrimSpace(username) == "" {
		return "", false
	}

	return username, true
}

func authenticatedRolCodigo(c *gin.Context) (string, bool) {
	value, exists := c.Get("auth_rol_codigo")
	if !exists {
		return "", false
	}

	rolCodigo, ok := value.(string)
	if !ok || strings.TrimSpace(rolCodigo) == "" {
		return "", false
	}

	return rolCodigo, true
}

type authenticatedLocalScope struct {
	LocalID     *int
	NombreLocal string
}

func authenticatedLocalScopeFromToken(c *gin.Context) (authenticatedLocalScope, bool) {
	rolCodigo, ok := authenticatedRolCodigo(c)
	if !ok {
		return authenticatedLocalScope{}, false
	}
	if strings.EqualFold(rolCodigo, "admin_sys") {
		return authenticatedLocalScope{}, true
	}

	localID, localIDOK := authenticatedLocalID(c)
	nombreLocal, nombreLocalOK := authenticatedNombreLocal(c)
	if !localIDOK || !nombreLocalOK {
		return authenticatedLocalScope{}, false
	}

	return authenticatedLocalScope{
		LocalID:     &localID,
		NombreLocal: nombreLocal,
	}, true
}

func authenticatedLocalID(c *gin.Context) (int, bool) {
	value, exists := c.Get("auth_local_id")
	if !exists {
		return 0, false
	}

	localID, ok := value.(int)
	if !ok || localID <= 0 {
		return 0, false
	}

	return localID, true
}

func authenticatedNombreLocal(c *gin.Context) (string, bool) {
	value, exists := c.Get("auth_nombre_local")
	if !exists {
		return "", false
	}

	nombreLocal, ok := value.(string)
	if !ok || strings.TrimSpace(nombreLocal) == "" {
		return "", false
	}

	return nombreLocal, true
}
