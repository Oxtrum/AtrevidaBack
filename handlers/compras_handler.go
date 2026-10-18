package handlers

import (
	"net/http"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

type comprasResponse struct {
	// Compras de inventario, mas reciente primero
	Compras []models.Compra `json:"compras"`
}

type compraResponse struct {
	// Compra con su detalle
	Compra models.Compra `json:"compra"`
}

type compraLineaRequest struct {
	// ID del item comprado
	ItemID int `json:"item_id" binding:"required" example:"1"`
	// Cantidad comprada (en la unidad del item)
	Cantidad float64 `json:"cantidad" binding:"required" example:"1000"`
	// Costo unitario
	CostoUnitario float64 `json:"costo_unitario" example:"0.04"`
}

type crearCompraRequest struct {
	// Local que recibe la compra
	LocalID int `json:"local_id" binding:"required" example:"1"`
	// Proveedor (opcional)
	Proveedor *string `json:"proveedor" example:"Distribuidora Sur"`
	// Fecha de la compra YYYY-MM-DD
	Fecha string `json:"fecha" binding:"required" example:"2026-10-18"`
	// Notas (opcional)
	Notas *string `json:"notas" example:"Factura 1234"`
	// Metodo de pago del egreso: efectivo, qr, transferencia o debito_automatico (opcional)
	MetodoPago *string `json:"metodo_pago" example:"efectivo"`
	// Lineas de la compra (al menos una)
	Detalle []compraLineaRequest `json:"detalle" binding:"required"`
}

type anularCompraRequest struct {
	// Motivo de la anulacion (obligatorio)
	Motivo string `json:"motivo" example:"Compra duplicada"`
}

// GetCompras godoc
// @Summary Listar compras de inventario
// @Description Lista las compras. Los usuarios no admin_sys quedan limitados a su local. Requiere token Bearer con rol admin_sys o gerencia.
// @Tags Compras
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param local_id query int false "Filtrar por local (admin_sys)"
// @Success 200 {object} utils.APIResponse{data=comprasResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: local_id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario o local no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/compras [get]
func (h *Container) GetCompras(c *gin.Context) {
	localPedido, err := queryInt(c, "local_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	localID, ok := resolverLocalID(c, localPedido)
	if !ok {
		return
	}
	compras, err := h.Finanzas.ListarCompras(localID)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, comprasResponse{Compras: compras})
}

// GetCompraByID godoc
// @Summary Detalle de compra
// @Description Devuelve la compra con sus lineas y el codigo del egreso asociado. Requiere token Bearer con rol admin_sys o gerencia.
// @Tags Compras
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID de la compra" example(1)
// @Success 200 {object} utils.APIResponse{data=compraResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario o local no autorizado"
// @Failure 404 {object} utils.APIResponse "Compra no encontrada"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/compras/{id} [get]
func (h *Container) GetCompraByID(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	compra, err := h.Finanzas.ObtenerCompra(id)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	local := compra.LocalID
	if _, ok := resolverLocalID(c, &local); !ok {
		return
	}
	utils.Respond(c, http.StatusOK, compraResponse{Compra: *compra})
}

// PostCompra godoc
// @Summary Registrar compra de inventario
// @Description Registra una compra en una sola transaccion: crea la compra y su detalle, suma el stock de cada item en el local, actualiza el costo de referencia y genera UN egreso VARIABLE (categoria Compras de inventario) por el total. Si falla cualquier linea no queda nada a medias. Gerencia solo puede comprar para su local. Requiere token Bearer con rol admin_sys o gerencia.
// @Tags Compras
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body crearCompraRequest true "Datos de la compra"
// @Success 201 {object} utils.APIResponse{data=compraResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: local, fecha o detalle invalidos; cantidad no positiva; item repetido o inactivo; metodo_pago invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario o local no autorizado"
// @Failure 404 {object} utils.APIResponse "Item o local no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/compras [post]
func (h *Container) PostCompra(c *gin.Context) {
	var req crearCompraRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	local := req.LocalID
	if _, ok := resolverLocalID(c, &local); !ok {
		return
	}
	actor := consumoActor(c)
	detalle := make([]services.CompraLineaInput, 0, len(req.Detalle))
	for _, l := range req.Detalle {
		detalle = append(detalle, services.CompraLineaInput{ItemID: l.ItemID, Cantidad: l.Cantidad, CostoUnitario: l.CostoUnitario})
	}
	compra, err := h.Finanzas.RegistrarCompra(services.CompraInput{
		LocalID: req.LocalID, Proveedor: req.Proveedor, Fecha: req.Fecha, Notas: req.Notas, MetodoPago: req.MetodoPago,
		Detalle: detalle, UsuarioID: actor.UsuarioID, UsuarioNombre: actor.UsuarioNombre,
	})
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusCreated, compraResponse{Compra: *compra})
}

// PatchCompraAnular godoc
// @Summary Anular compra de inventario
// @Description Anula la compra y su egreso, y revierte las entradas de stock. Si parte de ese stock ya se consumio y no alcanza para revertir, no cambia nada y responde 409 con el detalle. Requiere token Bearer con rol admin_sys.
// @Tags Compras
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID de la compra" example(1)
// @Param payload body anularCompraRequest true "Motivo de la anulacion"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido, motivo requerido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Compra no encontrada"
// @Failure 409 {object} utils.APIResponse "La compra ya esta anulada o el stock ya se consumio y no puede revertirse"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/compras/{id}/anular [patch]
func (h *Container) PatchCompraAnular(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var req anularCompraRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Finanzas.AnularCompra(id, req.Motivo, consumoActor(c)); err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "compra anulada y stock revertido correctamente"})
}
