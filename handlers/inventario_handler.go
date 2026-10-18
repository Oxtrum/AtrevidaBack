package handlers

import (
	"net/http"
	"strings"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

// ─── Requests / responses ─────────────────────────────────────────────────────

type inventarioItemsResponse struct {
	// Items de inventario con el stock del local consultado (o la suma de todos)
	Items []models.InventarioItem `json:"items"`
}

type inventarioItemResponse struct {
	// Item de inventario
	Item models.InventarioItem `json:"item"`
}

type crearInventarioItemRequest struct {
	// Nombre del item
	Nombre string `json:"nombre" binding:"required" example:"Gel conductor"`
	// Descripcion opcional
	Descripcion *string `json:"descripcion" example:"Gel para radiofrecuencia"`
	// ID de la categoria de inventario
	CategoriaID *int `json:"categoria_id" example:"1"`
	// Unidad de medida: UNIDAD, ML, GR o CAJA (default UNIDAD)
	UnidadMedida string `json:"unidad_medida" example:"ML"`
	// Ultimo costo de compra (se actualiza solo con cada compra)
	CostoReferencia *float64 `json:"costo_referencia" example:"0.04"`
	// Stock minimo: por debajo se marca como bajo stock
	StockMinimo float64 `json:"stock_minimo" example:"500"`
}

type actualizarInventarioItemRequest struct {
	// Nombre del item
	Nombre *string `json:"nombre" example:"Gel conductor"`
	// Descripcion (null limpia)
	Descripcion *string `json:"descripcion" example:"Gel para radiofrecuencia"`
	// Categoria (null limpia)
	CategoriaID *int `json:"categoria_id" example:"1"`
	// Unidad de medida: UNIDAD, ML, GR o CAJA
	UnidadMedida *string `json:"unidad_medida" example:"ML"`
	// Costo de referencia (null limpia)
	CostoReferencia *float64 `json:"costo_referencia" example:"0.04"`
	// Stock minimo
	StockMinimo *float64 `json:"stock_minimo" example:"500"`
	// Activo o desactivado (RN-07)
	Activo *bool `json:"activo" example:"true"`
}

type inventarioImagenUploadResponse struct {
	// URL firmada a la que el cliente hace PUT con el archivo
	UploadURL string `json:"upload_url" example:"https://bucket.example.com/inventario/1/imagen?X-Signature=abc"`
	// Token de la subida firmada (si el storage lo usa)
	Token string `json:"token" example:"abc"`
	// Path del objeto en el bucket
	Path string `json:"path" example:"inventario/1/imagen"`
	// Clave del objeto (igual a path); es lo que el cliente confirma
	ImagenKey string `json:"imagen_key" example:"inventario/1/imagen"`
}

type inventarioImagenResponse struct {
	// URL publica de la imagen
	ImagenURL string `json:"imagen_url" example:"https://bucket.example.com/inventario/1/imagen"`
}

type categoriasInventarioResponse struct {
	// Categorias de inventario
	Categorias []models.CategoriaInventario `json:"categorias"`
}

type crearCategoriaInventarioRequest struct {
	// Nombre de la categoria
	Nombre string `json:"nombre" binding:"required" example:"Cremas y geles"`
}

type actualizarCategoriaInventarioRequest struct {
	// Nuevo nombre
	Nombre *string `json:"nombre" example:"Cremas"`
	// Activa o inactiva
	Activo *bool `json:"activo" example:"true"`
}

type movimientosInventarioResponse struct {
	// Kardex filtrado, mas reciente primero
	Movimientos []models.MovimientoInventario `json:"movimientos"`
}

type ajusteInventarioRequest struct {
	// ID del item
	ItemID int `json:"item_id" binding:"required" example:"1"`
	// ID del local donde se ajusta
	LocalID int `json:"local_id" binding:"required" example:"1"`
	// AJUSTE_POSITIVO o AJUSTE_NEGATIVO
	Tipo string `json:"tipo" binding:"required" example:"AJUSTE_NEGATIVO"`
	// Cantidad a ajustar (siempre positiva)
	Cantidad float64 `json:"cantidad" binding:"required" example:"10"`
	// Motivo obligatorio (merma, vencimiento, conteo fisico)
	Motivo string `json:"motivo" example:"Producto vencido"`
}

type ajusteInventarioResponse struct {
	// Stock resultante tras el ajuste
	StockResultante float64 `json:"stock_resultante" example:"340"`
	// Mensaje de confirmacion
	Mensaje string `json:"mensaje" example:"ajuste registrado correctamente"`
}

type insumosServicioResponse struct {
	// Insumos que consume el servicio por sesion
	Insumos []models.ServicioInsumo `json:"insumos"`
}

type reemplazarInsumosRequest struct {
	// Lista completa de insumos del servicio (reemplaza la anterior; vacia los quita todos)
	Insumos []insumoRequest `json:"insumos"`
}

type insumoRequest struct {
	// ID del item de inventario
	ItemID int `json:"item_id" binding:"required" example:"1"`
	// Cantidad que consume cada sesion
	CantidadPorSesion float64 `json:"cantidad_por_sesion" binding:"required" example:"30"`
}

// ─── Items ────────────────────────────────────────────────────────────────────

// GetInventarioItems godoc
// @Summary Listar items de inventario
// @Description Lista items con el stock del local indicado (o la suma de todos). Filtros: q (nombre o codigo), categoria_id, local_id, bajo_stock=true (stock <= stock_minimo), activo. Los usuarios no admin_sys quedan limitados a su local. Requiere token Bearer.
// @Tags Inventario
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param q query string false "Busqueda por nombre o codigo"
// @Param categoria_id query int false "Filtrar por categoria"
// @Param local_id query int false "Stock de este local (admin_sys)"
// @Param bajo_stock query bool false "Solo items bajo su stock minimo"
// @Param activo query bool false "Filtrar por activos/inactivos"
// @Param limit query int false "Maximo de resultados (default 200, tope 500)"
// @Success 200 {object} utils.APIResponse{data=inventarioItemsResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: parametro de consulta invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Local no permitido para el usuario"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/inventario/items [get]
func (h *Container) GetInventarioItems(c *gin.Context) {
	categoriaID, err := queryInt(c, "categoria_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	localPedido, err := queryInt(c, "local_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	bajo, err := queryBool(c, "bajo_stock")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	activo, err := queryBool(c, "activo")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	limit, err := queryInt(c, "limit")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	localID, ok := resolverLocalID(c, localPedido)
	if !ok {
		return
	}
	f := repository.FiltroItemsInventario{
		Q: c.Query("q"), CategoriaID: categoriaID, LocalID: localID, BajoStock: bajo != nil && *bajo, Activo: activo,
	}
	if limit != nil {
		f.Limit = *limit
	}
	items, err := h.Inventario.ListarItems(f)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, inventarioItemsResponse{Items: items})
}

// GetInventarioItemByID godoc
// @Summary Detalle de item de inventario
// @Description Devuelve el item con su stock por local. Requiere token Bearer.
// @Tags Inventario
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del item" example(1)
// @Success 200 {object} utils.APIResponse{data=inventarioItemResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 404 {object} utils.APIResponse "Item no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/inventario/items/{id} [get]
func (h *Container) GetInventarioItemByID(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.Inventario.ObtenerItem(id)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	scope, ok := trastiendaLocalScope(c)
	if !ok {
		return
	}
	if scope != nil { // usuarios de un local solo ven su stock
		filtrado := item.StockPorLocal[:0:0]
		item.StockActual = 0
		for _, s := range item.StockPorLocal {
			if s.LocalID == *scope {
				filtrado = append(filtrado, s)
				item.StockActual = s.Cantidad
			}
		}
		item.StockPorLocal = filtrado
		item.BajoStock = item.StockActual <= item.StockMinimo
	}
	utils.Respond(c, http.StatusOK, inventarioItemResponse{Item: *item})
}

// CreateInventarioItem godoc
// @Summary Crear item de inventario
// @Description Crea un item del catalogo (stock inicial 0: se carga con una compra o un ajuste). El codigo INV-xxxxxx se genera solo. Requiere token Bearer con rol admin_sys.
// @Tags Inventario
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body crearInventarioItemRequest true "Datos del item"
// @Success 201 {object} utils.APIResponse{data=inventarioItemResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: nombre requerido, unidad_medida invalida, costo o stock minimo negativos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/inventario/items [post]
func (h *Container) CreateInventarioItem(c *gin.Context) {
	var req crearInventarioItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.Inventario.CrearItem(pgsqlrepo.ItemInput{
		Nombre: req.Nombre, Descripcion: req.Descripcion, CategoriaID: req.CategoriaID, UnidadMedida: req.UnidadMedida,
		CostoReferencia: req.CostoReferencia, StockMinimo: req.StockMinimo,
	})
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusCreated, inventarioItemResponse{Item: *item})
}

// PatchInventarioItem godoc
// @Summary Editar item de inventario
// @Description Edita datos del item. No edita stock: el stock solo cambia con movimientos (compras, ajustes, servicios). Los campos descripcion, categoria_id y costo_referencia aceptan null para limpiar. Requiere token Bearer con rol admin_sys.
// @Tags Inventario
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del item" example(1)
// @Param payload body actualizarInventarioItemRequest true "Campos a actualizar"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido, sin campos, unidad invalida, valores negativos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Item no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/inventario/items/{id} [patch]
func (h *Container) PatchInventarioItem(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var req actualizarInventarioItemRequest
	present, ok := decodePatch(c, &req)
	if !ok {
		return
	}
	err = h.Inventario.ActualizarItem(id, pgsqlrepo.ItemPatch{
		Nombre: req.Nombre, Descripcion: req.Descripcion, DescripcionSet: present["descripcion"],
		CategoriaID: req.CategoriaID, CategoriaSet: present["categoria_id"], UnidadMedida: req.UnidadMedida,
		CostoReferencia: req.CostoReferencia, CostoReferenciaSet: present["costo_referencia"],
		StockMinimo: req.StockMinimo, Activo: req.Activo,
	})
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "item actualizado correctamente"})
}

// DeleteInventarioItem godoc
// @Summary Desactivar item de inventario
// @Description Desactiva el item (activo=false). Nunca se borra fisicamente: conserva su historial de movimientos. Requiere token Bearer con rol admin_sys.
// @Tags Inventario
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del item" example(1)
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Item no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/inventario/items/{id} [delete]
func (h *Container) DeleteInventarioItem(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Inventario.DesactivarItem(id); err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "item desactivado correctamente"})
}

// ─── Imagen ───────────────────────────────────────────────────────────────────

// PostInventarioImagenUploadURL godoc
// @Summary Emitir URL firmada para subir imagen de un item
// @Description Devuelve una URL firmada para que el cliente suba la imagen directo al storage, sin pasar los bytes por el backend. Tras subir (PUT a upload_url), confirmar con PUT /bd/inventario/items/{id}/imagen. Requiere token Bearer con rol admin_sys.
// @Tags Inventario
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del item" example(1)
// @Success 200 {object} utils.APIResponse{data=inventarioImagenUploadResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Item no encontrado"
// @Failure 503 {object} utils.APIResponse "Almacenamiento de imagenes no configurado"
// @Security BearerAuth
// @Router /bd/inventario/items/{id}/imagen/upload-url [post]
func (h *Container) PostInventarioImagenUploadURL(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	subida, err := h.Inventario.GenerarURLSubidaImagen(id)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, inventarioImagenUploadResponse{
		UploadURL: subida.UploadURL, Token: subida.Token, Path: subida.Path, ImagenKey: subida.Path,
	})
}

// PutInventarioImagen godoc
// @Summary Confirmar imagen de item subida
// @Description Persiste la clave de la imagen del item tras una subida exitosa y devuelve su URL. Requiere token Bearer con rol admin_sys.
// @Tags Inventario
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del item" example(1)
// @Success 200 {object} utils.APIResponse{data=inventarioImagenResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Item no encontrado"
// @Failure 503 {object} utils.APIResponse "Almacenamiento de imagenes no configurado"
// @Security BearerAuth
// @Router /bd/inventario/items/{id}/imagen [put]
func (h *Container) PutInventarioImagen(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	url, err := h.Inventario.ConfirmarImagen(id)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, inventarioImagenResponse{ImagenURL: url})
}

// DeleteInventarioImagen godoc
// @Summary Quitar imagen de item
// @Description Borra el objeto del storage y limpia la clave del item. Requiere token Bearer con rol admin_sys.
// @Tags Inventario
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del item" example(1)
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Item no encontrado"
// @Failure 503 {object} utils.APIResponse "Almacenamiento de imagenes no configurado"
// @Security BearerAuth
// @Router /bd/inventario/items/{id}/imagen [delete]
func (h *Container) DeleteInventarioImagen(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Inventario.EliminarImagen(id); err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "imagen eliminada correctamente"})
}

// ─── Categorias ───────────────────────────────────────────────────────────────

// GetInventarioCategorias godoc
// @Summary Listar categorias de inventario
// @Description Lista las categorias de inventario. Requiere token Bearer.
// @Tags Inventario
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Success 200 {object} utils.APIResponse{data=categoriasInventarioResponse}
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/inventario/categorias [get]
func (h *Container) GetInventarioCategorias(c *gin.Context) {
	cats, err := h.Inventario.ListarCategorias()
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, categoriasInventarioResponse{Categorias: cats})
}

// PostInventarioCategoria godoc
// @Summary Crear categoria de inventario
// @Description Crea una categoria de inventario. El nombre es unico sin distinguir mayusculas. Requiere token Bearer con rol admin_sys.
// @Tags Inventario
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body crearCategoriaInventarioRequest true "Datos de la categoria"
// @Success 201 {object} utils.APIResponse{data=idResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: nombre requerido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 409 {object} utils.APIResponse "Ya existe una categoria con ese nombre"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/inventario/categorias [post]
func (h *Container) PostInventarioCategoria(c *gin.Context) {
	var req crearCategoriaInventarioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	id, err := h.Inventario.CrearCategoria(req.Nombre)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusCreated, idResponse{ID: id})
}

// PatchInventarioCategoria godoc
// @Summary Editar categoria de inventario
// @Description Cambia el nombre o activa/desactiva la categoria. Requiere token Bearer con rol admin_sys.
// @Tags Inventario
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID de la categoria" example(1)
// @Param payload body actualizarCategoriaInventarioRequest true "Campos a actualizar"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido, sin campos, nombre invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Categoria no encontrada"
// @Failure 409 {object} utils.APIResponse "Ya existe una categoria con ese nombre"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/inventario/categorias/{id} [patch]
func (h *Container) PatchInventarioCategoria(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var req actualizarCategoriaInventarioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Inventario.ActualizarCategoria(id, req.Nombre, req.Activo); err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "categoria actualizada correctamente"})
}

// ─── Kardex, ajustes y alertas ────────────────────────────────────────────────

// GetInventarioMovimientos godoc
// @Summary Kardex de inventario
// @Description Lista movimientos de stock (entradas, salidas por servicio, reversiones y ajustes) con el stock resultante. Filtros: item_id, local_id, tipo, fecha_desde, fecha_hasta (YYYY-MM-DD). Los usuarios no admin_sys quedan limitados a su local. Requiere token Bearer.
// @Tags Inventario
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param item_id query int false "Filtrar por item"
// @Param local_id query int false "Filtrar por local (admin_sys)"
// @Param tipo query string false "ENTRADA_COMPRA, SALIDA_SERVICIO, REVERSION_SERVICIO, AJUSTE_POSITIVO o AJUSTE_NEGATIVO"
// @Param fecha_desde query string false "Desde (YYYY-MM-DD)"
// @Param fecha_hasta query string false "Hasta (YYYY-MM-DD)"
// @Param limit query int false "Maximo de resultados (default 200, tope 500)"
// @Success 200 {object} utils.APIResponse{data=movimientosInventarioResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: parametro invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Local no permitido para el usuario"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/inventario/movimientos [get]
func (h *Container) GetInventarioMovimientos(c *gin.Context) {
	itemID, err := queryInt(c, "item_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	localPedido, err := queryInt(c, "local_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	limit, err := queryInt(c, "limit")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	desde, err := services.ParseFechaOpcional(c.Query("fecha_desde"), "fecha_desde")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	hasta, err := services.ParseFechaOpcional(c.Query("fecha_hasta"), "fecha_hasta")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	localID, ok := resolverLocalID(c, localPedido)
	if !ok {
		return
	}
	f := repository.FiltroMovimientosInventario{
		ItemID: itemID, LocalID: localID, Tipo: strings.ToUpper(strings.TrimSpace(c.Query("tipo"))),
		FechaDesde: desde, FechaHasta: hasta,
	}
	if limit != nil {
		f.Limit = *limit
	}
	movs, err := h.Inventario.ListarMovimientos(f)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, movimientosInventarioResponse{Movimientos: movs})
}

// PostInventarioAjuste godoc
// @Summary Ajustar stock
// @Description Registra un ajuste manual de stock (merma, vencimiento o conteo fisico). El motivo es obligatorio. Un ajuste negativo no puede dejar el stock bajo cero. Gerencia solo puede ajustar su local. Requiere token Bearer con rol admin_sys o gerencia.
// @Tags Inventario
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body ajusteInventarioRequest true "Datos del ajuste"
// @Success 201 {object} utils.APIResponse{data=ajusteInventarioResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: tipo invalido, cantidad no positiva, motivo requerido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario o local no autorizado"
// @Failure 404 {object} utils.APIResponse "Item no encontrado"
// @Failure 409 {object} utils.APIResponse "Stock insuficiente para el ajuste negativo"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/inventario/ajustes [post]
func (h *Container) PostInventarioAjuste(c *gin.Context) {
	var req ajusteInventarioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	local := req.LocalID
	if _, ok := resolverLocalID(c, &local); !ok {
		return
	}
	actor := consumoActor(c)
	stock, err := h.Inventario.Ajustar(services.AjusteInput{
		ItemID: req.ItemID, LocalID: req.LocalID, Tipo: strings.ToUpper(strings.TrimSpace(req.Tipo)),
		Cantidad: req.Cantidad, Motivo: req.Motivo, UsuarioID: actor.UsuarioID, UsuarioNombre: actor.UsuarioNombre,
	})
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusCreated, ajusteInventarioResponse{StockResultante: stock, Mensaje: "ajuste registrado correctamente"})
}

// GetInventarioAlertas godoc
// @Summary Alertas de inventario
// @Description Devuelve los items bajo su stock minimo y las reservas futuras con falta de insumos para su servicio. Los usuarios no admin_sys quedan limitados a su local. Requiere token Bearer.
// @Tags Inventario
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param local_id query int false "Filtrar por local (admin_sys)"
// @Success 200 {object} utils.APIResponse{data=models.AlertasInventario}
// @Failure 400 {object} utils.APIResponse "Error de validacion: local_id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Local no permitido para el usuario"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/inventario/alertas [get]
func (h *Container) GetInventarioAlertas(c *gin.Context) {
	localPedido, err := queryInt(c, "local_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	localID, ok := resolverLocalID(c, localPedido)
	if !ok {
		return
	}
	alertas, err := h.Inventario.Alertas(localID)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, alertas)
}

// ─── Insumos por servicio ─────────────────────────────────────────────────────

// GetServicioInsumos godoc
// @Summary Insumos de un servicio
// @Description Lista los items de inventario que consume el servicio por sesion. Requiere token Bearer.
// @Tags Inventario
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del servicio" example(8)
// @Success 200 {object} utils.APIResponse{data=insumosServicioResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/servicios/{id}/insumos [get]
func (h *Container) GetServicioInsumos(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	insumos, err := h.Inventario.ListarInsumos(id)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, insumosServicioResponse{Insumos: insumos})
}

// PutServicioInsumos godoc
// @Summary Reemplazar insumos de un servicio
// @Description Reemplaza la lista completa de insumos del servicio (patron de PUT /bd/combos/{id}/servicios). Una lista vacia quita todos. No se permiten items repetidos ni cantidades no positivas. Requiere token Bearer con rol admin_sys.
// @Tags Inventario
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del servicio" example(8)
// @Param payload body reemplazarInsumosRequest true "Insumos del servicio"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido, item repetido, cantidad no positiva"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Servicio o item no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/servicios/{id}/insumos [put]
func (h *Container) PutServicioInsumos(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var req reemplazarInsumosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	insumos := make([]models.ServicioInsumo, 0, len(req.Insumos))
	for _, in := range req.Insumos {
		insumos = append(insumos, models.ServicioInsumo{ItemID: in.ItemID, CantidadPorSesion: in.CantidadPorSesion})
	}
	if err := h.Inventario.ReemplazarInsumos(id, insumos); err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "insumos del servicio actualizados correctamente"})
}

