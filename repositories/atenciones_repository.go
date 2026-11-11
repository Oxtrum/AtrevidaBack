package repository

import "time"

// FiltroAtenciones filtra el historial de atenciones activas. Todos los campos
// son opcionales; LocalID se resuelve con el alcance del usuario autenticado.
type FiltroAtenciones struct {
	LocalID         *int
	FechaDesde      *time.Time
	FechaHasta      *time.Time
	Cliente         string
	Servicio        string
	Empleado        string
	EmpleadoID      *int
	ItemID          *int
	PendientesStock *bool
	// Resultado admite ATENDIDA, CANCELADA o INASISTENCIA (cancelada cuyo
	// motivo es INASISTENCIA).
	Resultado string
	// Q busca por cliente, reserva, servicio, personal, lugar u observaciones.
	Q     string
	Limit int
	// Paginacion por cursor (fecha_atencion DESC, id DESC): PageLimit > 0 la
	// activa y pide una fila extra para saber si hay pagina siguiente.
	PageLimit   int
	CursorSet   bool
	CursorFecha time.Time
	CursorID    int
	// OrdenarPor es una clave de pgsql.AtencionesOrdenables. Con orden pedido la
	// paginacion avanza por Offset y no por CursorFecha/CursorID.
	OrdenarPor string
	OrdenDesc  bool
	Offset     int
}

// FiltroJornada consulta las reservas de un dia y su atencion activa, si la
// tienen. El estado operativo se deriva en el service con la zona horaria
// configurada.
type FiltroJornada struct {
	Fecha     time.Time
	LocalID   *int
	Estado    string
	Q         string
	Ahora     time.Time
	PageLimit int
	Offset    int
}

// JornadaRow es la proyeccion cruda de una reserva del dia con los datos de su
// atencion activa. El service calcula el estado operativo y las acciones.
type JornadaRow struct {
	ReservaID             int       `db:"reserva_id"`
	Fecha                 time.Time `db:"fecha"`
	HoraDesde             string    `db:"hora_desde"`
	HoraHasta             string    `db:"hora_hasta"`
	Cliente               string    `db:"cliente"`
	Servicio              string    `db:"servicio"`
	LocalID               *int      `db:"local_id"`
	LocalNombre           string    `db:"local_nombre"`
	EstadoReserva         string    `db:"estado_reserva"`
	MotivoCancelacionTipo *string   `db:"motivo_cancelacion_tipo"`
	AtencionID            *int      `db:"atencion_id"`
	EmpleadoNombre        *string   `db:"empleado_nombre"`
	InsumosRegistrados    int       `db:"insumos_registrados"`
	InsumosResumen        *string   `db:"insumos_resumen"`
	InsumosPendientes     int       `db:"insumos_pendientes"`
}

// FiltroCalendarioPersonal consulta atenciones activas para la agenda semanal
// de Personal. Se mantiene separado de FiltroConsumos porque su contenido
// principal no son líneas de inventario.
type FiltroCalendarioPersonal struct {
	LocalID    *int
	FechaDesde time.Time
	FechaHasta time.Time
	Cliente    string
	Servicio   string
	Empleado   string
	Q          string
	PageLimit  int
	Offset     int
}

// FiltroSeguimiento reúne los casos operativos que requieren una acción. No
// introduce estados nuevos: compone hechos existentes de reservas y atenciones.
type FiltroSeguimiento struct {
	LocalID    *int
	FechaDesde time.Time
	FechaHasta time.Time
	Tipo       string
	Q          string
	Ahora      time.Time
	PageLimit  int
	Offset     int
}
