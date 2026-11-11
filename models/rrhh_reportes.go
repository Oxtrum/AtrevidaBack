package models

// ReporteAtencionesRRHH es el reporte de productividad del personal: quien atendio mas en un periodo,
// en general y por local, comparado con el periodo anterior equivalente.
type ReporteAtencionesRRHH struct {
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
	// Todos los locales juntos
	General ReporteAtencionesBloque `json:"general"`
	// Un bloque por local con atenciones en el periodo, del que mas atendio al que menos
	Locales []ReporteAtencionesBloque `json:"locales"`
}

// ReporteAtencionesBloque agrupa las metricas de un ambito: general o un local.
type ReporteAtencionesBloque struct {
	// Local del bloque; null en el bloque general y en atenciones sin local
	LocalID *int `json:"local_id" example:"1"`
	// Nombre del local; "General" en el bloque general
	LocalNombre string `json:"local_nombre" example:"SAN MARTIN"`
	// Totales del bloque
	Resumen ResumenAtencionesRRHH `json:"resumen"`
	// Personal ordenado del que mas atendio al que menos
	Ranking []PersonalAtenciones `json:"ranking"`
	// Atenciones por dia; solo trae los dias con atenciones
	Serie []AtencionesDia `json:"serie"`
	// Usuarios que registraron las atenciones en el sistema
	Registradores []RegistradorAtenciones `json:"registradores"`
}

type ResumenAtencionesRRHH struct {
	// Atenciones del periodo
	Atenciones int `json:"atenciones" example:"124"`
	// Atenciones del periodo anterior
	AtencionesAnterior int `json:"atenciones_anterior" example:"110"`
	// Cambio contra el periodo anterior en %; null si el anterior no tuvo atenciones
	VariacionPct *float64 `json:"variacion_pct" example:"12.7"`
	// Personas distintas que atendieron
	Personal int `json:"personal" example:"6"`
	// Dias con al menos una atencion
	DiasConAtencion int `json:"dias_con_atencion" example:"24"`
	// Atenciones por dia trabajado
	PromedioDiario float64 `json:"promedio_diario" example:"5.2"`
	// Atenciones por persona
	PromedioPorPersona float64 `json:"promedio_por_persona" example:"20.7"`
	// Clientes distintos atendidos
	ClientesDistintos int `json:"clientes_distintos" example:"80"`
	// Minutos atendidos, sumando solo las atenciones con hora de inicio y fin
	MinutosAtendidos int `json:"minutos_atendidos" example:"7440"`
	// Duracion media en minutos de las atenciones con horario
	DuracionPromedioMin float64 `json:"duracion_promedio_min" example:"60"`
	// Atenciones registradas sin hora de inicio o fin
	SinHorario int `json:"sin_horario" example:"3"`
}

type PersonalAtenciones struct {
	// Puesto en el ranking (1 = quien mas atendio)
	Posicion int `json:"posicion" example:"1"`
	// Empleado de RRHH; null si se escribio un nombre que no esta en Empleados
	EmpleadoID *int `json:"empleado_id" example:"4"`
	// Nombre de quien atendio
	Nombre string `json:"nombre" example:"Maria Rojas"`
	// Cargo del empleado
	Cargo *string `json:"cargo" example:"Masajista"`
	// Locales donde atendio en el periodo, separados por coma
	Locales string `json:"locales" example:"SAN MARTIN, PASEO ARANJUEZ"`
	// Atenciones del periodo
	Atenciones int `json:"atenciones" example:"38"`
	// Porcentaje de las atenciones del bloque
	ParticipacionPct float64 `json:"participacion_pct" example:"30.6"`
	// Atenciones del periodo anterior
	AtencionesAnterior int `json:"atenciones_anterior" example:"31"`
	// Cambio contra el periodo anterior en %; null si antes no atendio
	VariacionPct *float64 `json:"variacion_pct" example:"22.6"`
	// Dias en los que atendio
	DiasTrabajados int `json:"dias_trabajados" example:"20"`
	// Atenciones por dia trabajado
	PromedioPorDia float64 `json:"promedio_por_dia" example:"1.9"`
	// Clientes distintos atendidos
	ClientesDistintos int `json:"clientes_distintos" example:"27"`
	// Minutos atendidos (solo atenciones con horario)
	MinutosAtendidos int `json:"minutos_atendidos" example:"2280"`
	// Duracion media en minutos de sus atenciones con horario
	DuracionPromedioMin float64 `json:"duracion_promedio_min" example:"60"`
	// Servicio que mas realizo
	ServicioTop string `json:"servicio_top" example:"Masaje reductor"`
	// Veces que realizo ese servicio
	ServicioTopCantidad int `json:"servicio_top_cantidad" example:"14"`
	// Fecha de su ultima atencion del periodo (YYYY-MM-DD)
	UltimaAtencion string `json:"ultima_atencion" example:"2026-11-10"`
}

type AtencionesDia struct {
	// Dia (YYYY-MM-DD)
	Fecha string `json:"fecha" example:"2026-11-10"`
	// Atenciones de ese dia
	Atenciones int `json:"atenciones" example:"7"`
}

type RegistradorAtenciones struct {
	// Usuario que registro la atencion en el sistema
	Usuario string `json:"usuario" example:"recepcion.sm"`
	// Atenciones que registro
	Registros int `json:"registros" example:"52"`
	// Porcentaje de las atenciones del bloque
	ParticipacionPct float64 `json:"participacion_pct" example:"41.9"`
}
