package handlers

import (
	"net/http"

	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

// PostPaqueteImagenUploadURL godoc
// @Summary Emitir URL firmada para subir portada de paquete
// @Description Devuelve una URL firmada de Supabase Storage para que el cliente suba la imagen de portada directamente, sin pasar los bytes por el backend. Requiere token Bearer con rol admin_sys. Tras subir el archivo, confirmar con PUT /bd/paquetes/{id}/imagen.
// @Tags Paquetes BD
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del paquete" example(4)
// @Success 200 {object} utils.APIResponse{data=paqueteImagenUploadResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Paquete no encontrado"
// @Failure 503 {object} utils.APIResponse "Almacenamiento de imagenes no configurado"
// @Router /bd/paquetes/{id}/imagen/upload-url [post]
func (h *Container) PostPaqueteImagenUploadURL(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	subida, err := h.PaquetesPG.GenerarURLSubidaImagen(id)
	if err != nil {
		responderErrorPaquete(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, paqueteImagenUploadResponse{
		UploadURL: subida.UploadURL, Token: subida.Token, Path: subida.Path,
	})
}

// PutPaqueteImagen godoc
// @Summary Confirmar portada de paquete subida
// @Description Persiste la portada del paquete tras una subida exitosa y devuelve su URL publica. Requiere token Bearer con rol admin_sys.
// @Tags Paquetes BD
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del paquete" example(4)
// @Success 200 {object} utils.APIResponse{data=paqueteImagenResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Paquete no encontrado"
// @Failure 503 {object} utils.APIResponse "Almacenamiento de imagenes no configurado"
// @Router /bd/paquetes/{id}/imagen [put]
func (h *Container) PutPaqueteImagen(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	url, err := h.PaquetesPG.ConfirmarImagen(id)
	if err != nil {
		responderErrorPaquete(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, paqueteImagenResponse{ImagenURL: url})
}

// DeletePaqueteImagen godoc
// @Summary Eliminar portada de paquete
// @Description Borra el objeto de portada en Supabase Storage y limpia el path del paquete. Requiere token Bearer con rol admin_sys.
// @Tags Paquetes BD
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del paquete" example(4)
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Paquete no encontrado"
// @Failure 503 {object} utils.APIResponse "Almacenamiento de imagenes no configurado"
// @Router /bd/paquetes/{id}/imagen [delete]
func (h *Container) DeletePaqueteImagen(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.PaquetesPG.EliminarImagen(id); err != nil {
		responderErrorPaquete(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "imagen de paquete eliminada correctamente"})
}
