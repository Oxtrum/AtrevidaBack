package handlers

import (
	"net/http"

	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

// PostComboImagenUploadURL godoc
// @Summary Emitir URL firmada para subir portada de combo
// @Description Devuelve una URL firmada de Supabase Storage para que el cliente suba la imagen de portada directamente, sin pasar los bytes por el backend. Requiere token Bearer con rol admin_sys. Tras subir el archivo, confirmar con PUT /bd/combos/{id}/imagen.
// @Tags Combos BD
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del combo" example(12)
// @Success 200 {object} utils.APIResponse{data=comboImagenUploadResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Combo no encontrado"
// @Failure 503 {object} utils.APIResponse "Almacenamiento de imagenes no configurado"
// @Router /bd/combos/{id}/imagen/upload-url [post]
func (h *Container) PostComboImagenUploadURL(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	subida, err := h.CombosPG.GenerarURLSubidaImagen(id)
	if err != nil {
		responderErrorCombo(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, comboImagenUploadResponse{
		UploadURL: subida.UploadURL, Token: subida.Token, Path: subida.Path,
	})
}

// PutComboImagen godoc
// @Summary Confirmar portada de combo subida
// @Description Persiste la portada del combo tras una subida exitosa y devuelve su URL publica. Requiere token Bearer con rol admin_sys.
// @Tags Combos BD
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del combo" example(12)
// @Success 200 {object} utils.APIResponse{data=comboImagenResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Combo no encontrado"
// @Failure 503 {object} utils.APIResponse "Almacenamiento de imagenes no configurado"
// @Router /bd/combos/{id}/imagen [put]
func (h *Container) PutComboImagen(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	url, err := h.CombosPG.ConfirmarImagen(id)
	if err != nil {
		responderErrorCombo(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, comboImagenResponse{ImagenURL: url})
}

// DeleteComboImagen godoc
// @Summary Eliminar portada de combo
// @Description Borra el objeto de portada en Supabase Storage y limpia el path del combo. Requiere token Bearer con rol admin_sys.
// @Tags Combos BD
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del combo" example(12)
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Combo no encontrado"
// @Failure 503 {object} utils.APIResponse "Almacenamiento de imagenes no configurado"
// @Router /bd/combos/{id}/imagen [delete]
func (h *Container) DeleteComboImagen(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.CombosPG.EliminarImagen(id); err != nil {
		responderErrorCombo(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "imagen de combo eliminada correctamente"})
}
