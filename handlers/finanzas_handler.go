package handlers

import (
	"net/http"
	"strings"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

type categoriasFinancierasResponse struct {
	// Categorias financieras
	Categorias []models.CategoriaFinanciera `json:"categorias"`
}

type crearCategoriaFinancieraRequest struct {
	// Nombre de la categoria
	Nombre string `json:"nombre" binding:"required" example:"Alquiler"`
	// INGRESO o EGRESO
	Naturaleza string `json:"naturaleza" binding:"required" example:"EGRESO"`
}

type actualizarCategoriaFinancieraRequest struct {
	// Nuevo nombre
	Nombre *string `json:"nombre" example:"Alquiler de local"`
	// Activa o inactiva
	Activo *bool `json:"activo" example:"true"`
}

type movimientosFinancierosResponse struct {
	// Egresos y otros ingresos, mas reciente primero
	Movimientos []models.MovimientoFinanciero `json:"movimientos"`
}

type movimientoFinancieroResponse struct {
	// Movimiento financiero
	Movimiento models.MovimientoFinanciero `json:"movimiento"`
}

type crearMovimientoFinancieroRequest struct {
	// INGRESO (otro ingreso) o EGRESO
	Naturaleza string `json:"naturaleza" binding:"required" example:"EGRESO"`
	// Para egresos: solo VARIABLE (los FIJO se generan desde un gasto fijo)
	Clase string `json:"clase" example:"VARIABLE"`
	// Categoria financiera (debe coincidir con la naturaleza)
	CategoriaID int `json:"categoria_id" binding:"required" example:"6"`
	// Local; null = gasto general de la empresa
	LocalID *int `json:"local_id" example:"1"`
	// Fecha YYYY-MM-DD
	Fecha string `json:"fecha" binding:"required" example:"2026-10-18"`
	// Monto mayor a 0
	Monto float64 `json:"monto" binding:"required" example:"480"`
	// Descripcion (opcional)
	Descripcion *string `json:"descripcion" example:"Reparacion de camilla"`
	// Proveedor o beneficiario (opcional)
	Proveedor *string `json:"proveedor" example:"Taller Medina"`
	// efectivo, qr, transferencia o debito_automatico (opcional)
	MetodoPago *string `json:"metodo_pago" example:"efectivo"`
	// PAGADO (default) o PENDIENTE
	Estado string `json:"estado" example:"PAGADO"`
}

type actualizarMovimientoFinancieroRequest struct {
	// Nueva categoria
	CategoriaID *int `json:"categoria_id" example:"6"`
	// Nuevo local (null = general)
	LocalID *int `json:"local_id" example:"1"`
	// Nueva fecha YYYY-MM-DD
	Fecha *string `json:"fecha" example:"2026-10-18"`
	// Nuevo monto
	Monto *float64 `json:"monto" example:"520"`
	// Descripcion (null limpia)
	Descripcion *string `json:"descripcion" example:"Reparacion de camilla"`
	// Proveedor (null limpia)
	Proveedor *string `json:"proveedor" example:"Taller Medina"`
	// Metodo de pago (null limpia)
	MetodoPago *string `json:"metodo_pago" example:"qr"`
	// PAGADO o PENDIENTE
	Estado *string `json:"estado" example:"PAGADO"`
}

type cambiarEstadoMovimientoRequest struct {
	// PAGADO o ANULADO
	Estado string `json:"estado" binding:"required" example:"ANULADO"`
	// Motivo, obligatorio al anular
	Motivo *string `json:"motivo" example:"Registrado por error"`
}

type gastosFijosResponse struct {
	// Plantillas de gasto fijo
	GastosFijos []models.GastoFijo `json:"gastos_fijos"`
}

type gastoFijoResponse struct {
	// Plantilla de gasto fijo
	GastoFijo models.GastoFijo `json:"gasto_fijo"`
}

type gastosFijosProximosResponse struct {
	// Cobros que vencen dentro de los dias de aviso configurados
	Proximos []models.GastoFijoProximo `json:"proximos"`
}

type generarGastosFijosResponse struct {
	// Cantidad de egresos generados en esta ejecucion
	Generados int `json:"generados" example:"2"`
	// Mensaje descriptivo
	Mensaje string `json:"mensaje" example:"gastos fijos al dia"`
}

type gastoFijoRequest struct {
	// Nombre de la plantilla
	Nombre string `json:"nombre" binding:"required" example:"Alquiler Paseo Aranjuez"`
	// Categoria financiera de egreso
	CategoriaID int `json:"categoria_id" binding:"required" example:"1"`
	// Monto mensual
	Monto float64 `json:"monto" binding:"required" example:"6500"`
	// Dia de cobro 1-31 (si el mes no tiene ese dia se usa el ultimo)
	DiaCobro int `json:"dia_cobro" binding:"required" example:"5"`
	// Local; null = gasto general
	LocalID *int `json:"local_id" example:"1"`
	// Metodo de pago (opcional)
	MetodoPago *string `json:"metodo_pago" example:"transferencia"`
	// Proveedor o beneficiario (opcional)
	Proveedor *string `json:"proveedor" example:"Inmobiliaria Sur"`
	// Vigente desde YYYY-MM-DD
	FechaInicio string `json:"fecha_inicio" binding:"required" example:"2026-01-01"`
	// Vigente hasta YYYY-MM-DD (opcional)
	FechaFin *string `json:"fecha_fin" example:"2026-12-31"`
	// Activo (default true)
	Activo *bool `json:"activo" example:"true"`
	// Notas (opcional)
	Notas *string `json:"notas" example:"Contrato anual"`
}

type actualizarGastoFijoRequest struct {
	// Nombre
	Nombre *string `json:"nombre" example:"Alquiler"`
	// Categoria
	CategoriaID *int `json:"categoria_id" example:"1"`
	// Monto mensual
	Monto *float64 `json:"monto" example:"7000"`
	// Dia de cobro
	DiaCobro *int `json:"dia_cobro" example:"10"`
	// Local (null = general)
	LocalID *int `json:"local_id" example:"1"`
	// Metodo de pago (null limpia)
	MetodoPago *string `json:"metodo_pago" example:"qr"`
	// Proveedor (null limpia)
	Proveedor *string `json:"proveedor" example:"Inmobiliaria Sur"`
	// Vigente desde
	FechaInicio *string `json:"fecha_inicio" example:"2026-01-01"`
	// Vigente hasta (null quita el fin)
	FechaFin *string `json:"fecha_fin" example:"2026-12-31"`
	// Activo
	Activo *bool `json:"activo" example:"true"`
	// Notas (null limpia)
	Notas *string `json:"notas" example:"Contrato anual"`
}

// ─── Categorias financieras ───────────────────────────────────────────────────

// GetCategoriasFinancieras godoc
// @Summary Listar categorias financieras
// @Description Lista las categorias de ingresos y egresos. Requiere token Bearer con rol admin_sys.
// @Tags Finanzas
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Success 200 {object} utils.APIResponse{data=categoriasFinancierasResponse}
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/finanzas/categorias [get]
func (h *Container) GetCategoriasFinancieras(c *gin.Context) {
	cats, err := h.Finanzas.ListarCategorias()
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, categoriasFinancierasResponse{Categorias: cats})
}

// PostCategoriaFinanciera godoc
// @Summary Crear categoria financiera
// @Description Crea una categoria de INGRESO o EGRESO. El nombre es unico por naturaleza. Requiere token Bearer con rol admin_sys.
// @Tags Finanzas
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body crearCategoriaFinancieraRequest true "Datos de la categoria"
// @Success 201 {object} utils.APIResponse{data=idResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: nombre o naturaleza invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 409 {object} utils.APIResponse "Ya existe una categoria con ese nombre"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/finanzas/categorias [post]
func (h *Container) PostCategoriaFinanciera(c *gin.Context) {
	var req crearCategoriaFinancieraRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	id, err := h.Finanzas.CrearCategoria(req.Nombre, req.Naturaleza)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusCreated, idResponse{ID: id})
}

// PatchCategoriaFinanciera godoc
// @Summary Editar categoria financiera
// @Description Cambia el nombre o activa/desactiva la categoria (la naturaleza no cambia). Requiere token Bearer con rol admin_sys.
// @Tags Finanzas
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID de la categoria" example(1)
// @Param payload body actualizarCategoriaFinancieraRequest true "Campos a actualizar"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido, sin campos, nombre invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Categoria no encontrada"
// @Failure 409 {object} utils.APIResponse "Ya existe una categoria con ese nombre"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/finanzas/categorias/{id} [patch]
func (h *Container) PatchCategoriaFinanciera(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var req actualizarCategoriaFinancieraRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Finanzas.ActualizarCategoria(id, req.Nombre, req.Activo); err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "categoria actualizada correctamente"})
}

// ─── Movimientos ──────────────────────────────────────────────────────────────

// GetMovimientosFinancieros godoc
// @Summary Listar movimientos financieros
// @Description Lista egresos (fijos y variables, incluidas las compras de inventario) y otros ingresos. Las ventas no aparecen aqui: siguen en /bd/pagos. Filtros: naturaleza, clase, categoria_id, estado, local_id, fecha_desde, fecha_hasta. Requiere token Bearer con rol admin_sys.
// @Tags Finanzas
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param naturaleza query string false "INGRESO o EGRESO"
// @Param clase query string false "FIJO o VARIABLE"
// @Param categoria_id query int false "Filtrar por categoria"
// @Param estado query string false "PENDIENTE, PAGADO o ANULADO"
// @Param local_id query int false "Filtrar por local"
// @Param fecha_desde query string false "Desde (YYYY-MM-DD)"
// @Param fecha_hasta query string false "Hasta (YYYY-MM-DD)"
// @Success 200 {object} utils.APIResponse{data=movimientosFinancierosResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: parametro invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/finanzas/movimientos [get]
func (h *Container) GetMovimientosFinancieros(c *gin.Context) {
	categoriaID, err := queryInt(c, "categoria_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	localID, err := queryInt(c, "local_id")
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
	movs, err := h.Finanzas.ListarMovimientos(repository.FiltroMovimientosFinancieros{
		Naturaleza: strings.ToUpper(strings.TrimSpace(c.Query("naturaleza"))), Clase: strings.ToUpper(strings.TrimSpace(c.Query("clase"))),
		CategoriaID: categoriaID, Estado: strings.ToUpper(strings.TrimSpace(c.Query("estado"))), LocalID: localID,
		FechaDesde: desde, FechaHasta: hasta,
	})
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, movimientosFinancierosResponse{Movimientos: movs})
}

// PostMovimientoFinanciero godoc
// @Summary Registrar egreso variable u otro ingreso
// @Description Registra un egreso VARIABLE o un otro ingreso. Los egresos FIJO nacen solo de una plantilla de gasto fijo y las compras de inventario de POST /bd/compras. La categoria debe coincidir con la naturaleza. Requiere token Bearer con rol admin_sys.
// @Tags Finanzas
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body crearMovimientoFinancieroRequest true "Datos del movimiento"
// @Success 201 {object} utils.APIResponse{data=movimientoFinancieroResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: naturaleza, clase, monto, fecha, estado, metodo_pago o categoria invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Categoria no encontrada"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/finanzas/movimientos [post]
func (h *Container) PostMovimientoFinanciero(c *gin.Context) {
	var req crearMovimientoFinancieroRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	actor := consumoActor(c)
	mov, err := h.Finanzas.RegistrarMovimiento(services.MovimientoFinancieroInput{
		Naturaleza: req.Naturaleza, Clase: req.Clase, CategoriaID: req.CategoriaID, LocalID: req.LocalID, Fecha: req.Fecha,
		Monto: req.Monto, Descripcion: req.Descripcion, Proveedor: req.Proveedor, MetodoPago: req.MetodoPago,
		Estado: req.Estado, UsuarioID: actor.UsuarioID, UsuarioNombre: actor.UsuarioNombre,
	})
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusCreated, movimientoFinancieroResponse{Movimiento: *mov})
}

// PatchMovimientoFinanciero godoc
// @Summary Editar movimiento financiero
// @Description Edita un movimiento no anulado. Editar un egreso FIJO ya generado afecta solo ese mes; para cambiar los meses siguientes se edita la plantilla. Los egresos de compras de inventario no se editan (se anula la compra). descripcion, proveedor, metodo_pago y local_id aceptan null para limpiar. Requiere token Bearer con rol admin_sys.
// @Tags Finanzas
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del movimiento" example(1)
// @Param payload body actualizarMovimientoFinancieroRequest true "Campos a actualizar"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id, monto, fecha, estado o categoria invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Movimiento no encontrado"
// @Failure 409 {object} utils.APIResponse "El movimiento esta anulado o pertenece a una compra"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/finanzas/movimientos/{id} [patch]
func (h *Container) PatchMovimientoFinanciero(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var req actualizarMovimientoFinancieroRequest
	present, ok := decodePatch(c, &req)
	if !ok {
		return
	}
	err = h.Finanzas.EditarMovimiento(id, services.MovimientoPatch{
		CategoriaID: req.CategoriaID, LocalID: req.LocalID, LocalSet: present["local_id"], Fecha: req.Fecha, Monto: req.Monto,
		Descripcion: req.Descripcion, DescripcionSet: present["descripcion"], Proveedor: req.Proveedor,
		ProveedorSet: present["proveedor"], MetodoPago: req.MetodoPago, MetodoSet: present["metodo_pago"], Estado: req.Estado,
	})
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "movimiento actualizado correctamente"})
}

// PatchMovimientoFinancieroEstado godoc
// @Summary Marcar pagado o anular movimiento
// @Description Cambia el estado a PAGADO o ANULADO. Anular exige motivo; los movimientos no se borran. Un egreso de compra de inventario se anula desde la compra. Requiere token Bearer con rol admin_sys.
// @Tags Finanzas
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del movimiento" example(1)
// @Param payload body cambiarEstadoMovimientoRequest true "Nuevo estado"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: estado invalido, motivo requerido al anular"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Movimiento no encontrado o ya anulado"
// @Failure 409 {object} utils.APIResponse "El egreso pertenece a una compra de inventario"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/finanzas/movimientos/{id}/estado [patch]
func (h *Container) PatchMovimientoFinancieroEstado(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var req cambiarEstadoMovimientoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Finanzas.CambiarEstado(id, req.Estado, req.Motivo); err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "estado actualizado correctamente"})
}

// ─── Gastos fijos ─────────────────────────────────────────────────────────────

// GetGastosFijos godoc
// @Summary Listar gastos fijos
// @Description Lista las plantillas de gasto fijo (activas e inactivas) con su proximo cobro. Requiere token Bearer con rol admin_sys.
// @Tags Finanzas
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Success 200 {object} utils.APIResponse{data=gastosFijosResponse}
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/finanzas/gastos-fijos [get]
func (h *Container) GetGastosFijos(c *gin.Context) {
	gastos, err := h.GastosFijos.Listar()
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, gastosFijosResponse{GastosFijos: gastos})
}

// PostGastoFijo godoc
// @Summary Crear gasto fijo
// @Description Crea una plantilla de gasto recurrente: el dia dia_cobro de cada mes se genera un egreso FIJO. Si el mes no tiene ese dia se usa el ultimo del mes. Los meses atrasados desde fecha_inicio se generan en la siguiente ejecucion (catch-up). Requiere token Bearer con rol admin_sys.
// @Tags Finanzas
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body gastoFijoRequest true "Datos del gasto fijo"
// @Success 201 {object} utils.APIResponse{data=gastoFijoResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: nombre, monto, dia_cobro, fechas, metodo_pago o categoria invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Categoria no encontrada"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/finanzas/gastos-fijos [post]
func (h *Container) PostGastoFijo(c *gin.Context) {
	var req gastoFijoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	gasto, err := h.GastosFijos.Crear(services.GastoFijoInput{
		Nombre: req.Nombre, CategoriaID: req.CategoriaID, Monto: req.Monto, DiaCobro: req.DiaCobro, LocalID: req.LocalID,
		MetodoPago: req.MetodoPago, Proveedor: req.Proveedor, FechaInicio: req.FechaInicio, FechaFin: req.FechaFin,
		Activo: req.Activo, Notas: req.Notas,
	})
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusCreated, gastoFijoResponse{GastoFijo: *gasto})
}

// PatchGastoFijo godoc
// @Summary Editar gasto fijo
// @Description Edita la plantilla. Afecta solo periodos futuros aun no generados: los egresos ya generados no cambian (se editan uno a uno). fecha_fin permite dar de baja sin perder historia. Requiere token Bearer con rol admin_sys.
// @Tags Finanzas
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del gasto fijo" example(1)
// @Param payload body actualizarGastoFijoRequest true "Campos a actualizar"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id, monto, dia_cobro, fechas o categoria invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Gasto fijo no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/finanzas/gastos-fijos/{id} [patch]
func (h *Container) PatchGastoFijo(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var req actualizarGastoFijoRequest
	present, ok := decodePatch(c, &req)
	if !ok {
		return
	}
	err = h.GastosFijos.Actualizar(id, services.GastoFijoPatch{
		Nombre: req.Nombre, CategoriaID: req.CategoriaID, Monto: req.Monto, DiaCobro: req.DiaCobro,
		LocalID: req.LocalID, LocalSet: present["local_id"], MetodoPago: req.MetodoPago, MetodoSet: present["metodo_pago"],
		Proveedor: req.Proveedor, ProveedorSet: present["proveedor"], FechaInicio: req.FechaInicio,
		FechaFin: req.FechaFin, FechaFinSet: present["fecha_fin"], Activo: req.Activo, Notas: req.Notas, NotasSet: present["notas"],
	})
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "gasto fijo actualizado correctamente"})
}

// DeleteGastoFijo godoc
// @Summary Desactivar gasto fijo
// @Description Desactiva la plantilla (activo=false). Deja de generarse; el historial ya generado se conserva. Requiere token Bearer con rol admin_sys.
// @Tags Finanzas
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del gasto fijo" example(1)
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Gasto fijo no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/finanzas/gastos-fijos/{id} [delete]
func (h *Container) DeleteGastoFijo(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.GastosFijos.Desactivar(id); err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "gasto fijo desactivado correctamente"})
}

// GetGastosFijosProximos godoc
// @Summary Gastos fijos por vencer
// @Description Lista los cobros que vencen dentro de finanzas.dias_aviso_gasto_fijo (para el dashboard). Antes de calcular genera los gastos fijos atrasados, asi el aviso es correcto aunque el job diario haya fallado. Requiere token Bearer con rol admin_sys.
// @Tags Finanzas
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Success 200 {object} utils.APIResponse{data=gastosFijosProximosResponse}
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/finanzas/gastos-fijos/proximos [get]
func (h *Container) GetGastosFijosProximos(c *gin.Context) {
	proximos, err := h.GastosFijos.Proximos()
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, gastosFijosProximosResponse{Proximos: proximos})
}

// PostGenerarGastosFijos godoc
// @Summary Generar gastos fijos pendientes
// @Description Genera, de forma idempotente, los egresos FIJO que ya vencieron hasta hoy (incluye meses atrasados). Lo usan el job diario y el boton manual: ejecutarlo dos veces no duplica nada. Acepta token Bearer con rol admin_sys o el header X-Job-Secret configurado en el servidor.
// @Tags Finanzas
// @Produce json
// @Param Authorization header string false "Token Bearer (admin_sys) si no se usa X-Job-Secret" default(Bearer <token>)
// @Param X-Job-Secret header string false "Secreto interno del job diario"
// @Success 200 {object} utils.APIResponse{data=generarGastosFijosResponse}
// @Failure 401 {object} utils.APIResponse "Token o secreto requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/finanzas/gastos-fijos/generar [post]
func (h *Container) PostGenerarGastosFijos(c *gin.Context) {
	n, err := h.GastosFijos.GenerarHoy()
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	msg := "gastos fijos al dia"
	if n > 0 {
		msg = "gastos fijos generados correctamente"
	}
	utils.Respond(c, http.StatusOK, generarGastosFijosResponse{Generados: n, Mensaje: msg})
}
