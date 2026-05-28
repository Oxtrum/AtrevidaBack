package handlers

import (
	"errors"
	"net/http"
	"strings"

	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

type authRequest struct {
	// Nombre de usuario para registrar o iniciar sesion.
	Username string `json:"username" binding:"required" example:"admin"`
	// Password en texto plano enviada por el cliente; se guarda encriptada con bcrypt.
	Password string `json:"password" binding:"required" example:"Secreto123"`
}

type loginResponse struct {
	Token     string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType string `json:"token_type" example:"Bearer"`
	Username  string `json:"username" example:"admin"`
	ExpiresIn int    `json:"expires_in" example:"3600"`
}

// RegisterUsuario godoc
// @Summary Registrar usuario
// @Description Crea un usuario activo sin validacion de rol. Requiere token Bearer emitido por login. Body: username y password requeridos. La password se encripta con bcrypt antes de guardarse. Response: id (int ID del usuario creado).
// @Tags Auth
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body authRequest true "Credenciales del usuario"
// @Success 200 {object} utils.APIResponse{data=idResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: body invalido, username y password son obligatorios"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 409 {object} utils.APIResponse "Conflicto: usuario ya existe"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /auth/register [post]
func (h *Container) RegisterUsuario(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}

	id, err := h.Auth.RegistrarUsuario(services.RegistrarUsuarioInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrUsuarioYaExiste) {
			status = http.StatusConflict
		}
		if errors.Is(err, services.ErrCredencialesObligatorias) {
			status = http.StatusBadRequest
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, idResponse{ID: id})
}

// Login godoc
// @Summary Iniciar sesion
// @Description Valida username y password contra un usuario activo. La password recibida se compara con el hash bcrypt guardado en BD. Ante credenciales validas responde un token Bearer para acceder a endpoints protegidos.
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
		Token:     result.Token,
		TokenType: "Bearer",
		Username:  result.Username,
		ExpiresIn: result.ExpiresIn,
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

	if err := h.Auth.ValidarToken(token); err != nil {
		utils.RespondError(c, http.StatusUnauthorized, err.Error())
		c.Abort()
		return
	}

	c.Next()
}
