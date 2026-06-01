package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

type crearLocalHorarioRequest struct {
	// ID del local
	LocalID int `json:"local_id" example:"3"`
	// Día de la semana (1=lunes, 7=domingo)
	DiaSemana int `json:"dia_semana" example:"1"`
	// Hora de inicio (HH:MM)
	HoraDesde string `json:"hora_desde" example:"09:00"`
	// Hora de fin (HH:MM)
	HoraHasta string `json:"hora_hasta" example:"18:00"`
}

type actualizarLocalHorarioRequest struct {
	// Día de la semana 1-7 (opcional)
	DiaSemana *int `json:"dia_semana" example:"6"`
	// Hora de inicio HH:MM (opcional)
	HoraDesde *string `json:"hora_desde" example:"10:00"`
	// Hora de fin HH:MM (opcional)
	HoraHasta *string `json:"hora_hasta" example:"16:00"`
}

// GetHorariosByLocal godoc
// @Summary Listar horarios de un local
// @Description Devuelve los horarios activos de un local con filtro opcional. Query: local_id (requerido), dia_semana 1=lunes a 7=domingo (opcional). Response: total (int), filtros (objeto con local_id, dia_semana), horarios ([]LocalHorarioPG con: id, local_id, dia_semana, hora_desde HH:MM, hora_hasta HH:MM, activo).
// @Tags Horarios Locales
// @Produce json
// @Param local_id query int true "ID del local" example(3)
// @Param dia_semana query int false "Dia de la semana (1=lunes, 7=domingo)" example(1)
// @Success 200 {object} utils.APIResponse{data=horarioListResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: local_id invalido, dia_semana debe ser 1-7"
// @Failure 404 {object} utils.APIResponse "Local no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/locales/horarios [get]
func (h *Container) GetHorariosByLocal(c *gin.Context) {
	localID, err := strconv.Atoi(c.Query("local_id"))
	if err != nil || localID <= 0 {
		utils.RespondError(c, http.StatusBadRequest, "local_id invalido")
		return
	}

	var diaSemana *int
	if raw := strings.TrimSpace(c.Query("dia_semana")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 7 {
			utils.RespondError(c, http.StatusBadRequest, "dia_semana invalido")
			return
		}
		diaSemana = &value
	}

	horarios, err := h.LocalesHorariosPG.GetHorariosByLocal(services.FiltroLocalHorarios{
		LocalID:   localID,
		DiaSemana: diaSemana,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(strings.ToLower(err.Error()), "local no encontrado") {
			status = http.StatusNotFound
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, horarioListResponse{
		Total: len(horarios),
		Filtros: horarioFiltrosResponse{
			LocalID:   localID,
			DiaSemana: diaSemana,
		},
		Horarios: horarios,
	})
}

// GetHorarioByID godoc
// @Summary Obtener horario por ID
// @Description Devuelve un horario por su ID. Param: id (requerido, path). Response: horario (LocalHorarioPG con: id, local_id, dia_semana, hora_desde, hora_hasta, activo).
// @Tags Horarios Locales
// @Produce json
// @Param id path int true "ID del horario" example(15)
// @Success 200 {object} utils.APIResponse{data=horarioItemResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 404 {object} utils.APIResponse "Horario no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/locales/horarios/{id} [get]
func (h *Container) GetHorarioByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.RespondError(c, http.StatusBadRequest, "id invalido")
		return
	}

	horario, err := h.LocalesHorariosPG.GetHorarioByID(id)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(strings.ToLower(err.Error()), "no encontrado") {
			status = http.StatusNotFound
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, horarioItemResponse{Horario: horario})
}

// CreateHorarioByLocal godoc
// @Summary Crear horario para un local
// @Description Crea un tramo de horario para un local. Body: local_id (requerido), dia_semana 1-7 (requerido), hora_desde HH:MM (requerido), hora_hasta HH:MM (requerido, debe ser posterior a hora_desde). Response: id (int ID del horario creado).
// @Tags Horarios Locales
// @Accept json
// @Produce json
// @Param payload body crearLocalHorarioRequest true "Datos del horario (local_id, dia_semana, hora_desde, hora_hasta)"
// @Success 200 {object} utils.APIResponse{data=idResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: local_id invalido, dia_semana 1-7, hora_invalida, hora_hasta debe ser posterior"
// @Failure 404 {object} utils.APIResponse "Local no encontrado"
// @Failure 409 {object} utils.APIResponse "Conflicto: horario ya existe o se superpone con otro existente"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/locales/horarios [post]
func (h *Container) CreateHorarioByLocal(c *gin.Context) {
	var req crearLocalHorarioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}

	if req.LocalID <= 0 {
		utils.RespondError(c, http.StatusBadRequest, "local_id invalido")
		return
	}

	horaDesde, horaHasta, ok := validarHorarioInput(c, req.DiaSemana, req.HoraDesde, req.HoraHasta)
	if !ok {
		return
	}

	id, err := h.LocalesHorariosPG.CreateHorario(services.CrearLocalHorarioInput{
		LocalID:   req.LocalID,
		DiaSemana: req.DiaSemana,
		HoraDesde: horaDesde,
		HoraHasta: horaHasta,
	})
	if err != nil {
		status := http.StatusInternalServerError
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "local no encontrado") {
			status = http.StatusNotFound
		}
		if strings.Contains(lower, "ya existe") || strings.Contains(lower, "superpone") {
			status = http.StatusConflict
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, idResponse{ID: id})
}

// PatchHorario godoc
// @Summary Actualizar horario
// @Description Actualiza parcialmente un horario. Param: id (requerido, path). Body: dia_semana 1-7 (opcional), hora_desde HH:MM (opcional), hora_hasta HH:MM (opcional, debe ser posterior a hora_desde). Response: mensaje string.
// @Tags Horarios Locales
// @Accept json
// @Produce json
// @Param id path int true "ID del horario" example(15)
// @Param payload body actualizarLocalHorarioRequest true "Campos a actualizar (todos opcionales, al menos uno requerido)"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido, dia_semana 1-7, hora_invalida, hora_hasta debe ser posterior, sin cambios"
// @Failure 404 {object} utils.APIResponse "Horario no encontrado"
// @Failure 409 {object} utils.APIResponse "Conflicto: superposicion con otro horario existente"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/locales/horarios/{id} [patch]
func (h *Container) PatchHorario(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.RespondError(c, http.StatusBadRequest, "id invalido")
		return
	}

	var req actualizarLocalHorarioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}

	if req.DiaSemana == nil && req.HoraDesde == nil && req.HoraHasta == nil {
		utils.RespondError(c, http.StatusBadRequest, "debe enviar al menos un campo para actualizar")
		return
	}

	if req.DiaSemana != nil && (*req.DiaSemana < 1 || *req.DiaSemana > 7) {
		utils.RespondError(c, http.StatusBadRequest, "dia_semana invalido")
		return
	}

	var horaDesde *string
	if req.HoraDesde != nil {
		if strings.TrimSpace(*req.HoraDesde) == "" {
			utils.RespondError(c, http.StatusBadRequest, "hora_desde es obligatoria")
			return
		}
		value, err := normalizarHora(*req.HoraDesde)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, "hora_desde invalida")
			return
		}
		horaDesde = &value
	}

	var horaHasta *string
	if req.HoraHasta != nil {
		if strings.TrimSpace(*req.HoraHasta) == "" {
			utils.RespondError(c, http.StatusBadRequest, "hora_hasta es obligatoria")
			return
		}
		value, err := normalizarHora(*req.HoraHasta)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, "hora_hasta invalida")
			return
		}
		horaHasta = &value
	}

	if horaDesde != nil && horaHasta != nil {
		if !horaDesdeEsAnterior(*horaDesde, *horaHasta) {
			utils.RespondError(c, http.StatusBadRequest, "hora_hasta debe ser posterior a hora_desde")
			return
		}
	}

	err = h.LocalesHorariosPG.UpdateHorario(services.ActualizarLocalHorarioInput{
		ID:        id,
		DiaSemana: req.DiaSemana,
		HoraDesde: horaDesde,
		HoraHasta: horaHasta,
	})
	if err != nil {
		status := http.StatusInternalServerError
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "no encontrado") || strings.Contains(lower, "inactivo") {
			status = http.StatusNotFound
		}
		if strings.Contains(lower, "ya existe") || strings.Contains(lower, "superpone") {
			status = http.StatusConflict
		}
		if strings.Contains(lower, "debe especificarse") {
			status = http.StatusBadRequest
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "horario actualizado correctamente"})
}

// DeleteHorario godoc
// @Summary Eliminar horario
// @Description Realiza borrado logico de un horario (activo=false). Param: id (requerido, path). Response: mensaje string.
// @Tags Horarios Locales
// @Produce json
// @Param id path int true "ID del horario" example(15)
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 404 {object} utils.APIResponse "Horario no encontrado o ya inactivo"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/locales/horarios/{id} [delete]
func (h *Container) DeleteHorario(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.RespondError(c, http.StatusBadRequest, "id invalido")
		return
	}

	err = h.LocalesHorariosPG.DeleteHorario(id)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(strings.ToLower(err.Error()), "no encontrado") ||
			strings.Contains(strings.ToLower(err.Error()), "inactivo") {
			status = http.StatusNotFound
		}
		utils.RespondError(c, status, err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "horario eliminado correctamente"})
}

func validarHorarioInput(c *gin.Context, diaSemana int, horaDesdeRaw, horaHastaRaw string) (string, string, bool) {
	if diaSemana < 1 || diaSemana > 7 {
		utils.RespondError(c, http.StatusBadRequest, "dia_semana invalido")
		return "", "", false
	}

	horaDesde, err := normalizarHora(horaDesdeRaw)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "hora_desde invalida")
		return "", "", false
	}

	horaHasta, err := normalizarHora(horaHastaRaw)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "hora_hasta invalida")
		return "", "", false
	}

	if !horaDesdeEsAnterior(horaDesde, horaHasta) {
		utils.RespondError(c, http.StatusBadRequest, "hora_hasta debe ser posterior a hora_desde")
		return "", "", false
	}

	return horaDesde, horaHasta, true
}

func normalizarHora(raw string) (string, error) {
	hora := strings.TrimSpace(raw)
	if len(hora) == 4 {
		hora = "0" + hora
	}

	t, err := time.Parse("15:04", hora)
	if err != nil {
		return "", err
	}

	return t.Format("15:04"), nil
}

func horaDesdeEsAnterior(horaDesde, horaHasta string) bool {
	desde, errDesde := time.Parse("15:04", horaDesde)
	hasta, errHasta := time.Parse("15:04", horaHasta)
	if errDesde != nil || errHasta != nil {
		return false
	}
	return desde.Before(hasta)
}
