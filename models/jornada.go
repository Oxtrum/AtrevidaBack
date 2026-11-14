package models

import "atrevida-agenda-api/pagination"

// JornadaAtencion es una fila de la vista operativa del dia. Una reserva
// programada aparece una sola vez; si ya tiene una atencion activa se marca
// como ATENDIDA.
type JornadaAtencion struct {
	ReservaID             int     `db:"reserva_id" json:"reserva_id" example:"100"`
	Fecha                 string  `db:"fecha" json:"fecha" example:"2026-11-09"`
	HoraDesde             string  `db:"hora_desde" json:"hora_desde" example:"09:00"`
	HoraHasta             string  `db:"hora_hasta" json:"hora_hasta" example:"10:00"`
	Cliente               string  `db:"cliente" json:"cliente" example:"Maria Lopez"`
	Servicio              string  `db:"servicio" json:"servicio" example:"Depilacion Laser"`
	LocalID               *int    `db:"local_id" json:"local_id,omitempty" example:"2"`
	LocalNombre           string  `db:"local_nombre" json:"local_nombre" example:"SAN MARTIN"`
	EstadoReserva         string  `db:"estado_reserva" json:"estado_reserva" example:"AGENDADO"`
	MotivoCancelacionTipo *string `db:"motivo_cancelacion_tipo" json:"motivo_cancelacion_tipo,omitempty"`
	EstadoOperativo       string  `db:"estado_operativo" json:"estado_operativo" example:"PROGRAMADA"`
	AtencionID            *int    `db:"atencion_id" json:"atencion_id,omitempty" example:"55"`
	EmpleadoNombre        *string `db:"empleado_nombre" json:"empleado_nombre,omitempty" example:"Ana Perez"`
	InsumosRegistrados    int     `db:"insumos_registrados" json:"insumos_registrados" example:"3"`
	InsumosResumen        *string `db:"insumos_resumen" json:"insumos_resumen,omitempty" example:"Gel, Guantes"`
	InsumosPendientes     int     `db:"insumos_pendientes" json:"insumos_pendientes" example:"0"`
	// Acciones calculadas en backend para no duplicar reglas en el frontend.
	PuedeAtender        bool `db:"-" json:"puede_atender"`
	PuedeVerAtencion    bool `db:"-" json:"puede_ver_atencion"`
	PuedeEditarAtencion bool `db:"-" json:"puede_editar_atencion"`
}

// JornadaResumen agrega el total operativo del día para fecha y local. No se
// reduce al filtrar las filas por estado o texto: esos filtros sirven para la
// lista, mientras este resumen permite navegar entre estados.
type JornadaResumen struct {
	Programadas           int `json:"programadas" example:"4"`
	Atendidas             int `json:"atendidas" example:"2"`
	PendientesCierre      int `json:"pendientes_cierre" example:"1"`
	Canceladas            int `json:"canceladas" example:"1"`
	NoAsistieron          int `json:"no_asistieron" example:"0"`
	DeduccionesPendientes int `json:"deducciones_pendientes" example:"1"`
}

// JornadaResponse es la respuesta de la vista operativa de un dia.
type JornadaResponse struct {
	// Fecha consultada; por defecto la fecha actual de la zona horaria configurada.
	Fecha string `json:"fecha" example:"2026-11-09"`
	// Fecha actual de la zona horaria configurada, usada por el boton "Hoy".
	Hoy string `json:"hoy" example:"2026-11-10"`
	// Reservas del dia con su estado operativo derivado.
	Items []JornadaAtencion `json:"items"`
	// Se conserva por compatibilidad; la interfaz operativa usa
	// /jornada/resumen para evitar que la lista y sus filtros lo condicionen.
	Resumen JornadaResumen `json:"resumen"`
	// Metadatos de cursor presentes cuando se consulta limit o cursor.
	Paginacion *pagination.Metadata `json:"paginacion,omitempty"`
	// Total de filas de la lista filtrada, solo con include_total=true.
	TotalRegistros *int `json:"total_registros,omitempty"`
}
