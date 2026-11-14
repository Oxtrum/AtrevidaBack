package models

// ConsumoInsumo es una linea de uso declarado dentro de una atencion. La
// cantidad pendiente es la diferencia entre lo usado y lo descontado del stock.
type ConsumoInsumo struct {
	ItemID             *int    `db:"item_id" json:"item_id,omitempty" example:"14"`
	ItemNombre         string  `db:"item_nombre" json:"item_nombre" example:"Alcohol en gel"`
	UnidadMedida       string  `db:"unidad_medida" json:"unidad_medida" example:"LT"`
	CantidadUsada      float64 `db:"cantidad_usada" json:"cantidad_usada" example:"1.5"`
	CantidadDescontada float64 `db:"cantidad_descontada" json:"cantidad_descontada" example:"1.5"`
	CantidadPendiente  float64 `json:"cantidad_pendiente" example:"0"`
	EstadoDescuento    string  `db:"estado_descuento" json:"estado_descuento" example:"DESCONTADO"`
}

// ConsumoEvento es una atencion activa con su consumo declarado. Se usa en el
// calendario de Consumos y no reemplaza al Kardex.
type ConsumoEvento struct {
	AtencionID         int             `db:"atencion_id" json:"atencion_id" example:"8"`
	ReservaID          int             `db:"reserva_id" json:"reserva_id" example:"21"`
	Fecha              string          `db:"fecha" json:"fecha" example:"2026-11-10"`
	HoraInicio         string          `db:"hora_inicio" json:"hora_inicio" example:"09:00"`
	HoraFin            *string         `db:"hora_fin" json:"hora_fin,omitempty" example:"09:50"`
	HorarioProgramado  bool            `db:"horario_programado" json:"horario_programado" example:"false"`
	Cliente            string          `db:"cliente" json:"cliente" example:"Maria Lopez"`
	Servicio           string          `db:"servicio" json:"servicio" example:"Depilacion Laser"`
	LocalID            *int            `db:"local_id" json:"local_id,omitempty" example:"1"`
	LocalNombre        string          `db:"local_nombre" json:"local_nombre" example:"SAN MARTIN"`
	EmpleadoID         *int            `db:"empleado_id" json:"empleado_id,omitempty" example:"4"`
	EmpleadoNombre     string          `db:"empleado_nombre" json:"empleado_nombre" example:"Daniela Perez"`
	Lugar              string          `db:"lugar" json:"lugar" example:"Cabina 1"`
	Insumos            []ConsumoInsumo `db:"-" json:"insumos"`
	InsumosRegistrados int             `json:"insumos_registrados" example:"2"`
	InsumosPendientes  int             `json:"insumos_pendientes" example:"0"`
}

// ConsumoResumen agrega el consumo por insumo y unidad. No mezcla unidades
// distintas y cuenta atenciones distintas, no lineas.
type ConsumoResumen struct {
	ItemID             *int    `db:"item_id" json:"item_id,omitempty" example:"14"`
	ItemNombre         string  `db:"item_nombre" json:"item_nombre" example:"Alcohol en gel"`
	UnidadMedida       string  `db:"unidad_medida" json:"unidad_medida" example:"LT"`
	CantidadUsada      float64 `db:"cantidad_usada" json:"cantidad_usada" example:"4"`
	CantidadDescontada float64 `db:"cantidad_descontada" json:"cantidad_descontada" example:"4"`
	CantidadPendiente  float64 `db:"cantidad_pendiente" json:"cantidad_pendiente" example:"0"`
	Atenciones         int     `db:"atenciones" json:"atenciones" example:"2"`
}

// ConsumoEmpleadoResumen agrega la actividad de consumo por empleado. No suma
// cantidades entre unidades distintas; cuenta atenciones, lineas e insumos
// distintos, y separa las lineas y atenciones con deduccion pendiente.
type ConsumoEmpleadoResumen struct {
	EmpleadoID           *int   `db:"empleado_id" json:"empleado_id,omitempty" example:"4"`
	EmpleadoNombre       string `db:"empleado_nombre" json:"empleado_nombre" example:"Daniela Perez"`
	Atenciones           int    `db:"atenciones" json:"atenciones" example:"12"`
	InsumosRegistrados   int    `db:"insumos_registrados" json:"insumos_registrados" example:"30"`
	InsumosDistintos     int    `db:"insumos_distintos" json:"insumos_distintos" example:"8"`
	InsumosPendientes    int    `db:"insumos_pendientes" json:"insumos_pendientes" example:"2"`
	AtencionesPendientes int    `db:"atenciones_pendientes" json:"atenciones_pendientes" example:"2"`
	Locales              string `db:"locales" json:"locales" example:"SAN MARTIN"`
}
