package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"atrevida-agenda-api/cuenta"
	"atrevida-agenda-api/models"
	"atrevida-agenda-api/pagination"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
	"atrevida-agenda-api/services"
	"atrevida-agenda-api/utils"

	"github.com/gin-gonic/gin"
)

type crearCargoRequest struct {
	// Tipo del cargo: PAQUETE, SERVICIO, INSUMO u OTRO.
	Tipo string `json:"tipo" example:"PAQUETE"`
	// Concepto legible del cargo (hasta 300 caracteres).
	Concepto string `json:"concepto" example:"Paquete Relax 4 sesiones"`
	// Monto del cargo; 0 registra un consumo que no se cobra.
	Monto *float64 `json:"monto" example:"1000"`
	// Fecha del hecho (YYYY-MM-DD): cuando ocurrio lo que se cobra. Opcional; sin ella es ahora. No puede ser futura.
	FechaHecho string `json:"fecha_hecho,omitempty" example:"2026-10-25"`
	// Local donde ocurrio el hecho (opcional); un usuario limitado a un local solo puede usar el suyo.
	LocalID *int `json:"local_id,omitempty" example:"1"`
	// Reserva de la que nace el cargo (opcional).
	ReservaID *int `json:"reserva_id,omitempty" example:"44"`
	// Plan del que nace el cargo (opcional).
	PlanID *int `json:"plan_id,omitempty" example:"21"`
}

type anularCargoRequest struct {
	// Motivo de la anulacion (requerido).
	Motivo string `json:"motivo" example:"Cargado dos veces"`
}

type crearAbonoRequest struct {
	// Local donde se recibe el pago.
	LocalID *int `json:"local_id" example:"1"`
	// Nombre del local donde se recibe el pago.
	LocalNombre string `json:"local_nombre" example:"SAN MARTIN"`
	// Monto entregado; debe ser mayor a 0.
	Monto float64 `json:"monto" example:"500"`
	// Metodo de pago: efectivo o qr.
	TipoPago string `json:"tipo_pago" example:"efectivo"`
	// Fecha del hecho (YYYY-MM-DD): cuando se entrego el dinero. Opcional; sin ella es ahora. No puede ser futura.
	FechaHecho string `json:"fecha_hecho,omitempty" example:"2026-10-25"`
	// Reparto del abono entre cargos pendientes. Lo que no se aplica queda como saldo disponible.
	Aplicaciones []cuenta.Reparto `json:"aplicaciones"`
	// Nombre del cajero; si se omite se usa el username del token.
	Nombre *string `json:"nombre,omitempty" example:"Juan Perez"`
}

type aplicarAbonoRequest struct {
	// Reparto de lo que el abono tiene sin aplicar entre cargos pendientes.
	Aplicaciones []cuenta.Reparto `json:"aplicaciones"`
}

type cuentaClienteResponse struct {
	// Cuenta del cliente: saldo, cargos y abonos.
	Cuenta *models.CuentaCliente `json:"cuenta"`
}

type cargoCreadoResponse struct {
	// ID del cargo creado.
	ID int `json:"id" example:"12"`
}

type abonoCreadoResponse struct {
	// Codigo del pago creado.
	CodigoPago string `json:"codigo_pago" example:"PAGO-000123"`
}

// statusCuentaError mapea errores de la cuenta al codigo HTTP: lo no encontrado es 404, las reglas y validaciones
// son 400 y todo lo demas (infraestructura) es 500.
func statusCuentaError(err error) int {
	msg := strings.ToLower(err.Error())
	var regla pgsqlrepo.ErrReglaCuenta
	switch {
	case strings.Contains(msg, "no encontrad"):
		return http.StatusNotFound
	case errors.As(err, &regla):
		return http.StatusBadRequest
	}
	for _, palabra := range []string{"requerido", "invalido", "formato", "no puede", "debe ser", "debe enviar", "nombre del cajero"} {
		if strings.Contains(msg, palabra) {
			return http.StatusBadRequest
		}
	}
	return http.StatusInternalServerError
}

func respondCuentaError(c *gin.Context, err error) {
	utils.RespondError(c, statusCuentaError(err), err.Error())
}

func pathInt(c *gin.Context, name string) (int, bool) {
	v, err := strconv.Atoi(c.Param(name))
	if err != nil || v < 1 {
		utils.RespondError(c, http.StatusBadRequest, name+" debe ser un entero positivo")
		return 0, false
	}
	return v, true
}

// GetCuentaCliente godoc
// @Summary Cuenta del cliente
// @Description Devuelve la cuenta de un cliente: saldo (disponible, por pagar y deuda), cargos con su estado (PENDIENTE, PARCIAL o PAGADO) y abonos con lo que tienen sin usar. Saldo disponible es el dinero recargado y no usado; por pagar es lo que falta de los cargos vigentes; deuda es por pagar menos disponible y nunca es negativa. Las ventas de mostrador no lo mueven. Requiere token Bearer.
// @Tags Cuenta del cliente
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del cliente" example(7)
// @Param incluir_anulados query bool false "Incluye los cargos anulados en la lista (no cuentan en el saldo)" example(false)
// @Success 200 {object} utils.APIResponse{data=cuentaClienteResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido, incluir_anulados invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 404 {object} utils.APIResponse "Cliente no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/clientes/{id}/cuenta [get]
func (h *Container) GetCuentaCliente(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	incluir, err := queryBool(c, "incluir_anulados")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	cta, err := h.Cuenta.Obtener(id, incluir != nil && *incluir)
	if err != nil {
		respondCuentaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, cuentaClienteResponse{Cuenta: cta})
}

// PostCargoCliente godoc
// @Summary Crear cargo
// @Description Registra lo que un cliente debe: paquete, servicio, insumo u otro concepto. Un monto 0 registra un consumo que no se cobra. fecha_hecho permite registrar de noche lo anotado en papel (no puede ser futura ni anterior a cobros.dias_maximo_fecha_pasada dias; 0 = sin limite). Requiere token Bearer.
// @Tags Cuenta del cliente
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del cliente" example(7)
// @Param payload body crearCargoRequest true "Datos del cargo"
// @Success 201 {object} utils.APIResponse{data=cargoCreadoResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: tipo invalido, concepto requerido, monto negativo, fecha_hecho futura o con formato invalido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Local no permitido para el usuario"
// @Failure 404 {object} utils.APIResponse "Cliente no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/clientes/{id}/cargos [post]
func (h *Container) PostCargoCliente(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	var req crearCargoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}
	if req.Monto == nil {
		utils.RespondError(c, http.StatusBadRequest, "monto es requerido")
		return
	}
	localID := req.LocalID
	if localID != nil {
		if localID, ok = resolverLocalID(c, localID); !ok {
			return
		}
	}
	creadoPor, _ := authenticatedUsername(c)
	var creadoPorID *int
	if uid, ok := authenticatedUserID(c); ok {
		creadoPorID = &uid
	}
	cargoID, err := h.Cuenta.CrearCargo(services.CrearCargoInput{
		ClienteID: id, LocalID: localID, Tipo: req.Tipo, Concepto: req.Concepto, Monto: *req.Monto,
		FechaHecho: req.FechaHecho, ReservaID: req.ReservaID, PlanID: req.PlanID,
		CreadoPorID: creadoPorID, CreadoPor: creadoPor,
	})
	if err != nil {
		respondCuentaError(c, err)
		return
	}
	utils.Respond(c, http.StatusCreated, cargoCreadoResponse{ID: cargoID}, "Cargo creado")
}

// PostAnularCargo godoc
// @Summary Anular cargo
// @Description Anula un cargo con motivo; nunca se borra. Lo que los abonos le habian aplicado se libera y vuelve a ser saldo disponible. Requiere token Bearer con rol admin_sys o gerencia.
// @Tags Cuenta del cliente
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del cargo" example(12)
// @Param payload body anularCargoRequest true "Motivo de la anulacion"
// @Success 200 {object} utils.APIResponse "Cargo anulado"
// @Failure 400 {object} utils.APIResponse "Error de validacion: id invalido, motivo requerido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Rol no autorizado"
// @Failure 404 {object} utils.APIResponse "Cargo no encontrado o ya anulado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/cargos/{id}/anular [post]
func (h *Container) PostAnularCargo(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	var req anularCargoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}
	if err := h.Cuenta.AnularCargo(id, req.Motivo); err != nil {
		respondCuentaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, nil, "Cargo anulado")
}

// PostAbonoCliente godoc
// @Summary Registrar abono
// @Description Registra un pago a cuenta de un cliente y lo reparte entre sus cargos pendientes en una sola transaccion. Lo que no se aplica queda como saldo disponible. Un abono no aplica mas que su monto ni un cargo recibe mas que lo que tiene pendiente. No tiene detalle de servicios: las ventas de mostrador se siguen cobrando con POST /bd/pagos y no mueven el saldo. fecha_hecho permite registrar de noche lo anotado en papel. Requiere token Bearer; un usuario limitado a un local solo puede cobrar en el suyo.
// @Tags Cuenta del cliente
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del cliente" example(7)
// @Param payload body crearAbonoRequest true "Datos del abono y su reparto"
// @Success 201 {object} utils.APIResponse{data=abonoCreadoResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: monto no positivo, tipo de pago invalido, aplicacion que excede el pago o lo pendiente del cargo, cargo repetido, fecha_hecho futura"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 403 {object} utils.APIResponse "Local no permitido para el usuario"
// @Failure 404 {object} utils.APIResponse "Cliente o cargo no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/clientes/{id}/abonos [post]
func (h *Container) PostAbonoCliente(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	var req crearAbonoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}
	if req.LocalID == nil {
		utils.RespondError(c, http.StatusBadRequest, "local_id es requerido")
		return
	}
	localID, ok := resolverLocalID(c, req.LocalID)
	if !ok {
		return
	}
	cajero, ok := cajeroAuditoriaFromRequest(c, req.Nombre)
	if !ok {
		return
	}
	codigo, err := h.Cuenta.CrearAbono(services.CrearAbonoInput{
		ClienteID: id, LocalID: *localID, LocalNombre: req.LocalNombre, Monto: req.Monto, TipoPago: req.TipoPago,
		FechaHecho: req.FechaHecho, Aplicaciones: req.Aplicaciones, Cajero: cajero,
	})
	if err != nil {
		respondCuentaError(c, err)
		return
	}
	utils.Respond(c, http.StatusCreated, abonoCreadoResponse{CodigoPago: codigo}, "Abono registrado")
}

// PostAplicacionesAbono godoc
// @Summary Aplicar saldo de un abono
// @Description Reparte entre cargos pendientes lo que un abono ya existente tiene sin aplicar (saldo disponible que llego antes que el cargo). Valida que no se aplique mas de lo disponible del abono ni mas de lo pendiente de cada cargo. Requiere token Bearer.
// @Tags Cuenta del cliente
// @Accept json
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param id path int true "ID del cliente" example(7)
// @Param codigo_pago path string true "Codigo del abono" example(PAGO-000123)
// @Param payload body aplicarAbonoRequest true "Reparto entre cargos"
// @Success 200 {object} utils.APIResponse "Aplicaciones registradas"
// @Failure 400 {object} utils.APIResponse "Error de validacion: aplicaciones requerido, monto no positivo, excede lo disponible del abono o lo pendiente del cargo, cargo repetido"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 404 {object} utils.APIResponse "Abono o cargo no encontrado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/clientes/{id}/abonos/{codigo_pago}/aplicaciones [post]
func (h *Container) PostAplicacionesAbono(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	var req aplicarAbonoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}
	if err := h.Cuenta.AplicarAbono(c.Param("codigo_pago"), id, req.Aplicaciones); err != nil {
		respondCuentaError(c, err)
		return
	}
	utils.Respond(c, http.StatusOK, nil, "Aplicaciones registradas")
}

type deudoresFiltrosCursor struct {
	Busqueda   string `json:"busqueda,omitempty"`
	OrdenarPor string `json:"ordenar_por,omitempty"`
	Sentido    string `json:"sentido,omitempty"`
}

type deudoresResponse struct {
	// Cantidad de clientes devueltos en esta pagina.
	Total int `json:"total" example:"20"`
	// Clientes con deuda, con la mayor deuda primero salvo que se pida otro orden.
	Deudores []models.DeudorCliente `json:"deudores"`
	// Datos de paginacion.
	Paginacion *pagination.Metadata `json:"paginacion,omitempty"`
}

// GetDeudores godoc
// @Summary Clientes con deuda
// @Description Lista los clientes cuyo saldo es deuda (cargos no anulados mayores que abonos activos y pagados), con la mayor deuda primero salvo que se pida otro orden. El orden lo resuelve el servidor sobre todo el resultado; la paginacion avanza por desplazamiento y el cursor solo vale con la misma busqueda, ordenar_por y sentido que lo generaron. Requiere token Bearer.
// @Tags Cuenta del cliente
// @Produce json
// @Param Authorization header string true "Token Bearer" default(Bearer <token>)
// @Param busqueda query string false "Busqueda parcial por nombre, apellido o telefono" example(Maria)
// @Param ordenar_por query string false "Campo por el que ordenar; sin el, mayor deuda primero" Enums(deuda,total_cargos,total_abonos,nombre,apellido,numero_telefono) example(deuda)
// @Param sentido query string false "Sentido del orden; solo con ordenar_por, por defecto asc" Enums(asc,desc) example(desc)
// @Param limit query int false "Tamano de pagina opcional (1-100); sin limit ni cursor devuelve todos" example(50)
// @Param cursor query string false "Cursor opaco devuelto en paginacion.next_cursor"
// @Param include_total query bool false "Incluye el total de registros y de paginas" example(false)
// @Success 200 {object} utils.APIResponse{data=deudoresResponse}
// @Failure 400 {object} utils.APIResponse "Error de validacion: ordenar_por fuera de la lista, sentido distinto de asc o desc, cursor de otro orden o busqueda"
// @Failure 401 {object} utils.APIResponse "Token requerido, invalido o expirado"
// @Failure 500 {object} utils.APIResponse "Error interno del servidor"
// @Security BearerAuth
// @Router /bd/cuentas/deudores [get]
func (h *Container) GetDeudores(c *gin.Context) {
	page, ok := parsePagination(c)
	if !ok {
		return
	}
	includeTotal, ok := parseIncludeTotal(c)
	if !ok {
		return
	}
	sort, ok := parseSort(c, pgsqlrepo.DeudoresOrdenables)
	if !ok {
		return
	}
	filtros := deudoresFiltrosCursor{Busqueda: strings.TrimSpace(c.Query("busqueda"))}
	if sort.Active() {
		filtros.OrdenarPor, filtros.Sentido = sort.Key, sort.Direction()
	}
	// Se pagina siempre por desplazamiento: el orden depende de totales calculados.
	after, ok := decodeOffsetCursor(c, page, "deudores", filtros)
	if !ok {
		return
	}
	filtro := services.FiltroDeudoresInput{
		Busqueda: filtros.Busqueda, OrdenarPor: sort.Key, OrdenDesc: sort.Desc, PageLimit: page.QueryLimit(), Offset: after.Offset,
	}
	deudores, err := h.Cuenta.ListarDeudores(filtro)
	if err != nil {
		respondCuentaError(c, err)
		return
	}
	deudores, metadata, err := pagination.Build(deudores, page, "deudores", filtros, func(models.DeudorCliente) any {
		return pagination.OffsetPosition{Offset: after.Offset + page.Limit}
	})
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if includeTotal {
		filtro.PageLimit, filtro.Offset = 0, 0
		total, err := h.Cuenta.ContarDeudores(filtro)
		if err != nil {
			respondCuentaError(c, err)
			return
		}
		pagination.AddTotal(metadata, total)
	}
	utils.Respond(c, http.StatusOK, deudoresResponse{Total: len(deudores), Deudores: deudores, Paginacion: metadata})
}
