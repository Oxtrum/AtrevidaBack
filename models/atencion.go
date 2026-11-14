package models

import "time"

// Atencion registra la ejecucion real y el cierre operativo de una reserva.
type Atencion struct {
	ID              int       `db:"id" json:"id"`
	ReservaID       int       `db:"reserva_id" json:"reserva_id"`
	ServicioReserva string    `db:"servicio_reserva" json:"servicio_reserva" example:"Depilacion Laser"`
	ClienteNombre   string    `db:"cliente_nombre" json:"cliente_nombre" example:"Maria Lopez"`
	EmpleadoID      *int      `db:"empleado_id" json:"empleado_id,omitempty"`
	EmpleadoNombre  string    `db:"empleado_nombre" json:"empleado_nombre"`
	FechaAtencion   time.Time `db:"fecha_atencion" json:"fecha_atencion"`
	HoraInicio      *string   `db:"hora_inicio" json:"hora_inicio,omitempty"`
	HoraFin         *string   `db:"hora_fin" json:"hora_fin,omitempty"`
	LocalID         *int      `db:"local_id" json:"local_id,omitempty"`
	LocalNombre     string    `db:"local_nombre" json:"local_nombre" example:"SAN MARTIN"`
	Lugar           string    `db:"lugar" json:"lugar"`
	Observaciones   *string   `db:"observaciones" json:"observaciones,omitempty"`
	RegistradoPorID *int      `db:"registrado_por_id" json:"registrado_por_id,omitempty"`
	RegistradoPor   *string   `db:"registrado_por" json:"registrado_por,omitempty"`
	CreadoEn        time.Time `db:"creado_en" json:"creado_en"`
	Activo          bool      `db:"activo" json:"activo"`
	// Resultado distingue una atención realizada de una reserva cancelada que se
	// proyecta en el historial. Las cancelaciones no tienen atención física.
	Resultado                string           `db:"resultado" json:"resultado"`
	MotivoCancelacionTipo    *string          `db:"motivo_cancelacion_tipo" json:"motivo_cancelacion_tipo,omitempty"`
	MotivoCancelacionDetalle *string          `db:"motivo_cancelacion_detalle" json:"motivo_cancelacion_detalle,omitempty"`
	InsumosRegistrados       int              `db:"insumos_registrados" json:"insumos_registrados" example:"3"`
	InsumosResumen           *string          `db:"insumos_resumen" json:"insumos_resumen,omitempty" example:"Gel, Guantes"`
	InsumosPendientes        int              `db:"insumos_pendientes" json:"insumos_pendientes"`
	InsumosPendientesDetalle *string          `db:"insumos_pendientes_detalle" json:"insumos_pendientes_detalle,omitempty"`
	Insumos                  []AtencionInsumo `db:"-" json:"insumos"`
}

type AtencionInsumo struct {
	ID                 int     `db:"id" json:"id"`
	AtencionID         int     `db:"atencion_id" json:"atencion_id"`
	ItemID             *int    `db:"item_id" json:"item_id,omitempty"`
	ItemNombre         string  `db:"item_nombre" json:"item_nombre"`
	UnidadMedida       string  `db:"unidad_medida" json:"unidad_medida"`
	CantidadUsada      float64 `db:"cantidad_usada" json:"cantidad_usada"`
	CantidadDescontada float64 `db:"cantidad_descontada" json:"cantidad_descontada"`
	EstadoDescuento    string  `db:"estado_descuento" json:"estado_descuento"`
}

// PersonalSugerido es la proyeccion segura para el selector operativo.
type PersonalSugerido struct {
	ID          int     `db:"id" json:"id"`
	Nombre      string  `db:"nombre" json:"nombre"`
	Cargo       *string `db:"cargo" json:"cargo,omitempty"`
	LocalID     *int    `db:"local_id" json:"local_id,omitempty"`
	LocalNombre *string `db:"local_nombre" json:"local_nombre,omitempty"`
}
