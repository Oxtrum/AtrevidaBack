package handlers

import (
	"errors"
	"net/http"
	"strings"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/pagination"
	repository "atrevida-agenda-api/repositories"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

type paqueteServicioBaseRequest struct {
	// ID opcional del servicio de catalogo usado como origen.
	ServicioID *int `json:"servicio_id,omitempty" example:"8"`
	// Nombre snapshot manual; si se omite y hay servicio_id, se copia del servicio origen.
	ServicioTexto *string `json:"servicio_texto,omitempty" example:"Masaje relajante personalizado"`
	// Precio unitario snapshot de cada sesion.
	Costo float64 `json:"costo" example:"250"`
	// Posicion unica de la linea dentro del catalogo base, iniciando en cero.
	Orden int `json:"orden" example:"0"`
}

type paqueteTierRequest struct {
	// ID del tier a actualizar; si se omite se crea uno nuevo.
	ID *int `json:"id,omitempty" example:"5"`
	// Cantidad de sesiones incluidas en este tier.
	Sesiones int `json:"sesiones" example:"4"`
	// Precio al contado del tier.
	PrecioContado float64 `json:"precio_contado" example:"700"`
	// Precio regular opcional, mayor al de contado, usado para mostrar descuento.
	PrecioRegular *float64 `json:"precio_regular,omitempty" example:"800"`
	// Nota comercial opcional del tier.
	Nota *string `json:"nota,omitempty" example:"Promocion vigente"`
}

type paqueteRequest struct {
	// Nombre comercial del paquete.
	Nombre string `json:"nombre" example:"Paquete Relax"`
	// Descripcion opcional mostrada en catalogo.
	Descripcion *string `json:"descripcion,omitempty" example:"Masajes y drenaje segun cantidad de sesiones"`
	// ID opcional de la categoria del catalogo.
	CategoriaID *int `json:"categoria_id,omitempty" example:"3"`
	// Moneda ISO de tres letras; BOB por defecto.
	Moneda string `json:"moneda,omitempty" example:"BOB"`
	// IDs de los locales activos donde se publica el paquete.
	LocalIDs []int `json:"local_ids" example:"1,2"`
	// Catalogo base de servicios que componen el paquete.
	ServiciosBase []paqueteServicioBaseRequest `json:"servicios_base"`
	// Tiers (precio por cantidad de sesiones) del paquete.
	Tiers []paqueteTierRequest `json:"tiers"`
}

// GetPaquetes godoc
// @Summary Listar paquetes de catalogo
// @Description Devuelve los paquetes que cumplan los filtros, con su catalogo base de servicios, locales y tiers (precio por cantidad de sesiones). Por defecto solo devuelve paquetes activos.
// @Tags Paquetes BD
// @Produce json
// @Param nombre query string false "Busqueda parcial por nombre" example(relax)
// @Param categoria query string false "Busqueda parcial por categoria" example(Corporal)
// @Param local query string false "Busqueda parcial por nombre de local" example(SAN MARTIN)
// @Param activo query bool false "Filtrar por activo; default true" example(true)
// @Param limit query int false "Tamano de pagina opcional (1-100); sin limit ni cursor conserva modo legacy" example(50)
// @Param cursor query string false "Cursor opaco devuelto en paginacion.next_cursor"
// @Param include_total query bool false "Incluye el total de paginas" example(false)
// @Success 200 {object} utils.APIResponse{data=paqueteListResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: activo invalido"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/paquetes [get]
func (h *Container) GetPaquetes(c *gin.Context) {
	activo, ok := optionalBoolQuery(c, "activo", true)
	if !ok {
		return
	}
	nombre := strings.TrimSpace(c.Query("nombre"))
	categoria := strings.TrimSpace(c.Query("categoria"))
	local := strings.TrimSpace(c.Query("local"))
	page, ok := parsePagination(c)
	if !ok {
		return
	}
	includeTotal, ok := parseIncludeTotal(c)
	if !ok {
		return
	}
	filters := paqueteFiltrosResponse{Nombre: nombre, Categoria: categoria, Local: local, Activo: activo}
	var after nameCursor
	if !decodePaginationCursor(c, page, "paquetes", filters, &after) || (page.Cursor != "" && after.ID < 1) {
		if !c.Writer.Written() {
			utils.RespondError(c, http.StatusBadRequest, "paginacion invalida: cursor incompleto")
		}
		return
	}
	paquetes, err := h.PaquetesPG.ListarContext(c.Request.Context(), repository.FiltroPaquetes{
		Nombre: nombre, Categoria: categoria, Local: local, Activo: activo,
		PageLimit: page.QueryLimit(), CursorSet: page.Cursor != "", CursorNombre: after.Name, CursorID: after.ID,
	})
	if err != nil {
		responderErrorPaquete(c, err)
		return
	}
	paquetes, metadata, err := pagination.Build(paquetes, page, "paquetes", filters, func(item models.PaqueteDetalle) any {
		return nameCursor{Name: item.Paquete.Nombre, ID: item.Paquete.ID}
	})
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if includeTotal {
		count, countErr := h.PaquetesPG.ContarContext(c.Request.Context(), repository.FiltroPaquetes{
			Nombre: nombre, Categoria: categoria, Local: local, Activo: activo,
		})
		if countErr != nil {
			responderErrorPaquete(c, countErr)
			return
		}
		pagination.AddTotal(metadata, count)
	}
	utils.Respond(c, http.StatusOK, paqueteListResponse{
		Total:      len(paquetes),
		Filtros:    filters,
		Paquetes:   paquetes,
		Paginacion: metadata,
	})
}

// GetPaqueteByID godoc
// @Summary Obtener paquete por ID
// @Description Devuelve un paquete activo con su catalogo base de servicios, locales y tiers.
// @Tags Paquetes BD
// @Produce json
// @Param id path int true "ID del paquete" example(4)
// @Success 200 {object} utils.APIResponse{data=paqueteItemResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 404 {object} utils.APIResponse "Paquete no encontrado o inactivo"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/paquetes/{id} [get]
func (h *Container) GetPaqueteByID(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	paquete, err := h.PaquetesPG.Obtener(id)
	if err != nil {
		responderErrorPaquete(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, paqueteItemResponse{Paquete: paquete})
}

// CreatePaquete godoc
// @Summary Crear paquete de catalogo
// @Description Crea un paquete con su catalogo base de servicios, locales y tiers (precio por cantidad de sesiones) en una unica transaccion. Requiere token Bearer con rol admin_sys. Cada tier se materializa internamente como un combo enlazado al paquete.
// @Tags Paquetes BD
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body paqueteRequest true "Datos completos del paquete"
// @Success 201 {object} utils.APIResponse{data=idResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: datos de paquete, locales, tiers o servicios invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Categoria, local o servicio no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/paquetes [post]
func (h *Container) CreatePaquete(c *gin.Context) {
	var req paqueteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}
	id, err := h.PaquetesPG.Crear(services.CrearPaqueteInput{
		Nombre: req.Nombre, Descripcion: req.Descripcion, CategoriaID: req.CategoriaID, Moneda: req.Moneda,
		LocalIDs: req.LocalIDs, ServiciosBase: toPaqueteServiciosInput(req.ServiciosBase), Tiers: toPaqueteTiersInput(req.Tiers),
	})
	if err != nil {
		responderErrorPaquete(c, err)
		return
	}
	utils.Respond(c, http.StatusCreated, idResponse{ID: id})
}

// PatchPaquete godoc
// @Summary Actualizar paquete de catalogo
// @Description Reemplaza los datos, catalogo base de servicios, locales y tiers de un paquete en una unica transaccion. Requiere token Bearer con rol admin_sys.
// @Tags Paquetes BD
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del paquete" example(4)
// @Param payload body paqueteRequest true "Datos completos del paquete"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id, locales, tiers o servicios invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Paquete, categoria o local no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/paquetes/{id} [patch]
func (h *Container) PatchPaquete(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var req paqueteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}
	err = h.PaquetesPG.Actualizar(services.ActualizarPaqueteInput{
		ID: id, Nombre: req.Nombre, Descripcion: req.Descripcion, CategoriaID: req.CategoriaID, Moneda: req.Moneda,
		LocalIDs: req.LocalIDs, ServiciosBase: toPaqueteServiciosInput(req.ServiciosBase), Tiers: toPaqueteTiersInput(req.Tiers),
	})
	if err != nil {
		responderErrorPaquete(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "paquete actualizado correctamente"})
}

// DeletePaquete godoc
// @Summary Desactivar paquete
// @Description Realiza el borrado logico de un paquete de catalogo: lo marca inactivo y deja de aparecer en el catalogo publico. Requiere token Bearer con rol admin_sys.
// @Tags Paquetes BD
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del paquete" example(4)
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Paquete no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/paquetes/{id} [delete]
func (h *Container) DeletePaquete(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.PaquetesPG.Eliminar(id); err != nil {
		responderErrorPaquete(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "paquete desactivado correctamente"})
}

func toPaqueteServiciosInput(requests []paqueteServicioBaseRequest) []repository.PaqueteServicioInput {
	result := make([]repository.PaqueteServicioInput, 0, len(requests))
	for _, request := range requests {
		result = append(result, repository.PaqueteServicioInput{
			ServicioID: request.ServicioID, ServicioTexto: request.ServicioTexto, Costo: request.Costo, Orden: request.Orden,
		})
	}
	return result
}

func toPaqueteTiersInput(requests []paqueteTierRequest) []models.PaqueteTierInput {
	result := make([]models.PaqueteTierInput, 0, len(requests))
	for _, request := range requests {
		result = append(result, models.PaqueteTierInput{
			ID: request.ID, Sesiones: request.Sesiones, PrecioContado: request.PrecioContado,
			PrecioRegular: request.PrecioRegular, Nota: request.Nota,
		})
	}
	return result
}

func responderErrorPaquete(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, services.ErrPaqueteInvalido):
		status = http.StatusBadRequest
	case errors.Is(err, services.ErrPaqueteNoEncontrado):
		status = http.StatusNotFound
	case errors.Is(err, services.ErrAlmacenamientoNoConfigurado):
		status = http.StatusServiceUnavailable
	}
	utils.RespondError(c, status, err.Error())
}
