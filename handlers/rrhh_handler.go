package handlers

import (
	"net/http"
	"strings"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

// ─── Request / response ───────────────────────────────────────────────────────

type empleadosResponse struct {
	// Empleados con su dia de pago efectivo y adelantos pendientes
	Empleados []models.Empleado `json:"empleados"`
}

type empleadoResponse struct {
	// Empleado
	Empleado models.Empleado `json:"empleado"`
}

type empleadoRequest struct {
	// Nombre
	Nombre string `json:"nombre" binding:"required" example:"Maria"`
	// Apellido
	Apellido string `json:"apellido" binding:"required" example:"Rojas"`
	// Cedula de identidad (opcional)
	CI *string `json:"ci" example:"7654321"`
	// Cargo (opcional)
	Cargo *string `json:"cargo" example:"Masajista"`
	// Telefono (opcional)
	Telefono *string `json:"telefono" example:"71234567"`
	// Local donde trabaja; null = personal general
	LocalID *int `json:"local_id" example:"1"`
	// Usuario del sistema asociado (opcional)
	UsuarioID *int `json:"usuario_id" example:"3"`
	// Sueldo base mensual
	SueldoBase float64 `json:"sueldo_base" binding:"required" example:"3000"`
	// Dia de pago 1-31; null usa rrhh.dia_pago_defecto
	DiaPago *int `json:"dia_pago" example:"5"`
	// Si se calculan aportes de ley (default true)
	AplicaAportes *bool `json:"aplica_aportes" example:"true"`
	// Metodo de pago habitual (opcional)
	MetodoPago *string `json:"metodo_pago" example:"transferencia"`
	// Fecha de ingreso YYYY-MM-DD
	FechaIngreso string `json:"fecha_ingreso" binding:"required" example:"2026-01-01"`
	// Fecha de baja YYYY-MM-DD (opcional)
	FechaBaja *string `json:"fecha_baja" example:"2026-12-31"`
	// Activo (default true)
	Activo *bool `json:"activo" example:"true"`
	// Notas (opcional)
	Notas *string `json:"notas" example:"Turno tarde"`
}

type actualizarEmpleadoRequest struct {
	Nombre        *string  `json:"nombre" example:"Maria"`
	Apellido      *string  `json:"apellido" example:"Rojas"`
	CI            *string  `json:"ci" example:"7654321"`
	Cargo         *string  `json:"cargo" example:"Masajista"`
	Telefono      *string  `json:"telefono" example:"71234567"`
	LocalID       *int     `json:"local_id" example:"1"`
	UsuarioID     *int     `json:"usuario_id" example:"3"`
	SueldoBase    *float64 `json:"sueldo_base" example:"3200"`
	DiaPago       *int     `json:"dia_pago" example:"5"`
	AplicaAportes *bool    `json:"aplica_aportes" example:"true"`
	MetodoPago    *string  `json:"metodo_pago" example:"qr"`
	FechaIngreso  *string  `json:"fecha_ingreso" example:"2026-01-01"`
	FechaBaja     *string  `json:"fecha_baja" example:"2026-12-31"`
	Activo        *bool    `json:"activo" example:"true"`
	Notas         *string  `json:"notas" example:"Turno tarde"`
}

type adelantosResponse struct {
	// Adelantos, mas recientes primero
	Adelantos []models.Adelanto `json:"adelantos"`
}

type adelantoRequest struct {
	// Empleado
	EmpleadoID int `json:"empleado_id" binding:"required" example:"1"`
	// Fecha YYYY-MM-DD
	Fecha string `json:"fecha" binding:"required" example:"2026-10-10"`
	// Monto del adelanto
	Monto float64 `json:"monto" binding:"required" example:"200"`
	// Motivo (opcional)
	Motivo *string `json:"motivo" example:"Emergencia familiar"`
}

type planillasResponse struct {
	// Planillas del filtro
	Planillas []models.Planilla `json:"planillas"`
	// Totales de las planillas no anuladas del filtro
	Resumen models.PlanillaResumen `json:"resumen"`
}

type planillaResponse struct {
	// Planilla
	Planilla models.Planilla `json:"planilla"`
}

type planillasPendientesResponse struct {
	// Planillas vencidas que aun no se generaron
	Pendientes []models.PlanillaPendiente `json:"pendientes"`
}

type generarPlanillasResponse struct {
	// Cantidad de planillas generadas en esta ejecucion
	Generados int `json:"generados" example:"2"`
	// Mensaje descriptivo
	Mensaje string `json:"mensaje" example:"planillas al dia"`
}

type crearPlanillaRequest struct {
	// Empleado
	EmpleadoID int `json:"empleado_id" binding:"required" example:"1"`
	// Periodo YYYY-MM
	Periodo string `json:"periodo" binding:"required" example:"2026-10"`
}

type actualizarPlanillaRequest struct {
	// Bonos del mes
	Bonos *float64 `json:"bonos" example:"150"`
	// Dias de falta (admite fracciones, ej. 0.5)
	DiasFalta *float64 `json:"dias_falta" example:"1"`
	// Otros descuentos
	OtrosDescuentos *float64 `json:"otros_descuentos" example:"50"`
	// Notas (null limpia)
	Notas *string `json:"notas" example:"Bono por meta"`
}

type pagarPlanillaRequest struct {
	// Metodo de pago (opcional; si no se envia queda el del empleado)
	MetodoPago *string `json:"metodo_pago" example:"transferencia"`
}

type anularPlanillaRequest struct {
	// Motivo de la anulacion
	Motivo string `json:"motivo" binding:"required" example:"Sueldo mal calculado"`
}

type generarAguinaldosRequest struct {
	// Anio del aguinaldo
	Anio int `json:"anio" binding:"required" example:"2026"`
}

// ─── Empleados ────────────────────────────────────────────────────────────────

// GetEmpleados godoc
// @Summary Listar empleados
// @Description Lista los empleados con su dia de pago efectivo (el propio o rrhh.dia_pago_defecto) y los adelantos pendientes de descontar. Requiere token Bearer con rol admin_sys.
// @Tags RRHH
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param activos query bool false "true devuelve solo empleados activos"
// @Success 200 {object} utils.APIResponse{data=empleadosResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: activos invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/rrhh/empleados [get]
func (h *Container) GetEmpleados(c *gin.Context) {
	activos, err := queryBool(c, "activos")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	emps, err := h.RRHH.ListarEmpleados(activos != nil && *activos)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, empleadosResponse{Empleados: emps})
}

// GetEmpleadoByID godoc
// @Summary Ver empleado
// @Description Devuelve la ficha de un empleado. Requiere token Bearer con rol admin_sys.
// @Tags RRHH
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del empleado" example(1)
// @Success 200 {object} utils.APIResponse{data=empleadoResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Empleado no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/rrhh/empleados/{id} [get]
func (h *Container) GetEmpleadoByID(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	emp, err := h.RRHH.ObtenerEmpleado(id)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, empleadoResponse{Empleado: *emp})
}

// PostEmpleado godoc
// @Summary Crear empleado
// @Description Da de alta un empleado. Desde su dia de pago de cada mes se genera su planilla de sueldo y el egreso en finanzas. Si no se envia dia_pago se usa rrhh.dia_pago_defecto. Requiere token Bearer con rol admin_sys.
// @Tags RRHH
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body empleadoRequest true "Datos del empleado"
// @Success 201 {object} utils.APIResponse{data=empleadoResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: nombre, sueldo_base, dia_pago, fechas o metodo_pago invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/rrhh/empleados [post]
func (h *Container) PostEmpleado(c *gin.Context) {
	var req empleadoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	emp, err := h.RRHH.CrearEmpleado(services.EmpleadoInput{
		Nombre: req.Nombre, Apellido: req.Apellido, CI: req.CI, Cargo: req.Cargo, Telefono: req.Telefono,
		LocalID: req.LocalID, UsuarioID: req.UsuarioID, SueldoBase: req.SueldoBase, DiaPago: req.DiaPago,
		AplicaAportes: req.AplicaAportes, MetodoPago: req.MetodoPago, FechaIngreso: req.FechaIngreso,
		FechaBaja: req.FechaBaja, Activo: req.Activo, Notas: req.Notas,
	})
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusCreated, empleadoResponse{Empleado: *emp})
}

// PatchEmpleado godoc
// @Summary Editar empleado
// @Description Edita la ficha. Afecta solo planillas futuras aun no generadas. Para dar de baja envia fecha_baja o activo=false; con null en dia_pago vuelve al dia por defecto. Requiere token Bearer con rol admin_sys.
// @Tags RRHH
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del empleado" example(1)
// @Param payload body actualizarEmpleadoRequest true "Campos a actualizar"
// @Success 200 {object} utils.APIResponse{data=empleadoResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id, sueldo_base, dia_pago o fechas invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Empleado no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/rrhh/empleados/{id} [patch]
func (h *Container) PatchEmpleado(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var req actualizarEmpleadoRequest
	present, ok := decodePatch(c, &req)
	if !ok {
		return
	}
	emp, err := h.RRHH.ActualizarEmpleado(id, services.EmpleadoPatch{
		Nombre: req.Nombre, Apellido: req.Apellido, CI: req.CI, CISet: present["ci"], Cargo: req.Cargo,
		CargoSet: present["cargo"], Telefono: req.Telefono, TelefonoSet: present["telefono"], LocalID: req.LocalID,
		LocalSet: present["local_id"], UsuarioID: req.UsuarioID, UsuarioSet: present["usuario_id"],
		SueldoBase: req.SueldoBase, DiaPago: req.DiaPago, DiaPagoSet: present["dia_pago"],
		AplicaAportes: req.AplicaAportes, MetodoPago: req.MetodoPago, MetodoSet: present["metodo_pago"],
		FechaIngreso: req.FechaIngreso, FechaBaja: req.FechaBaja, FechaBajaSet: present["fecha_baja"],
		Activo: req.Activo, Notas: req.Notas, NotasSet: present["notas"],
	})
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, empleadoResponse{Empleado: *emp})
}

// ─── Adelantos ────────────────────────────────────────────────────────────────

// GetAdelantos godoc
// @Summary Listar adelantos
// @Description Lista los adelantos de sueldo. Un adelanto se descuenta solo en la siguiente planilla que se genere del empleado. Requiere token Bearer con rol admin_sys.
// @Tags RRHH
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param empleado_id query int false "Filtra por empleado"
// @Param pendientes query bool false "true devuelve solo los aun no descontados"
// @Success 200 {object} utils.APIResponse{data=adelantosResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: filtros invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/rrhh/adelantos [get]
func (h *Container) GetAdelantos(c *gin.Context) {
	empleadoID, err := queryInt(c, "empleado_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	pendientes, err := queryBool(c, "pendientes")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	lista, err := h.RRHH.ListarAdelantos(empleadoID, pendientes != nil && *pendientes)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, adelantosResponse{Adelantos: lista})
}

// PostAdelanto godoc
// @Summary Registrar adelanto
// @Description Registra un anticipo de sueldo; se descuenta automaticamente en la siguiente planilla del empleado (siempre que quepa en lo que queda tras aportes y descuentos). Requiere token Bearer con rol admin_sys.
// @Tags RRHH
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body adelantoRequest true "Datos del adelanto"
// @Success 201 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: monto, fecha o empleado invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Empleado no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/rrhh/adelantos [post]
func (h *Container) PostAdelanto(c *gin.Context) {
	var req adelantoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.RRHH.RegistrarAdelanto(req.EmpleadoID, req.Fecha, req.Monto, req.Motivo); err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusCreated, messageResponse{Mensaje: "adelanto registrado correctamente"})
}

// DeleteAdelanto godoc
// @Summary Anular adelanto
// @Description Anula un adelanto que aun no se descontó en ninguna planilla. Requiere token Bearer con rol admin_sys.
// @Tags RRHH
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del adelanto" example(1)
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Adelanto no encontrado"
// @Failure 409 {object} utils.APIResponse "El adelanto ya fue descontado o anulado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/rrhh/adelantos/{id} [delete]
func (h *Container) DeleteAdelanto(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.RRHH.AnularAdelanto(id); err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "adelanto anulado correctamente"})
}

// ─── Planillas ────────────────────────────────────────────────────────────────

// GetPlanillas godoc
// @Summary Listar planillas
// @Description Lista las planillas de sueldo con su desglose y los totales del filtro. Antes de listar genera las planillas vencidas (catch-up), asi la lista es correcta aunque el job diario no haya corrido. Requiere token Bearer con rol admin_sys.
// @Tags RRHH
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param periodo query string false "Mes YYYY-MM"
// @Param empleado_id query int false "Filtra por empleado"
// @Param local_id query int false "Filtra por local del empleado"
// @Param estado query string false "PENDIENTE, PAGADO o ANULADO"
// @Param tipo query string false "SUELDO o AGUINALDO"
// @Success 200 {object} utils.APIResponse{data=planillasResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: periodo, estado o tipo invalidos"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/rrhh/planillas [get]
func (h *Container) GetPlanillas(c *gin.Context) {
	empleadoID, err := queryInt(c, "empleado_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	localID, err := queryInt(c, "local_id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := h.RRHH.GenerarHoy(); err != nil {
		respondTrastiendaError(c, err)
		return
	}
	periodo := strings.TrimSpace(c.Query("periodo"))
	lista, err := h.RRHH.ListarPlanillas(services.FiltroPlanillas{
		Periodo: periodo, EmpleadoID: empleadoID, LocalID: localID, Estado: c.Query("estado"), Tipo: c.Query("tipo"),
	})
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, planillasResponse{Planillas: lista, Resumen: h.RRHH.ResumenPeriodo(lista, periodo)})
}

// GetPlanillaByID godoc
// @Summary Ver planilla
// @Description Devuelve una planilla con su desglose y el codigo del egreso asociado. Requiere token Bearer con rol admin_sys.
// @Tags RRHH
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID de la planilla" example(1)
// @Success 200 {object} utils.APIResponse{data=planillaResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Planilla no encontrada"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/rrhh/planillas/{id} [get]
func (h *Container) GetPlanillaByID(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.RRHH.ObtenerPlanilla(id)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, planillaResponse{Planilla: *p})
}

// PostPlanilla godoc
// @Summary Generar planilla de un empleado
// @Description Crea ya la planilla de sueldo de un empleado para un mes (por ejemplo para cargar bonos o faltas antes del dia de pago, o regenerar una anulada). Crea tambien el egreso PENDIENTE en finanzas y descuenta los adelantos pendientes. Requiere token Bearer con rol admin_sys.
// @Tags RRHH
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body crearPlanillaRequest true "Empleado y periodo"
// @Success 201 {object} utils.APIResponse{data=planillaResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: periodo invalido, futuro o anterior al ingreso"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Empleado no encontrado"
// @Failure 409 {object} utils.APIResponse "Ya existe una planilla vigente para ese periodo"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/rrhh/planillas [post]
func (h *Container) PostPlanilla(c *gin.Context) {
	var req crearPlanillaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.RRHH.GenerarPlanillaManual(req.EmpleadoID, req.Periodo)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusCreated, planillaResponse{Planilla: *p})
}

// PatchPlanilla godoc
// @Summary Editar planilla
// @Description Cambia bonos, dias de falta, otros descuentos o notas de una planilla PENDIENTE. Recalcula aportes y neto, y actualiza el monto del egreso. La falta se valora como sueldo_base / rrhh.dias_mes_faltas por dia. Requiere token Bearer con rol admin_sys.
// @Tags RRHH
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID de la planilla" example(1)
// @Param payload body actualizarPlanillaRequest true "Campos a actualizar"
// @Success 200 {object} utils.APIResponse{data=planillaResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: montos negativos, descuentos mayores a lo ganado o planilla no editable"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Planilla no encontrada"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/rrhh/planillas/{id} [patch]
func (h *Container) PatchPlanilla(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var req actualizarPlanillaRequest
	present, ok := decodePatch(c, &req)
	if !ok {
		return
	}
	p, err := h.RRHH.ActualizarPlanilla(id, services.PlanillaPatch{
		Bonos: req.Bonos, DiasFalta: req.DiasFalta, OtrosDescuentos: req.OtrosDescuentos,
		Notas: req.Notas, NotasSet: present["notas"],
	})
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, planillaResponse{Planilla: *p})
}

// PatchPlanillaPagar godoc
// @Summary Marcar planilla como pagada
// @Description Pasa la planilla PENDIENTE y su egreso a PAGADO; desde ese momento cuenta en el reporte financiero. Requiere token Bearer con rol admin_sys.
// @Tags RRHH
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID de la planilla" example(1)
// @Param payload body pagarPlanillaRequest false "Metodo de pago"
// @Success 200 {object} utils.APIResponse{data=planillaResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: planilla no pendiente o metodo_pago invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Planilla no encontrada"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/rrhh/planillas/{id}/pagar [patch]
func (h *Container) PatchPlanillaPagar(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var req pagarPlanillaRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.RespondError(c, http.StatusBadRequest, err.Error())
			return
		}
	}
	p, err := h.RRHH.Pagar(id, req.MetodoPago)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, planillaResponse{Planilla: *p})
}

// PatchPlanillaAnular godoc
// @Summary Anular planilla
// @Description Anula la planilla y su egreso (exige motivo) y libera los adelantos que descontaba para que vuelvan a pendientes. Se puede regenerar con POST /bd/rrhh/planillas. Requiere token Bearer con rol admin_sys.
// @Tags RRHH
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID de la planilla" example(1)
// @Param payload body anularPlanillaRequest true "Motivo"
// @Success 200 {object} utils.APIResponse{data=planillaResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: motivo requerido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 404 {object} utils.APIResponse "Planilla no encontrada"
// @Failure 409 {object} utils.APIResponse "La planilla ya esta anulada"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/rrhh/planillas/{id}/anular [patch]
func (h *Container) PatchPlanillaAnular(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var req anularPlanillaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.RRHH.Anular(id, req.Motivo)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, planillaResponse{Planilla: *p})
}

// GetPlanillasPendientes godoc
// @Summary Planillas pendientes de generar
// @Description Vista previa, sin crear nada, de las planillas que POST /bd/rrhh/planillas/generar generaria ahora (incluye meses atrasados). Requiere token Bearer con rol admin_sys.
// @Tags RRHH
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Success 200 {object} utils.APIResponse{data=planillasPendientesResponse}
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/rrhh/planillas/pendientes [get]
func (h *Container) GetPlanillasPendientes(c *gin.Context) {
	pend, err := h.RRHH.PendientesDeGenerar()
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, planillasPendientesResponse{Pendientes: pend})
}

// PostGenerarPlanillas godoc
// @Summary Generar planillas vencidas
// @Description Genera, de forma idempotente, las planillas de sueldo cuyo dia de pago ya llego (incluye meses atrasados) y su egreso PENDIENTE en finanzas. Lo usan el job diario y el boton manual: ejecutarlo dos veces no duplica nada. Acepta token Bearer con rol admin_sys o el header X-Job-Secret configurado en el servidor.
// @Tags RRHH
// @Produce json
// @Param Authorization header string false "Token Bearer (admin_sys) si no se usa X-Job-Secret" default(Bearer <token>)
// @Param X-Job-Secret header string false "Secreto interno del job diario"
// @Success 200 {object} utils.APIResponse{data=generarPlanillasResponse}
// @Failure 401 {object} utils.APIResponse "Token o secreto requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/rrhh/planillas/generar [post]
func (h *Container) PostGenerarPlanillas(c *gin.Context) {
	n, err := h.RRHH.GenerarHoy()
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	msg := "planillas al dia"
	if n > 0 {
		msg = "planillas generadas correctamente"
	}
	utils.Respond(c, http.StatusOK, generarPlanillasResponse{Generados: n, Mensaje: msg})
}

// PostGenerarAguinaldos godoc
// @Summary Generar aguinaldos del anio
// @Description Crea el aguinaldo (planilla tipo AGUINALDO, vence el 20 de diciembre) de los empleados con al menos 90 dias trabajados en el anio: un sueldo base proporcional a los dias trabajados, sin aportes. Idempotente. Requiere token Bearer con rol admin_sys.
// @Tags RRHH
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param payload body generarAguinaldosRequest true "Anio"
// @Success 200 {object} utils.APIResponse{data=generarPlanillasResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: anio invalido o futuro"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Usuario no autorizado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/rrhh/aguinaldos/generar [post]
func (h *Container) PostGenerarAguinaldos(c *gin.Context) {
	var req generarAguinaldosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	n, err := h.RRHH.GenerarAguinaldos(req.Anio)
	if err != nil {
		respondTrastiendaError(c, err)
		return
	}
	msg := "aguinaldos al dia"
	if n > 0 {
		msg = "aguinaldos generados correctamente"
	}
	utils.Respond(c, http.StatusOK, generarPlanillasResponse{Generados: n, Mensaje: msg})
}
