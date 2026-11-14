package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/pagination"
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
	// Paginacion por cursor (solo si se envia cursor o un limit de 1 a 100)
	Paginacion *pagination.Metadata `json:"paginacion,omitempty"`
	// Total de items con los filtros (solo con paginacion e include_total=true)
	TotalRegistros *int `json:"total_registros,omitempty"`
}

// itemsFiltrosCursor ata el cursor del listado de items a sus filtros.
type itemsFiltrosCursor struct {
	Q           string `json:"q,omitempty"`
	CategoriaID *int   `json:"categoria_id,omitempty"`
	LocalID     *int   `json:"local_id,omitempty"`
	BajoStock   bool   `json:"bajo_stock,omitempty"`
	Activo      *bool  `json:"activo,omitempty"`
	PorLocal    bool   `json:"por_local,omitempty"`
}

// itemsSePaginan decide si el listado de items va por paginas: con cursor, o con un limit que cabe en una pagina
// (1 a 100). Sin limit, o con uno mayor, devuelve todo hasta el tope (200 por defecto, 500 como maximo), que es lo
// que usan los selectores. Un limit que no es un numero tambien pasa al paginador, que lo rechaza con 400.
func itemsSePaginan(limit, cursor string) bool {
	if strings.TrimSpace(cursor) != "" {
		return true
	}
	limit = strings.TrimSpace(limit)
	if limit == "" {
		return false
	}
	n, err := strconv.Atoi(limit)
	return err != nil || n <= pagination.MaxLimit
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
	// Unidad de medida: UNIDAD, ML, GR, LT, KG, FRASCO, PAQUETE, SOBRE o CAJA (default UNIDAD)
	UnidadMedida string `json:"unidad_medida" example:"ML"`
	// Ultimo costo de compra (se actualiza solo con cada compra)
	CostoReferencia *float64 `json:"costo_referencia" example:"0.04"`
	// Precio de venta del insumo (opcional); es el que se sugiere al cobrarlo
	PrecioVenta *float64 `json:"precio_venta" example:"25"`
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
	// Unidad de medida: UNIDAD, ML, GR, LT, KG, FRASCO, PAQUETE, SOBRE o CAJA
	UnidadMedida *string `json:"unidad_medida" example:"ML"`
	// Costo de referencia (null limpia)
	CostoReferencia *float64 `json:"costo_referencia" example:"0.04"`
	// Precio de venta del insumo (null limpia)
	PrecioVenta *float64 `json:"precio_venta" example:"25"`
	// Stock minimo
	StockMinimo *float64 `json:"stock_minimo" example:"500"`
	// Activo o desactivado (RN-07)
	Activo *bool `json:"activo" example:"true"`
}

type inventarioImagenUploadResponse struct {
	// URL firmada a la que el cliente hace PUT con el archivo
	UploadURL string `json:"upload_url" example:"https://bucket.s3.sa-east-1.amazonaws.com/inventario/1/3f2a9c.webp?X-Amz-Signature=abc"`
	// Token de la subida firmada (si el storage lo usa)
	Token string `json:"token" example:"abc"`
	// Path del objeto en el bucket
	Path string `json:"path" example:"inventario/1/3f2a9c1e5b7d4a80b6c2d9e1f0a47b53.webp"`
	// Clave del objeto (igual a path); es lo que el cliente debe confirmar
	ImagenKey string `json:"imagen_key" example:"inventario/1/3f2a9c1e5b7d4a80b6c2d9e1f0a47b53.webp"`
	// Headers que el cliente debe enviar tal cual en el PUT (van firmados en la URL)
	Headers map[string]string `json:"headers,omitempty"`
}

type inventarioImagenConfirmRequest struct {
	// Clave devuelta por upload-url tras subir el archivo
	ImagenKey string `json:"imagen_key" binding:"required" example:"inventario/1/3f2a9c1e5b7d4a80b6c2d9e1f0a47b53.webp"`
}

type inventarioImagenResponse struct {
	// URL publica de la imagen
	ImagenURL string `json:"imagen_url" example:"https://bucket.s3.sa-east-1.amazonaws.com/inventario/1/3f2a9c1e5b7d4a80b6c2d9e1f0a47b53.webp"`
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
	// Paginacion por cursor (solo si se envia limit o cursor)
	Paginacion *pagination.Metadata `json:"paginacion,omitempty"`
	// Total de movimientos con los filtros (solo con include_total=true)
	TotalRegistros *int `json:"total_registros,omitempty"`
}

type consumosResponse struct {
	// Eventos de consumo declarado en atenciones activas
	Consumos []models.ConsumoEvento `json:"consumos"`
	// Paginacion por cursor (solo si se envia limit o cursor)
	Paginacion *pagination.Metadata `json:"paginacion,omitempty"`
	// Total de eventos con los filtros (solo con paginacion e include_total=true)
	TotalRegistros *int `json:"total_registros,omitempty"`
}

type consumosResumenResponse struct {
	// Totales por insumo y unidad
	Items []models.ConsumoResumen `json:"items"`
}

type consumosEmpleadosResponse struct {
	// Totales operativos por empleado
	Items []models.ConsumoEmpleadoResumen `json:"items"`
}

// consumosFiltrosCursor ata el cursor a los filtros del calendario.
type consumosFiltrosCursor struct {
	LocalID         *int   `json:"local_id,omitempty"`
	FechaDesde      string `json:"fecha_desde,omitempty"`
	FechaHasta      string `json:"fecha_hasta,omitempty"`
	ItemID          *int   `json:"item_id,omitempty"`
	Cliente         string `json:"cliente,omitempty"`
	Servicio        string `json:"servicio,omitempty"`
	Empleado        string `json:"empleado,omitempty"`
	EmpleadoID      *int   `json:"empleado_id,omitempty"`
	EstadoDescuento string `json:"estado_descuento,omitempty"`
	Q               string `json:"q,omitempty"`
}

// movimientosFiltrosCursor ata el cursor a los filtros: cambiar un filtro invalida el cursor.
type movimientosFiltrosCursor struct {
	ItemID     *int   `json:"item_id,omitempty"`
	LocalID    *int   `json:"local_id,omitempty"`
	Tipo       string `json:"tipo,omitempty"`
	FechaDesde string `json:"fecha_desde,omitempty"`
	FechaHasta string `json:"fecha_hasta,omitempty"`
	Q          string `json:"q,omitempty"`
	// Orden pedido; vacio con el orden por defecto. Va en el cursor para rechazar uno generado con otro orden.
	OrdenarPor string `json:"ordenar_por,omitempty"`
	Sentido    string `json:"sentido,omitempty"`
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
// @Param limit query int false "Sin cursor: 1 a 100 pagina; mayor a 100 o ausente devuelve todo hasta el tope (default 200, tope 500)"
// @Param cursor query string false "Cursor opaco devuelto en paginacion.next_cursor; solo vale con los mismos filtros"
// @Param include_total query bool false "Con paginacion, incluye total_registros y total_paginas"
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
	porLocal, err := queryBool(c, "por_local")
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
		PorLocal: porLocal != nil && *porLocal,
	}
	if !itemsSePaginan(c.Query("limit"), c.Query("cursor")) {
		if limit != nil {
			f.Limit = *limit
		}
		items, err := h.Inventario.ListarItems(f)
		if err != nil {
			respondTrastiendaError(c, err)
			return
		}
		utils.Respond(c, http.StatusOK, inventarioItemsResponse{Items: items})
		return
	}

	page, ok := parsePagination(c)
	if !ok {
		return
	}
	includeTotal, ok := parseIncludeTotal(c)
	if !ok {
		return
	}
	filters := itemsFiltrosCursor{Q: f.Q, CategoriaID: f.CategoriaID, LocalID: f.LocalID, BajoStock: f.BajoStock, Activo: f.Activo, PorLocal: f.PorLocal}
	after, ok := decodeOffsetCursor(c, page, "inventario_items", filters)
	if !ok {
		return
	}
	f.Offset = after.Offset
	f.PageLimit = page.QueryLimit()
	items, err := h.Inventario.ListarItems(f)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	var total *int
	if includeTotal {
		n, err := h.Inventario.ContarItems(f)
		if err != nil {
			respondTrastiendaError(c, err)
			return
		}
		total = &n
	}
	items, metadata, err := pagination.Build(items, page, "inventario_items", filters, func(models.InventarioItem) any {
		return pagination.OffsetPosition{Offset: after.Offset + page.Limit}
	})
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if total != nil {
		pagination.AddTotal(metadata, *total)
	}
	utils.Respond(c, http.StatusOK, inventarioItemsResponse{Items: items, Paginacion: metadata, TotalRegistros: total})
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
		CostoReferencia: req.CostoReferencia, PrecioVenta: req.PrecioVenta, StockMinimo: req.StockMinimo,
	})
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusCreated, inventarioItemResponse{Item: *item})
}

// PatchInventarioItem godoc
// @Summary Editar item de inventario
// @Description Edita datos del item. No edita stock: el stock solo cambia con movimientos (compras, ajustes, servicios). Los campos descripcion, categoria_id, costo_referencia y precio_venta aceptan null para limpiar. Requiere token Bearer con rol admin_sys.
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
		PrecioVenta: req.PrecioVenta, PrecioVentaSet: present["precio_venta"],
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
// @Description Devuelve una URL firmada para que el cliente suba la imagen (webp) directo al storage, sin pasar los bytes por el backend. La clave es nueva en cada llamada (inventario/{id}/{uuid}.webp). El PUT a upload_url debe enviar exactamente los headers devueltos. Tras subir, confirmar con PUT /bd/inventario/items/{id}/imagen enviando imagen_key. Requiere token Bearer con rol admin_sys.
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
		UploadURL: subida.UploadURL, Token: subida.Token, Path: subida.Path, ImagenKey: subida.Path, Headers: subida.Headers,
	})
}

// PutInventarioImagen godoc
// @Summary Confirmar imagen de item subida
// @Description Persiste la clave de la imagen del item tras una subida exitosa, borra la imagen anterior del storage y devuelve la URL. La clave debe ser la emitida por upload-url para este mismo item. Requiere token Bearer con rol admin_sys.
// @Tags Inventario
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del item" example(1)
// @Param body body inventarioImagenConfirmRequest true "Clave de la imagen subida"
// @Success 200 {object} utils.APIResponse{data=inventarioImagenResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido, imagen_key faltante o que no corresponde al item"
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
	var req inventarioImagenConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "imagen_key es requerido")
		return
	}
	url, err := h.Inventario.ConfirmarImagen(id, req.ImagenKey)
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
// @Param tipo query string false "ENTRADA_COMPRA, SALIDA_SERVICIO, REVERSION_SERVICIO, SALIDA_VENTA, REVERSION_VENTA, AJUSTE_POSITIVO o AJUSTE_NEGATIVO"
// @Param fecha_desde query string false "Desde (YYYY-MM-DD)"
// @Param fecha_hasta query string false "Hasta (YYYY-MM-DD)"
// @Param q query string false "Busca por item, local o motivo"
// @Param ordenar_por query string false "Campo por el que ordenar; sin el, creado_en descendente" Enums(creado_en,item_nombre,local_nombre,tipo,cantidad,stock_resultante,costo_unitario,motivo,usuario_nombre) example(stock_resultante)
// @Param sentido query string false "Sentido del orden; solo con ordenar_por, por defecto asc" Enums(asc,desc) example(desc)
// @Param limit query int false "Tamano de pagina (1-100); sin limit ni cursor devuelve los 200 mas recientes"
// @Param cursor query string false "Cursor opaco devuelto en paginacion.next_cursor; solo vale con el mismo ordenar_por y sentido que lo generaron"
// @Param include_total query bool false "Incluye total_registros y total_paginas"
// @Success 200 {object} utils.APIResponse{data=movimientosInventarioResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: parametro invalido, ordenar_por fuera de la lista, sentido distinto de asc o desc, cursor de otro orden"
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
	page, ok := parsePagination(c)
	if !ok {
		return
	}
	sort, ok := parseSort(c, pgsqlrepo.MovimientosOrdenables)
	if !ok {
		return
	}
	includeTotal, ok := parseIncludeTotal(c)
	if !ok {
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
		FechaDesde: desde, FechaHasta: hasta, Q: strings.TrimSpace(c.Query("q")),
	}
	filters := movimientosFiltrosCursor{
		ItemID: itemID, LocalID: localID, Tipo: f.Tipo, Q: f.Q,
		FechaDesde: c.Query("fecha_desde"), FechaHasta: c.Query("fecha_hasta"),
	}
	var after timeCursor
	var afterOffset pagination.OffsetPosition
	if sort.Active() {
		// Con orden pedido el cursor cuenta filas entregadas; el orden va en los filtros del cursor.
		filters.OrdenarPor, filters.Sentido = sort.Key, sort.Direction()
		f.OrdenarPor, f.OrdenDesc = sort.Key, sort.Desc
		if afterOffset, ok = decodeOffsetCursor(c, page, "inventario_movimientos", filters); !ok {
			return
		}
		f.Offset = afterOffset.Offset
	} else {
		if !decodePaginationCursor(c, page, "inventario_movimientos", filters, &after) {
			return
		}
		if page.Cursor != "" && after.ID < 1 {
			utils.RespondError(c, http.StatusBadRequest, "paginacion invalida: cursor incompleto")
			return
		}
		f.CursorSet = page.Cursor != ""
		f.CursorTime = after.CreatedAt
		f.CursorID = after.ID
	}
	f.PageLimit = page.QueryLimit()

	movs, err := h.Inventario.ListarMovimientos(f)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	var total *int
	if includeTotal && page.Enabled {
		n, err := h.Inventario.ContarMovimientos(f)
		if err != nil {
			respondTrastiendaError(c, err)
			return
		}
		total = &n
	}
	movs, metadata, err := pagination.Build(movs, page, "inventario_movimientos", filters, func(m models.MovimientoInventario) any {
		if sort.Active() {
			return pagination.OffsetPosition{Offset: afterOffset.Offset + page.Limit}
		}
		return timeCursor{CreatedAt: m.CreadoEn, ID: m.ID}
	})
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if total != nil {
		pagination.AddTotal(metadata, *total)
	}
	utils.Respond(c, http.StatusOK, movimientosInventarioResponse{Movimientos: movs, Paginacion: metadata, TotalRegistros: total})
}

// GetInventarioConsumos godoc
// @Summary Calendario de consumos
// @Description Lista los eventos de consumo declarado en atenciones activas. Usa atenciones.fecha_atencion (no la fecha del movimiento de stock) y atencion_insumos.cantidad_usada. Filtros: fecha_desde y fecha_hasta (YYYY-MM-DD, inclusivos; por defecto hoy), local_id, item_id, cliente, servicio, empleado (texto), empleado_id, estado_descuento (DESCONTADO o PENDIENTE) y q (cliente, reserva, servicio, personal, local o insumo). Los usuarios con local asignado quedan limitados a ese local. Si falta hora_inicio se usa la hora de la reserva y se marca horario_programado=true. Las atenciones desactivadas no aparecen.
// @Tags Inventario
// @Produce json
// @Security BearerAuth
// @Param fecha_desde query string false "Desde (YYYY-MM-DD); por defecto hoy" example(2026-11-09)
// @Param fecha_hasta query string false "Hasta (YYYY-MM-DD); por defecto fecha_desde" example(2026-11-14)
// @Param local_id query int false "Filtrar por local (admin_sys)"
// @Param item_id query int false "Filtrar atenciones que usaron este insumo"
// @Param cliente query string false "Busqueda por cliente"
// @Param servicio query string false "Busqueda por servicio"
// @Param empleado query string false "Busqueda por nombre de personal"
// @Param empleado_id query int false "Filtrar por personal"
// @Param estado_descuento query string false "Estado de deduccion" Enums(DESCONTADO,PENDIENTE) example(PENDIENTE)
// @Param q query string false "Busqueda general por cliente, reserva, servicio, personal, local o insumo"
// @Param limit query int false "Tamano de pagina (1-100); sin limit ni cursor devuelve hasta 500 eventos"
// @Param cursor query string false "Cursor opaco devuelto en paginacion.next_cursor; solo vale con los mismos filtros"
// @Param include_total query bool false "Incluye total_registros y total_paginas (requiere limit o cursor)"
// @Success 200 {object} utils.APIResponse{data=consumosResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: fecha invalida, estado_descuento fuera de la lista, parametro invalido o cursor de otros filtros"
// @Failure 401 {object} utils.APIResponse "Token requerido"
// @Failure 403 {object} utils.APIResponse "Sin acceso al local"
// @Failure 500 {object} utils.APIResponse "Error al consultar consumos"
// @Router /bd/inventario/consumos [get]
func (h *Container) GetInventarioConsumos(c *gin.Context) {
	localPedido, err := queryInt(c, "local_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	localID, ok := resolverLocalID(c, localPedido)
	if !ok {
		return
	}
	page, ok := parsePagination(c)
	if !ok {
		return
	}
	includeTotal, ok := parseIncludeTotal(c)
	if !ok {
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
	itemID, err := queryInt(c, "item_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	empleadoID, err := queryInt(c, "empleado_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	f, err := h.Inventario.PrepararConsumos(services.ConsumosInput{
		LocalID: localID, FechaDesde: desde, FechaHasta: hasta, ItemID: itemID,
		Cliente: c.Query("cliente"), Servicio: c.Query("servicio"), Empleado: c.Query("empleado"),
		EmpleadoID: empleadoID, EstadoDescuento: c.Query("estado_descuento"), Q: c.Query("q"),
		PageLimit: page.QueryLimit(),
	})
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	filters := consumosFiltrosCursor{
		LocalID: f.LocalID, FechaDesde: f.FechaDesde.Format("2006-01-02"), FechaHasta: f.FechaHasta.Format("2006-01-02"),
		ItemID: f.ItemID, Cliente: f.Cliente, Servicio: f.Servicio, Empleado: f.Empleado,
		EmpleadoID: f.EmpleadoID, EstadoDescuento: f.EstadoDescuento, Q: f.Q,
	}
	afterOffset, ok := decodeOffsetCursor(c, page, "inventario_consumos", filters)
	if !ok {
		return
	}
	f.Offset = afterOffset.Offset

	consumos, err := h.Inventario.ListarConsumos(f)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	var total *int
	if includeTotal && page.Enabled {
		n, err := h.Inventario.ContarConsumos(f)
		if err != nil {
			respondTrastiendaError(c, err)
			return
		}
		total = &n
	}
	consumos, metadata, err := pagination.Build(consumos, page, "inventario_consumos", filters, func(models.ConsumoEvento) any {
		return pagination.OffsetPosition{Offset: afterOffset.Offset + page.Limit}
	})
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if total != nil {
		pagination.AddTotal(metadata, *total)
	}
	utils.Respond(c, http.StatusOK, consumosResponse{Consumos: consumos, Paginacion: metadata, TotalRegistros: total})
}

// GetInventarioConsumosResumen godoc
// @Summary Reporte agregado de consumos
// @Description Agrega el consumo declarado por insumo y unidad para el rango y filtros indicados. No suma unidades distintas y cuenta atenciones distintas. Los filtros son los mismos del calendario; estado_descuento filtra las lineas que componen el total. Las reversiones de stock no alteran cantidad_usada.
// @Tags Inventario
// @Produce json
// @Security BearerAuth
// @Param fecha_desde query string false "Desde (YYYY-MM-DD); por defecto hoy" example(2026-11-01)
// @Param fecha_hasta query string false "Hasta (YYYY-MM-DD); por defecto fecha_desde" example(2026-11-30)
// @Param local_id query int false "Filtrar por local (admin_sys)"
// @Param item_id query int false "Filtrar por insumo"
// @Param cliente query string false "Busqueda por cliente"
// @Param servicio query string false "Busqueda por servicio"
// @Param empleado query string false "Busqueda por nombre de personal"
// @Param empleado_id query int false "Filtrar por personal"
// @Param estado_descuento query string false "Estado de deduccion" Enums(DESCONTADO,PENDIENTE) example(PENDIENTE)
// @Param q query string false "Busqueda general por cliente, reserva, servicio, personal, local o insumo"
// @Success 200 {object} utils.APIResponse{data=consumosResumenResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: fecha invalida o estado_descuento fuera de la lista"
// @Failure 401 {object} utils.APIResponse "Token requerido"
// @Failure 403 {object} utils.APIResponse "Sin acceso al local"
// @Failure 500 {object} utils.APIResponse "Error al resumir consumos"
// @Router /bd/inventario/consumos/resumen [get]
func (h *Container) GetInventarioConsumosResumen(c *gin.Context) {
	localPedido, err := queryInt(c, "local_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	localID, ok := resolverLocalID(c, localPedido)
	if !ok {
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
	itemID, err := queryInt(c, "item_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	empleadoID, err := queryInt(c, "empleado_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	f, err := h.Inventario.PrepararConsumos(services.ConsumosInput{
		LocalID: localID, FechaDesde: desde, FechaHasta: hasta, ItemID: itemID,
		Cliente: c.Query("cliente"), Servicio: c.Query("servicio"), Empleado: c.Query("empleado"),
		EmpleadoID: empleadoID, EstadoDescuento: c.Query("estado_descuento"), Q: c.Query("q"),
	})
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	items, err := h.Inventario.ResumenConsumos(f)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, consumosResumenResponse{Items: items})
}

// GetInventarioConsumosEmpleados godoc
// @Summary Reporte de consumos por empleado
// @Description Agrega la actividad de consumo por empleado para el rango y filtros indicados. No suma cantidades entre unidades distintas: cuenta atenciones, lineas de insumo, insumos distintos, lineas pendientes y atenciones con deduccion pendiente. Los filtros son los mismos del calendario; estado_descuento filtra las lineas que componen los conteos. Las atenciones sin insumos se conservan cuando no se filtra por estado. Los empleados eliminados se agrupan por el nombre snapshot.
// @Tags Inventario
// @Produce json
// @Security BearerAuth
// @Param fecha_desde query string false "Desde (YYYY-MM-DD); por defecto hoy" example(2026-11-01)
// @Param fecha_hasta query string false "Hasta (YYYY-MM-DD); por defecto fecha_desde" example(2026-11-30)
// @Param local_id query int false "Filtrar por local (admin_sys)"
// @Param item_id query int false "Filtrar por insumo"
// @Param cliente query string false "Busqueda por cliente"
// @Param servicio query string false "Busqueda por servicio"
// @Param empleado query string false "Busqueda por nombre de personal"
// @Param empleado_id query int false "Filtrar por personal"
// @Param estado_descuento query string false "Estado de deduccion" Enums(DESCONTADO,PENDIENTE) example(PENDIENTE)
// @Param q query string false "Busqueda general por cliente, reserva, servicio, personal, local o insumo"
// @Success 200 {object} utils.APIResponse{data=consumosEmpleadosResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: fecha invalida o estado_descuento fuera de la lista"
// @Failure 401 {object} utils.APIResponse "Token requerido"
// @Failure 403 {object} utils.APIResponse "Sin acceso al local"
// @Failure 500 {object} utils.APIResponse "Error al resumir consumos por empleado"
// @Router /bd/inventario/consumos/empleados [get]
func (h *Container) GetInventarioConsumosEmpleados(c *gin.Context) {
	localPedido, err := queryInt(c, "local_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	localID, ok := resolverLocalID(c, localPedido)
	if !ok {
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
	itemID, err := queryInt(c, "item_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	empleadoID, err := queryInt(c, "empleado_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	f, err := h.Inventario.PrepararConsumos(services.ConsumosInput{
		LocalID: localID, FechaDesde: desde, FechaHasta: hasta, ItemID: itemID,
		Cliente: c.Query("cliente"), Servicio: c.Query("servicio"), Empleado: c.Query("empleado"),
		EmpleadoID: empleadoID, EstadoDescuento: c.Query("estado_descuento"), Q: c.Query("q"),
	})
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	items, err := h.Inventario.ResumenConsumosPorEmpleado(f)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, consumosEmpleadosResponse{Items: items})
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
// @Description Reemplaza la lista completa de insumos del servicio (patron de PUT /bd/combos/{id}/servicios). Una lista vacia quita todos. No se permiten items repetidos ni cantidades no positivas. Requiere token Bearer (mismo acceso que editar el servicio).
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
