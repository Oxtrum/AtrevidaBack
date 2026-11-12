package models

// ReporteClientes es el reporte de clientes de un periodo: quien genero mas dinero, quien tuvo mas citas,
// quien es mas fiel y quien debe mas, en general y por local, comparado con el periodo anterior equivalente.
// Solo trae totales y rankings acotados; la lista completa de clientes se pide paginada aparte.
type ReporteClientes struct {
	// Periodo pedido: semanal, quincenal o mensual
	Periodo string `json:"periodo" example:"mensual"`
	// Primer dia del periodo (YYYY-MM-DD)
	Desde string `json:"desde" example:"2026-11-01"`
	// Ultimo dia del periodo (YYYY-MM-DD)
	Hasta string `json:"hasta" example:"2026-11-30"`
	// Primer dia del periodo anterior con el que se compara
	DesdeAnterior string `json:"desde_anterior" example:"2026-10-01"`
	// Ultimo dia del periodo anterior con el que se compara
	HastaAnterior string `json:"hasta_anterior" example:"2026-10-31"`
	// Tope de la racha: cuantos periodos hacia atras se revisan para medir la fidelidad
	RachaMaxima int `json:"racha_maxima" example:"12"`
	// Todos los locales juntos
	General ReporteClientesBloque `json:"general"`
	// Un bloque por local con movimiento de clientes en el periodo o con deudas, del que mas ingreso al que menos
	Locales []ReporteClientesBloque `json:"locales"`
}

// ReporteClientesBloque agrupa las metricas de un ambito: general o un local.
type ReporteClientesBloque struct {
	// Local del bloque; null en el bloque general
	LocalID *int `json:"local_id" example:"1"`
	// Nombre del local; "General" en el bloque general
	LocalNombre string `json:"local_nombre" example:"SAN MARTIN"`
	// Totales del bloque
	Resumen ResumenClientes `json:"resumen"`
	// Los 10 clientes que mas dinero generaron en el periodo
	TopIngresos []ClienteReporte `json:"top_ingresos"`
	// Los 10 clientes con mas citas completadas en el periodo
	TopCitas []ClienteReporte `json:"top_citas"`
	// Los 10 clientes con mas periodos seguidos viniendo (minimo 2)
	TopFieles []ClienteReporte `json:"top_fieles"`
	// Los 10 clientes con mas reservas canceladas en el periodo
	TopCanceladas []ClienteReporte `json:"top_canceladas"`
	// Los 10 clientes que mas deben hoy (no depende del periodo); el total esta en resumen.deudores
	Deudores []ClienteDeuda `json:"deudores"`
	// Actividad por dia; solo trae los dias con movimiento
	Serie []ClientesDia `json:"serie"`
}

type ResumenClientes struct {
	// Clientes que pagaron o completaron una cita en el periodo
	ClientesActivos int `json:"clientes_activos" example:"84"`
	// Clientes activos del periodo anterior
	ClientesActivosAnterior int `json:"clientes_activos_anterior" example:"76"`
	// Cambio de clientes activos contra el periodo anterior en %; null si el anterior no tuvo
	VariacionClientesPct *float64 `json:"variacion_clientes_pct" example:"10.5"`
	// Clientes activos cuya primera actividad en el negocio cae en este periodo
	ClientesNuevos int `json:"clientes_nuevos" example:"19"`
	// Clientes activos que ya habian venido antes del periodo
	ClientesRecurrentes int `json:"clientes_recurrentes" example:"65"`
	// Dinero pagado por clientes registrados en el periodo
	Ingresos float64 `json:"ingresos" example:"18450.00"`
	// Dinero pagado por clientes registrados en el periodo anterior
	IngresosAnterior float64 `json:"ingresos_anterior" example:"16200.00"`
	// Cambio de ingresos contra el periodo anterior en %; null si el anterior fue 0
	VariacionIngresosPct *float64 `json:"variacion_ingresos_pct" example:"13.9"`
	// Ventas del periodo sin cliente asignado (no entran en ningun ranking)
	IngresosSinCliente float64 `json:"ingresos_sin_cliente" example:"2300.00"`
	// Porcentaje de las ventas del periodo sin cliente asignado
	IngresosSinClientePct float64 `json:"ingresos_sin_cliente_pct" example:"11.1"`
	// Clientes que pagaron algo en el periodo
	ClientesQuePagaron int `json:"clientes_que_pagaron" example:"70"`
	// Ingresos entre clientes que pagaron
	TicketPromedioCliente float64 `json:"ticket_promedio_cliente" example:"263.57"`
	// Ticket promedio por cliente del periodo anterior
	TicketPromedioClienteAnterior float64 `json:"ticket_promedio_cliente_anterior" example:"241.79"`
	// Citas completadas de clientes registrados
	Citas int `json:"citas" example:"212"`
	// Citas completadas de clientes registrados en el periodo anterior
	CitasAnterior int `json:"citas_anterior" example:"190"`
	// Citas completadas cuya reserva no esta vinculada a un cliente (no entran en ningun ranking)
	CitasSinCliente int `json:"citas_sin_cliente" example:"14"`
	// Reservas canceladas de clientes registrados
	Canceladas int `json:"canceladas" example:"17"`
	// De las canceladas, cuantas fueron por inasistencia
	Inasistencias int `json:"inasistencias" example:"6"`
	// Clientes activos del periodo anterior que volvieron en este
	ClientesQueVolvieron int `json:"clientes_que_volvieron" example:"48"`
	// Porcentaje de los clientes del periodo anterior que volvio; null si el anterior no tuvo clientes
	TasaRetornoPct *float64 `json:"tasa_retorno_pct" example:"63.2"`
	// Saldo pendiente hoy de todos los clientes del bloque
	PorCobrar float64 `json:"por_cobrar" example:"3120.00"`
	// Clientes con saldo pendiente hoy
	Deudores int `json:"deudores" example:"11"`
	// Clientes activos en mas de un local en el periodo; solo se calcula en el bloque general
	ClientesCompartidos int `json:"clientes_compartidos" example:"5"`
}

type ClienteReporte struct {
	// Puesto por dinero generado (1 = quien mas genero)
	Posicion int `json:"posicion" example:"1"`
	// ID del cliente
	ClienteID int `json:"cliente_id" example:"42"`
	// Nombre y apellido
	Nombre string `json:"nombre" example:"Ana Perez"`
	// Telefono registrado
	Telefono string `json:"telefono" example:"70012345"`
	// Locales donde tuvo movimiento en el periodo, separados por coma
	Locales string `json:"locales" example:"SAN MARTIN, PASEO ARANJUEZ"`
	// Dinero que pago en el periodo
	Ingresos float64 `json:"ingresos" example:"1250.00"`
	// Porcentaje de los ingresos de clientes del bloque
	ParticipacionPct float64 `json:"participacion_pct" example:"6.8"`
	// Dinero que pago en el periodo anterior
	IngresosAnterior float64 `json:"ingresos_anterior" example:"900.00"`
	// Cambio contra el periodo anterior en %; null si antes no pago
	VariacionPct *float64 `json:"variacion_pct" example:"38.9"`
	// Pagos registrados en el periodo
	Pagos int `json:"pagos" example:"3"`
	// Dinero por pago
	TicketPromedio float64 `json:"ticket_promedio" example:"416.67"`
	// Citas completadas en el periodo
	Citas int `json:"citas" example:"8"`
	// Citas completadas en el periodo anterior
	CitasAnterior int `json:"citas_anterior" example:"6"`
	// Reservas pendientes o agendadas dentro del periodo
	CitasAgendadas int `json:"citas_agendadas" example:"2"`
	// Reservas canceladas en el periodo
	Canceladas int `json:"canceladas" example:"1"`
	// De las canceladas, cuantas fueron por inasistencia
	Inasistencias int `json:"inasistencias" example:"0"`
	// Servicio que mas tomo (citas completadas)
	ServicioTop string `json:"servicio_top" example:"Masaje reductor"`
	// Veces que tomo ese servicio
	ServicioTopCantidad int `json:"servicio_top_cantidad" example:"5"`
	// Periodos seguidos, contando este, en los que pago o completo una cita; se corta en racha_maxima
	RachaPeriodos int `json:"racha_periodos" example:"4"`
	// true si su primera actividad en el negocio cae en este periodo
	EsNuevo bool `json:"es_nuevo" example:"false"`
	// Fecha de su ultimo movimiento del periodo (YYYY-MM-DD)
	UltimaActividad string `json:"ultima_actividad" example:"2026-11-10"`
	// Saldo pendiente hoy en el bloque
	SaldoPendiente float64 `json:"saldo_pendiente" example:"150.00"`
}

type ClienteDeuda struct {
	// ID del cliente
	ClienteID int `json:"cliente_id" example:"42"`
	// Nombre y apellido
	Nombre string `json:"nombre" example:"Ana Perez"`
	// Telefono registrado
	Telefono string `json:"telefono" example:"70012345"`
	// Saldo pendiente hoy
	SaldoPendiente float64 `json:"saldo_pendiente" example:"450.00"`
	// Cargos con algo pendiente
	Cargos int `json:"cargos" example:"2"`
	// Fecha del cargo pendiente mas antiguo (YYYY-MM-DD)
	DeudaDesde string `json:"deuda_desde" example:"2026-09-18"`
	// Dias que lleva pendiente el cargo mas antiguo
	DiasDeuda int `json:"dias_deuda" example:"55"`
}

type ClientesDia struct {
	// Dia (YYYY-MM-DD)
	Fecha string `json:"fecha" example:"2026-11-10"`
	// Clientes distintos que pagaron o completaron una cita ese dia
	Clientes int `json:"clientes" example:"9"`
	// Dinero pagado por clientes registrados ese dia
	Ingresos float64 `json:"ingresos" example:"870.00"`
}
