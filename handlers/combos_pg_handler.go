package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	repository "atrevida-agenda-api/repositories"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

type comboServicioCatalogoRequest struct {
	// ID opcional del servicio de catalogo usado como origen.
	ServicioID *int `json:"servicio_id,omitempty" example:"8"`
	// Nombre snapshot manual; si se omite y hay servicio_id, se copia del servicio origen.
	ServicioTexto string `json:"servicio_texto,omitempty" example:"Masaje relajante personalizado"`
	// Duracion snapshot del servicio en formato libre, por ejemplo HH:MM.
	Tiempo *string `json:"tiempo,omitempty" example:"01:00"`
	// Precio unitario snapshot de cada sesion.
	Costo *float64 `json:"costo,omitempty" example:"250"`
	// Cantidad de sesiones incluidas en esta linea.
	Sesiones int `json:"sesiones" example:"2"`
	// Posicion unica de la linea dentro del combo, iniciando en cero.
	Orden int `json:"orden" example:"0"`
}

type crearComboCatalogoRequest struct {
	// Nombre comercial de la promocion.
	Nombre string `json:"nombre" example:"Combo Relax"`
	// Descripcion opcional mostrada en catalogo.
	Descripcion *string `json:"descripcion,omitempty" example:"Masaje y drenaje para cuatro sesiones"`
	// ID opcional de la categoria del catalogo.
	CategoriaID *int `json:"categoria_id,omitempty" example:"3"`
	// Regla de precio: POR_ITEMS o PRECIO_PAQUETE.
	TipoPrecio string `json:"tipo_precio" example:"PRECIO_PAQUETE"`
	// Precio final requerido solo cuando tipo_precio es PRECIO_PAQUETE.
	PrecioPaquete *float64 `json:"precio_paquete,omitempty" example:"700"`
	// Moneda ISO de tres letras; BOB por defecto.
	Moneda string `json:"moneda,omitempty" example:"BOB"`
	// Sesiones/visitas del paquete (nivel paquete, no la suma de líneas).
	SesionesTotales int `json:"sesiones_totales" example:"4"`
	// Duracion sugerida por sesion en minutos; opcional.
	DuracionMin *int `json:"duracion_min,omitempty" example:"90"`
	// IDs de los locales activos donde se publica el combo.
	LocalIDs []int `json:"local_ids" example:"1,2"`
	// Lineas que componen la promocion.
	Servicios []comboServicioCatalogoRequest `json:"servicios"`
}

type actualizarComboCatalogoRequest struct {
	// Nuevo nombre comercial opcional.
	Nombre *string `json:"nombre,omitempty" example:"Combo Relax Premium"`
	// Nueva descripcion opcional; una cadena vacia la limpia.
	Descripcion *string `json:"descripcion,omitempty" example:"Masaje y drenaje personalizado"`
	// Nueva categoria opcional.
	CategoriaID *int `json:"categoria_id,omitempty" example:"3"`
	// Nueva regla de precio opcional: POR_ITEMS o PRECIO_PAQUETE.
	TipoPrecio *string `json:"tipo_precio,omitempty" example:"POR_ITEMS"`
	// Nuevo precio final de paquete cuando corresponda.
	PrecioPaquete *float64 `json:"precio_paquete,omitempty" example:"750"`
	// Nueva moneda ISO de tres letras.
	Moneda *string `json:"moneda,omitempty" example:"BOB"`
	// Nuevas sesiones/visitas del paquete.
	SesionesTotales *int `json:"sesiones_totales,omitempty" example:"4"`
	// Nueva duracion sugerida por sesion en minutos.
	DuracionMin *int `json:"duracion_min,omitempty" example:"90"`
}

type reemplazarLocalesComboRequest struct {
	// Lista completa de locales activos donde estara disponible el combo.
	LocalIDs []int `json:"local_ids" example:"1,2"`
}

type reemplazarServiciosComboRequest struct {
	// Lista completa que reemplaza las lineas activas actuales del combo.
	Servicios []comboServicioCatalogoRequest `json:"servicios"`
}

// GetCombosPG godoc
// @Summary Listar combos de catalogo
// @Description Devuelve todas las promociones activas que cumplan los filtros. Un combo no es una compra ni contiene pagos, reservas o progreso de cliente. Los filtros se aplican en PostgreSQL y la respuesta incluye los locales y lineas snapshot.
// @Tags Combos BD
// @Produce json
// @Param nombre query string false "Busqueda parcial por nombre" example(relax)
// @Param categoria query string false "Busqueda parcial por categoria" example(Corporal)
// @Param local query string false "Busqueda parcial por nombre de local" example(SAN MARTIN)
// @Param local_id query int false "ID exacto de local" example(1)
// @Success 200 {object} utils.APIResponse{data=comboCatalogoListResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: local_id invalido"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/combos [get]
func (h *Container) GetCombosPG(c *gin.Context) {
	localID, err := optionalPositiveInt(c, "local_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	combos, total, err := h.CombosPG.ListarCombos(services.FiltroCombos{
		Nombre: strings.TrimSpace(c.Query("nombre")), Categoria: strings.TrimSpace(c.Query("categoria")),
		Local: strings.TrimSpace(c.Query("local")), LocalID: localID,
	})
	if err != nil {
		responderErrorCombo(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, comboCatalogoListResponse{
		Total: total,
		Filtros: comboCatalogoFiltrosResponse{
			Nombre: strings.TrimSpace(c.Query("nombre")), Categoria: strings.TrimSpace(c.Query("categoria")),
			Local: strings.TrimSpace(c.Query("local")), LocalID: localID,
		},
		Combos: combos,
	})
}

// GetComboPGByID godoc
// @Summary Obtener combo de catalogo por ID
// @Description Devuelve una promocion activa con sus locales y lineas snapshot. No devuelve planes adquiridos ni saldos de clientes.
// @Tags Combos BD
// @Produce json
// @Param id path int true "ID del combo" example(12)
// @Success 200 {object} utils.APIResponse{data=comboCatalogoItemResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 404 {object} utils.APIResponse "Combo no encontrado o inactivo"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/combos/{id} [get]
func (h *Container) GetComboPGByID(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	combo, err := h.CombosPG.ObtenerCombo(id)
	if err != nil {
		responderErrorCombo(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, comboCatalogoItemResponse{Combo: combo})
}

// CreateComboPG godoc
// @Summary Crear combo de catalogo
// @Description Crea una promocion reutilizable con locales y lineas en una unica transaccion. Requiere token Bearer con rol admin_sys. tipo_precio POR_ITEMS calcula el total desde las lineas; PRECIO_PAQUETE exige precio_paquete. servicio_id es solo trazabilidad: el nombre, tiempo y costo se guardan como snapshot en el combo.
// @Tags Combos BD
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body crearComboCatalogoRequest true "Datos completos del combo"
// @Success 201 {object} utils.APIResponse{data=idResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: datos de combo, locales, precios o servicios invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Categoria, local o servicio no encontrado"
// @Failure 409 {object} utils.APIResponse "Conflicto al crear el combo"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/combos [post]
func (h *Container) CreateComboPG(c *gin.Context) {
	var req crearComboCatalogoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}
	id, err := h.CombosPG.CrearCombo(services.CrearComboCatalogoInput{
		Nombre: req.Nombre, Descripcion: req.Descripcion, CategoriaID: req.CategoriaID, TipoPrecio: req.TipoPrecio,
		PrecioPaquete: req.PrecioPaquete, Moneda: req.Moneda, SesionesTotales: req.SesionesTotales, DuracionMin: req.DuracionMin,
		LocalIDs: req.LocalIDs, Servicios: toComboServiciosInput(req.Servicios),
	})
	if err != nil {
		responderErrorCombo(c, err)
		return
	}
	utils.Respond(c, http.StatusCreated, idResponse{ID: id})
}

// PatchComboPG godoc
// @Summary Actualizar metadatos de combo
// @Description Actualiza metadatos financieros y comerciales del catalogo. No modifica planes adquiridos. Requiere token Bearer con rol admin_sys. Para reemplazar locales o lineas use las rutas dedicadas.
// @Tags Combos BD
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del combo" example(12)
// @Param payload body actualizarComboCatalogoRequest true "Campos del combo a modificar"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id o campos invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Combo o categoria no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/combos/{id} [patch]
func (h *Container) PatchComboPG(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var req actualizarComboCatalogoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}
	err = h.CombosPG.ActualizarCombo(services.ActualizarComboCatalogoInput{ID: id, Nombre: req.Nombre, Descripcion: req.Descripcion, CategoriaID: req.CategoriaID, TipoPrecio: req.TipoPrecio, PrecioPaquete: req.PrecioPaquete, Moneda: req.Moneda, SesionesTotales: req.SesionesTotales, DuracionMin: req.DuracionMin})
	if err != nil {
		responderErrorCombo(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "combo actualizado correctamente"})
}

// DeleteComboPG godoc
// @Summary Desactivar combo
// @Description Realiza el borrado lógico de una promoción de catálogo: la marca inactiva, deja de aparecer en el catálogo público y conserva sus snapshots históricos. Requiere token Bearer con rol admin_sys.
// @Tags Combos BD
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del combo" example(12)
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Combo no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/combos/{id} [delete]
func (h *Container) DeleteComboPG(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.CombosPG.DesactivarCombo(id); err != nil {
		responderErrorCombo(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "combo desactivado correctamente"})
}

// PutComboLocalesPG godoc
// @Summary Reemplazar locales de combo
// @Description Reemplaza la disponibilidad de una promocion por una lista completa de locales activos. Requiere token Bearer con rol admin_sys.
// @Tags Combos BD
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del combo" example(12)
// @Param payload body reemplazarLocalesComboRequest true "Locales finales del combo"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: locales invalidos o duplicados"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Combo o local no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/combos/{id}/locales [put]
func (h *Container) PutComboLocalesPG(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var req reemplazarLocalesComboRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}
	if err := h.CombosPG.ReemplazarLocales(id, req.LocalIDs); err != nil {
		responderErrorCombo(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "locales de combo actualizados correctamente"})
}

// PutComboServiciosPG godoc
// @Summary Reemplazar servicios de combo
// @Description Reemplaza en una transaccion las lineas activas de una promocion. Las lineas anteriores se desactivan para conservar historia de catalogo. Requiere token Bearer con rol admin_sys. Esta operacion recalcula sesiones y precio final sin afectar planes adquiridos.
// @Tags Combos BD
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del combo" example(12)
// @Param payload body reemplazarServiciosComboRequest true "Servicios finales del combo"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: servicios, ordenes, sesiones o costos invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Combo o servicio no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/combos/{id}/servicios [put]
func (h *Container) PutComboServiciosPG(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var req reemplazarServiciosComboRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}
	if err := h.CombosPG.ReemplazarServicios(id, toComboServiciosInput(req.Servicios)); err != nil {
		responderErrorCombo(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "servicios de combo actualizados correctamente"})
}

func toComboServiciosInput(requests []comboServicioCatalogoRequest) []repository.ComboServicioCatalogoInput {
	result := make([]repository.ComboServicioCatalogoInput, 0, len(requests))
	for _, request := range requests {
		result = append(result, repository.ComboServicioCatalogoInput{ServicioID: request.ServicioID, ServicioTexto: request.ServicioTexto, Tiempo: request.Tiempo, Costo: request.Costo, Sesiones: request.Sesiones, Orden: request.Orden})
	}
	return result
}

func requiredPositiveParam(c *gin.Context, name string) (int, error) {
	value, err := strconv.Atoi(c.Param(name))
	if err != nil || value < 1 {
		return 0, errors.New(name + " debe ser un entero positivo")
	}
	return value, nil
}

func optionalPositiveInt(c *gin.Context, name string) (*int, error) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return nil, errors.New(name + " debe ser un entero positivo")
	}
	return &value, nil
}

func responderErrorCombo(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, services.ErrComboInvalido), errors.Is(err, services.ErrComboServicioInvalido):
		status = http.StatusBadRequest
	case errors.Is(err, services.ErrComboNoEncontrado), errors.Is(err, services.ErrComboReferenciaNoEncontrada):
		status = http.StatusNotFound
	}
	utils.RespondError(c, status, err.Error())
}
