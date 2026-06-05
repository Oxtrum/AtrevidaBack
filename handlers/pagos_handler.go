package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

type crearDetallePagoRequest struct {
	// ID opcional del servicio cobrado; puede enviarse null.
	ServicioID *int `json:"servicio_id" example:"8"`
	// Texto del servicio cobrado.
	Servicio *string `json:"servicio" example:"Limpieza facial"`
	// Precio unitario del servicio.
	PrecioUnitario *float64 `json:"precio_unitario" example:"250"`
	// Cantidad cobrada.
	Cantidad *int `json:"cantidad" example:"2"`
	// Subtotal de la linea.
	Subtotal *float64 `json:"subtotal" example:"500"`
}

type crearPagoRequest struct {
	// ID del local donde se registra el pago.
	LocalID *int `json:"local_id" example:"1"`
	// Nombre del local donde se registra el pago.
	LocalNombre *string `json:"local_nombre" example:"SAN MARTIN"`
	// ID opcional del cliente registrado; puede enviarse null.
	ClienteID *int `json:"cliente_id" example:"12"`
	// NIT del cliente.
	ClienteNIT *string `json:"cliente_nit" example:"1234567"`
	// Nombre del cliente.
	ClienteNombre *string `json:"cliente_nombre" example:"Maria Lopez"`
	// Subtotal del pago.
	Subtotal *float64 `json:"subtotal" example:"500"`
	// Descuento aplicado.
	Descuento *float64 `json:"descuento" example:"50"`
	// Total final del pago.
	TotalFinal *float64 `json:"total_final" example:"450"`
	// Tipo de pago utilizado: efectivo o qr.
	TipoPago *string `json:"tipo_pago" example:"efectivo"`
	// Estado inicial del pago.
	Estado *string `json:"estado" example:"PENDIENTE"`
	// Estado activo inicial del pago.
	Activo *bool `json:"activo" example:"true"`
	// Detalle de servicios cobrados.
	Detalle []crearDetallePagoRequest `json:"detalle"`
}

type actualizarPagoRequest struct {
	// Nuevo ID del local (opcional).
	LocalID *int `json:"local_id" example:"1"`
	// Nuevo nombre del local (opcional).
	LocalNombre *string `json:"local_nombre" example:"SAN MARTIN"`
	// Nuevo ID del cliente; puede enviarse null para limpiar la referencia.
	ClienteID *int `json:"cliente_id" example:"12"`
	// Nuevo NIT del cliente (opcional).
	ClienteNIT *string `json:"cliente_nit" example:"1234567"`
	// Nuevo nombre del cliente (opcional).
	ClienteNombre *string `json:"cliente_nombre" example:"Maria Lopez"`
	// Nuevo subtotal del pago (opcional).
	Subtotal *float64 `json:"subtotal" example:"500"`
	// Nuevo descuento del pago (opcional).
	Descuento *float64 `json:"descuento" example:"50"`
	// Nuevo total final del pago (opcional).
	TotalFinal *float64 `json:"total_final" example:"450"`
	// Nuevo tipo de pago (opcional): efectivo o qr.
	TipoPago *string `json:"tipo_pago" example:"qr"`
	// Nuevo estado del pago (opcional).
	Estado *string `json:"estado" example:"PAGADO"`
	// Nuevo estado activo (opcional).
	Activo *bool `json:"activo" example:"true"`
	// Detalle completo deseado para sincronizar; ids presentes se conservan, ids ausentes se eliminan y items sin id se crean.
	Detalle []actualizarDetallePagoRequest `json:"detalle"`
}

type actualizarDetallePagoRequest struct {
	// ID del detalle existente que debe conservarse.
	ID *int `json:"id" example:"25"`
	// ID opcional del servicio para un detalle nuevo.
	ServicioID *int `json:"servicio_id" example:"8"`
	// Texto del servicio para un detalle nuevo.
	Servicio *string `json:"servicio" example:"Limpieza facial"`
	// Precio unitario para un detalle nuevo.
	PrecioUnitario *float64 `json:"precio_unitario" example:"250"`
	// Cantidad para un detalle nuevo.
	Cantidad *int `json:"cantidad" example:"2"`
	// Subtotal para un detalle nuevo.
	Subtotal *float64 `json:"subtotal" example:"500"`
}

// GetPagos godoc
// @Summary Listar pagos
// @Description Devuelve pagos activos de BD con filtros opcionales. Requiere token Bearer. Solo retorna la informacion base del pago, sin detalle. Filtros: codigo_pago busqueda parcial, local_id, local_nombre busqueda parcial, cliente_id, cliente_nit busqueda parcial, cliente_nombre busqueda parcial, tipo_pago efectivo/qr, estado PAGADO/BORRADOR/PENDIENTE y activo true/false. Si activo no se envia, lista solo pagos activos.
// @Tags Pagos
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param codigo_pago query string false "Busqueda parcial por codigo de pago" example(PAGO-000001)
// @Param local_id query int false "ID del local" example(1)
// @Param local_nombre query string false "Busqueda parcial por nombre del local" example(SAN MARTIN)
// @Param cliente_id query int false "ID del cliente" example(12)
// @Param cliente_nit query string false "Busqueda parcial por NIT del cliente" example(1234567)
// @Param cliente_nombre query string false "Busqueda parcial por nombre del cliente" example(Maria)
// @Param tipo_pago query string false "Tipo de pago" Enums(efectivo,qr) example(efectivo)
// @Param estado query string false "Estado del pago" Enums(PAGADO,BORRADOR,PENDIENTE) example(PENDIENTE)
// @Param activo query bool false "Filtrar por activo; default true" example(true)
// @Success 200 {object} utils.APIResponse{data=pagoListResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: local_id invalido, cliente_id invalido, activo invalido, Tipo de pago invalido. Solo se aceptan pagos en efectivo y QR, estado invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/pagos [get]
func (h *Container) GetPagos(c *gin.Context) {
	localID, ok := optionalPositiveIntQuery(c, "local_id")
	if !ok {
		return
	}
	clienteID, ok := optionalPositiveIntQuery(c, "cliente_id")
	if !ok {
		return
	}
	activo, ok := optionalBoolQuery(c, "activo", true)
	if !ok {
		return
	}

	filtro := services.FiltroPagos{
		CodigoPago:    c.Query("codigo_pago"),
		LocalID:       localID,
		LocalNombre:   c.Query("local_nombre"),
		ClienteID:     clienteID,
		ClienteNIT:    c.Query("cliente_nit"),
		ClienteNombre: c.Query("cliente_nombre"),
		TipoPago:      c.Query("tipo_pago"),
		Estado:        c.Query("estado"),
		Activo:        activo,
	}

	pagos, err := h.PagosPG.GetPagos(filtro)
	if err != nil {
		utils.RespondError(c, statusPagoError(err), err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, pagoListResponse{
		Total: len(pagos),
		Filtros: pagoFiltrosResponse{
			CodigoPago:    strings.TrimSpace(c.Query("codigo_pago")),
			LocalID:       localID,
			LocalNombre:   strings.TrimSpace(c.Query("local_nombre")),
			ClienteID:     clienteID,
			ClienteNIT:    strings.TrimSpace(c.Query("cliente_nit")),
			ClienteNombre: strings.TrimSpace(c.Query("cliente_nombre")),
			TipoPago:      strings.ToLower(strings.TrimSpace(c.Query("tipo_pago"))),
			Estado:        strings.TrimSpace(c.Query("estado")),
			Activo:        activo,
		},
		Pagos: pagos,
	})
}

// GetPagoByCodigo godoc
// @Summary Obtener pago por codigo
// @Description Devuelve un pago activo por codigo_pago junto con su detalle. Requiere token Bearer. Param: codigo_pago (requerido, path). Response: pago (PagoCompletoPG con cabecera y detalle_pagos).
// @Tags Pagos
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param codigo_pago path string true "Codigo publico del pago" example(PAGO-000001)
// @Success 200 {object} utils.APIResponse{data=pagoItemResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: codigo_pago requerido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 404 {object} utils.APIResponse "Pago no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/pagos/{codigo_pago} [get]
func (h *Container) GetPagoByCodigo(c *gin.Context) {
	codigoPago := strings.TrimSpace(c.Param("codigo_pago"))
	if codigoPago == "" {
		utils.RespondError(c, http.StatusBadRequest, "codigo_pago requerido")
		return
	}

	pago, err := h.PagosPG.GetPagoByCodigo(codigoPago)
	if err != nil {
		utils.RespondError(c, statusPagoError(err), err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, pagoItemResponse{Pago: pago})
}

// CreatePago godoc
// @Summary Crear pago
// @Description Crea un pago independiente junto con su detalle en una sola transaccion. Requiere token Bearer. Todos los campos de cabecera deben venir en el body excepto codigo_pago, fecha_creacion y fecha_modificacion, que se generan en BD. tipo_pago es requerido y solo acepta efectivo o qr. subtotal y total_final son opcionales: si no se envian, se calculan desde la suma de subtotales del detalle y el descuento. cliente_id puede venir como null. Cada item de detalle debe incluir servicio_id (puede ser null), servicio, precio_unitario, cantidad y subtotal. Response: codigo_pago generado incrementalmente.
// @Tags Pagos
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body crearPagoRequest true "Datos completos del pago y su detalle"
// @Success 201 {object} utils.APIResponse{data=pagoCreatedResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: body invalido, campos requeridos, importes invalidos, Tipo de pago invalido. Solo se aceptan pagos en efectivo y QR, estado invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 404 {object} utils.APIResponse "Referencia no encontrada: local, cliente o servicio invalido"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/pagos [post]
func (h *Container) CreatePago(c *gin.Context) {
	req, ok := parseCrearPagoRequest(c)
	if !ok {
		return
	}

	detalle := make([]services.CrearDetallePagoInput, 0, len(req.Detalle))
	for _, d := range req.Detalle {
		detalle = append(detalle, services.CrearDetallePagoInput{
			ServicioID:     d.ServicioID,
			Servicio:       stringValueRequired(d.Servicio),
			PrecioUnitario: floatValueRequired(d.PrecioUnitario),
			Cantidad:       intValueRequired(d.Cantidad),
			Subtotal:       floatValueRequired(d.Subtotal),
		})
	}

	codigoPago, err := h.PagosPG.CreatePago(services.CrearPagoInput{
		LocalID:       intValueRequired(req.LocalID),
		LocalNombre:   stringValueRequired(req.LocalNombre),
		ClienteID:     req.ClienteID,
		ClienteNIT:    stringValueRequired(req.ClienteNIT),
		ClienteNombre: stringValueRequired(req.ClienteNombre),
		Subtotal:      req.Subtotal,
		Descuento:     req.Descuento,
		TotalFinal:    req.TotalFinal,
		TipoPago:      stringValueRequired(req.TipoPago),
		Estado:        stringValueRequired(req.Estado),
		Activo:        boolValueRequired(req.Activo),
		Detalle:       detalle,
	})
	if err != nil {
		utils.RespondError(c, statusPagoError(err), err.Error())
		return
	}

	utils.Respond(c, http.StatusCreated, pagoCreatedResponse{
		CodigoPago: codigoPago,
		Mensaje:    "pago creado correctamente",
	})
}

// PatchPago godoc
// @Summary Actualizar pago
// @Description Actualiza parcialmente la cabecera de un pago activo por codigo_pago y puede sincronizar detalle_pagos. Requiere token Bearer. Solo permite modificar pagos cuyo estado actual no sea PAGADO. tipo_pago es opcional, pero si se envia solo acepta efectivo o qr. Si se envia detalle, la lista representa el estado final: ids presentes se conservan, ids ausentes se eliminan y items sin id se crean. Si se envia detalle y no se envia subtotal o total_final, esos campos se recalculan automaticamente desde los subtotales del detalle final. cliente_id puede enviarse como null para limpiar la referencia.
// @Tags Pagos
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param codigo_pago path string true "Codigo publico del pago" example(PAGO-000001)
// @Param payload body actualizarPagoRequest true "Campos base a actualizar y/o detalle a sincronizar"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: codigo_pago requerido, body invalido, sin cambios, campos invalidos, Tipo de pago invalido. Solo se aceptan pagos en efectivo y QR, estado invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 404 {object} utils.APIResponse "Pago no encontrado"
// @Failure 409 {object} utils.APIResponse "Conflicto: no se puede modificar un pago en estado PAGADO"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/pagos/{codigo_pago} [patch]
func (h *Container) PatchPago(c *gin.Context) {
	codigoPago := strings.TrimSpace(c.Param("codigo_pago"))
	if codigoPago == "" {
		utils.RespondError(c, http.StatusBadRequest, "codigo_pago requerido")
		return
	}

	req, clienteIDSet, detalleSet, ok := parseActualizarPagoRequest(c)
	if !ok {
		return
	}

	var detalle *[]services.ActualizarDetallePagoInput
	if detalleSet {
		parsedDetalle := make([]services.ActualizarDetallePagoInput, 0, len(req.Detalle))
		for _, d := range req.Detalle {
			parsed := services.ActualizarDetallePagoInput{
				ID:         d.ID,
				ServicioID: d.ServicioID,
			}
			if d.Servicio != nil {
				parsed.Servicio = stringValueRequired(d.Servicio)
			}
			if d.PrecioUnitario != nil {
				parsed.PrecioUnitario = floatValueRequired(d.PrecioUnitario)
			}
			if d.Cantidad != nil {
				parsed.Cantidad = intValueRequired(d.Cantidad)
			}
			if d.Subtotal != nil {
				parsed.Subtotal = floatValueRequired(d.Subtotal)
			}
			parsedDetalle = append(parsedDetalle, parsed)
		}
		detalle = &parsedDetalle
	}

	err := h.PagosPG.UpdatePago(services.ActualizarPagoInput{
		CodigoPago:    codigoPago,
		LocalID:       req.LocalID,
		LocalNombre:   req.LocalNombre,
		ClienteID:     req.ClienteID,
		ClienteIDSet:  clienteIDSet,
		ClienteNIT:    req.ClienteNIT,
		ClienteNombre: req.ClienteNombre,
		Subtotal:      req.Subtotal,
		Descuento:     req.Descuento,
		TotalFinal:    req.TotalFinal,
		TipoPago:      req.TipoPago,
		Estado:        req.Estado,
		Activo:        req.Activo,
		Detalle:       detalle,
	})
	if err != nil {
		utils.RespondError(c, statusPagoError(err), err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "pago actualizado correctamente"})
}

// DeletePago godoc
// @Summary Eliminar pago
// @Description Realiza borrado logico de un pago activo por codigo_pago (activo=false). Requiere token Bearer. No elimina fisicamente la cabecera ni el detalle.
// @Tags Pagos
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param codigo_pago path string true "Codigo publico del pago" example(PAGO-000001)
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: codigo_pago requerido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 404 {object} utils.APIResponse "Pago no encontrado o ya inactivo"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Router /bd/pagos/{codigo_pago} [delete]
func (h *Container) DeletePago(c *gin.Context) {
	codigoPago := strings.TrimSpace(c.Param("codigo_pago"))
	if codigoPago == "" {
		utils.RespondError(c, http.StatusBadRequest, "codigo_pago requerido")
		return
	}

	err := h.PagosPG.DeletePago(codigoPago)
	if err != nil {
		utils.RespondError(c, statusPagoError(err), err.Error())
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "pago eliminado correctamente"})
}

func parseCrearPagoRequest(c *gin.Context) (crearPagoRequest, bool) {
	var req crearPagoRequest
	raw, err := c.GetRawData()
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return req, false
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return req, false
	}
	for _, field := range []string{"local_id", "local_nombre", "cliente_id", "cliente_nit", "cliente_nombre", "descuento", "tipo_pago", "estado", "activo", "detalle"} {
		if _, exists := body[field]; !exists {
			utils.RespondError(c, http.StatusBadRequest, field+" es requerido")
			return req, false
		}
	}

	var detalleRaw []map[string]json.RawMessage
	if err := json.Unmarshal(body["detalle"], &detalleRaw); err != nil || len(detalleRaw) == 0 {
		utils.RespondError(c, http.StatusBadRequest, "detalle es requerido")
		return req, false
	}
	for i, item := range detalleRaw {
		for _, field := range []string{"servicio_id", "servicio", "precio_unitario", "cantidad", "subtotal"} {
			if _, exists := item[field]; !exists {
				utils.RespondError(c, http.StatusBadRequest, "detalle["+strconv.Itoa(i)+"]."+field+" es requerido")
				return req, false
			}
		}
	}

	if err := json.Unmarshal(raw, &req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return req, false
	}

	if req.LocalID == nil || req.LocalNombre == nil || req.ClienteNIT == nil || req.ClienteNombre == nil ||
		req.Descuento == nil || req.TipoPago == nil || req.Estado == nil || req.Activo == nil {
		utils.RespondError(c, http.StatusBadRequest, "campos requeridos no pueden ser null")
		return req, false
	}
	for i, item := range req.Detalle {
		if item.Servicio == nil || item.PrecioUnitario == nil || item.Cantidad == nil || item.Subtotal == nil {
			utils.RespondError(c, http.StatusBadRequest, "detalle["+strconv.Itoa(i)+"] tiene campos requeridos null")
			return req, false
		}
	}

	return req, true
}

func parseActualizarPagoRequest(c *gin.Context) (actualizarPagoRequest, bool, bool, bool) {
	var req actualizarPagoRequest
	raw, err := c.GetRawData()
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return req, false, false, false
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return req, false, false, false
	}
	if len(body) == 0 {
		utils.RespondError(c, http.StatusBadRequest, "debe enviar al menos un campo para actualizar")
		return req, false, false, false
	}

	if rawDetalle, exists := body["detalle"]; exists && strings.TrimSpace(string(rawDetalle)) == "null" {
		utils.RespondError(c, http.StatusBadRequest, "detalle debe ser una lista")
		return req, false, false, false
	}
	if rawTipoPago, exists := body["tipo_pago"]; exists && strings.TrimSpace(string(rawTipoPago)) == "null" {
		utils.RespondError(c, http.StatusBadRequest, "tipo_pago no puede ser null")
		return req, false, false, false
	}

	if err := json.Unmarshal(raw, &req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return req, false, false, false
	}

	_, clienteIDSet := body["cliente_id"]
	_, detalleSet := body["detalle"]
	return req, clienteIDSet, detalleSet, true
}

func optionalPositiveIntQuery(c *gin.Context, name string) (*int, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, true
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		utils.RespondError(c, http.StatusBadRequest, name+" invalido")
		return nil, false
	}

	return &value, true
}

func optionalBoolQuery(c *gin.Context, name string, defaultValue bool) (*bool, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return &defaultValue, true
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, name+" invalido")
		return nil, false
	}

	return &value, true
}

func statusPagoError(err error) int {
	lower := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lower, "no encontrado"):
		return http.StatusNotFound
	case strings.Contains(lower, "estado pagado"):
		return http.StatusConflict
	case strings.Contains(lower, "referencia no encontrada"):
		return http.StatusNotFound
	case strings.Contains(lower, "invalido"),
		strings.Contains(lower, "inválido"),
		strings.Contains(lower, "requerido"),
		strings.Contains(lower, "negativo"),
		strings.Contains(lower, "mayor"),
		strings.Contains(lower, "debe enviar"),
		strings.Contains(lower, "no puede"):
		return http.StatusBadRequest
	case strings.Contains(lower, "ya existe"):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func stringValueRequired(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func floatValueRequired(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func intValueRequired(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func boolValueRequired(value *bool) bool {
	return value != nil && *value
}
