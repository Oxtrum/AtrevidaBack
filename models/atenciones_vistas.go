package models

// CalendarioPersonalEvento es la proyección de una atención activa para la
// agenda semanal de Personal. Es distinta de ConsumoEvento: su dato principal
// es quién atendió, no los insumos que se declararon.
//
// El endpoint se incorporará en la fase de Calendario - Personal. Definir el
// contrato ahora evita reutilizar por error las respuestas de Inventario.
type CalendarioPersonalEvento struct {
	AtencionID        int     `db:"atencion_id" json:"atencion_id" example:"8"`
	ReservaID         int     `db:"reserva_id" json:"reserva_id" example:"21"`
	Fecha             string  `db:"fecha" json:"fecha" example:"2026-11-10"`
	HoraInicio        *string `db:"hora_inicio" json:"hora_inicio,omitempty" example:"09:00"`
	HoraFin           *string `db:"hora_fin" json:"hora_fin,omitempty" example:"09:50"`
	Cliente           string  `db:"cliente" json:"cliente" example:"Maria Lopez"`
	Servicio          string  `db:"servicio" json:"servicio" example:"Depilacion Laser"`
	LocalID           *int    `db:"local_id" json:"local_id,omitempty" example:"1"`
	LocalNombre       string  `db:"local_nombre" json:"local_nombre" example:"SAN MARTIN"`
	EmpleadoID        *int    `db:"empleado_id" json:"empleado_id,omitempty" example:"4"`
	EmpleadoNombre    string  `db:"empleado_nombre" json:"empleado_nombre" example:"Daniela Perez"`
	Lugar             string  `db:"lugar" json:"lugar" example:"Cabina 1"`
	InsumosPendientes int     `db:"insumos_pendientes" json:"insumos_pendientes" example:"0"`
}

// CalendarioConsumoEvento define el contrato de la agenda semanal de Consumo.
// Se mantiene separado del calendario de Personal aunque ambos partan de una
// atención: aquí los insumos y cantidades son información principal.
type CalendarioConsumoEvento struct {
	AtencionID         int             `json:"atencion_id" example:"8"`
	ReservaID          int             `json:"reserva_id" example:"21"`
	Fecha              string          `json:"fecha" example:"2026-11-10"`
	HoraInicio         *string         `json:"hora_inicio,omitempty" example:"09:00"`
	HoraFin            *string         `json:"hora_fin,omitempty" example:"09:50"`
	Cliente            string          `json:"cliente" example:"Maria Lopez"`
	Servicio           string          `json:"servicio" example:"Depilacion Laser"`
	LocalID            *int            `json:"local_id,omitempty" example:"1"`
	LocalNombre        string          `json:"local_nombre" example:"SAN MARTIN"`
	EmpleadoNombre     string          `json:"empleado_nombre" example:"Daniela Perez"`
	Insumos            []ConsumoInsumo `json:"insumos"`
	InsumosRegistrados int             `json:"insumos_registrados" example:"2"`
	InsumosPendientes  int             `json:"insumos_pendientes" example:"0"`
}

// TipoSeguimiento identifica el motivo operativo por el que un hecho sigue en
// la bandeja. No es un estado nuevo de reserva ni de atención.
type TipoSeguimiento string

const (
	SeguimientoDeduccionPendiente TipoSeguimiento = "DEDUCCION_PENDIENTE"
	SeguimientoInasistencia       TipoSeguimiento = "INASISTENCIA"
	SeguimientoCancelacion        TipoSeguimiento = "CANCELACION"
	SeguimientoPendienteCierre    TipoSeguimiento = "PENDIENTE_CIERRE"
)

// SeguimientoItem es el caso accionable de la futura bandeja de Seguimiento.
// Historial conserva todos los hechos; esta vista solo contiene los que
// requieren revisión o una acción del operador.
type SeguimientoItem struct {
	Tipo                     TipoSeguimiento `db:"tipo" json:"tipo" example:"DEDUCCION_PENDIENTE"`
	AtencionID               *int            `db:"atencion_id" json:"atencion_id,omitempty" example:"8"`
	ReservaID                int             `db:"reserva_id" json:"reserva_id" example:"21"`
	Fecha                    string          `db:"fecha" json:"fecha" example:"2026-11-10"`
	HoraInicio               *string         `db:"hora_inicio" json:"hora_inicio,omitempty" example:"09:00"`
	Cliente                  string          `db:"cliente" json:"cliente" example:"Maria Lopez"`
	Servicio                 string          `db:"servicio" json:"servicio" example:"Depilacion Laser"`
	LocalID                  *int            `db:"local_id" json:"local_id,omitempty" example:"1"`
	LocalNombre              string          `db:"local_nombre" json:"local_nombre" example:"SAN MARTIN"`
	EmpleadoNombre           *string         `db:"empleado_nombre" json:"empleado_nombre,omitempty" example:"Daniela Perez"`
	Motivo                   *string         `db:"motivo" json:"motivo,omitempty" example:"INASISTENCIA"`
	InsumosPendientes        int             `db:"insumos_pendientes" json:"insumos_pendientes" example:"1"`
	DetalleInsumosPendientes *string         `db:"detalle_insumos_pendientes" json:"detalle_insumos_pendientes,omitempty" example:"Gel: 1 ML"`
	PuedeAtender             bool            `db:"puede_atender" json:"puede_atender"`
	PuedeEditarAtencion      bool            `db:"puede_editar_atencion" json:"puede_editar_atencion"`
}

// SeguimientoResumen son los contadores independientes de la bandeja. Solo
// dependen de local y periodo, no del filtro de tipo ni del texto de búsqueda.
type SeguimientoResumen struct {
	DeduccionesPendientes int `json:"deducciones_pendientes" example:"2"`
	Inasistencias         int `json:"inasistencias" example:"1"`
	Cancelaciones         int `json:"cancelaciones" example:"3"`
	PendientesCierre      int `json:"pendientes_cierre" example:"4"`
}
