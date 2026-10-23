package models

// Empleado es una persona en planilla de sueldos.
type Empleado struct {
	ID            int     `db:"id" json:"id" example:"1"`
	Nombre        string  `db:"nombre" json:"nombre" example:"Maria"`
	Apellido      string  `db:"apellido" json:"apellido" example:"Rojas"`
	CI            *string `db:"ci" json:"ci,omitempty" example:"7654321"`
	Cargo         *string `db:"cargo" json:"cargo,omitempty" example:"Masajista"`
	Telefono      *string `db:"telefono" json:"telefono,omitempty"`
	LocalID       *int    `db:"local_id" json:"local_id,omitempty"`
	LocalNombre   *string `db:"local_nombre" json:"local_nombre,omitempty" example:"PASEO ARANJUEZ"`
	UsuarioID     *int    `db:"usuario_id" json:"usuario_id,omitempty"`
	SueldoBase    float64 `db:"sueldo_base" json:"sueldo_base" example:"3000"`
	DiaPago       *int    `db:"dia_pago" json:"dia_pago,omitempty" example:"5"`
	AplicaAportes bool    `db:"aplica_aportes" json:"aplica_aportes" example:"true"`
	MetodoPago    *string `db:"metodo_pago" json:"metodo_pago,omitempty" example:"transferencia"`
	FechaIngreso  string  `db:"fecha_ingreso" json:"fecha_ingreso" example:"2026-01-01"`
	FechaBaja     *string `db:"fecha_baja" json:"fecha_baja,omitempty"`
	Activo        bool    `db:"activo" json:"activo" example:"true"`
	Notas         *string `db:"notas" json:"notas,omitempty"`
	// DiaPagoEfectivo es el dia propio o el por defecto de configuracion; se calcula en el servicio.
	DiaPagoEfectivo int `db:"-" json:"dia_pago_efectivo" example:"1"`
	// AdelantosPendientes suma los adelantos aun no descontados en una planilla.
	AdelantosPendientes float64 `db:"adelantos_pendientes" json:"adelantos_pendientes" example:"200"`
}

// Adelanto es un anticipo de sueldo que se descuenta en la siguiente planilla.
type Adelanto struct {
	ID             int     `db:"id" json:"id" example:"1"`
	EmpleadoID     int     `db:"empleado_id" json:"empleado_id" example:"1"`
	EmpleadoNombre string  `db:"empleado_nombre" json:"empleado_nombre" example:"Maria Rojas"`
	Fecha          string  `db:"fecha" json:"fecha" example:"2026-10-10"`
	Monto          float64 `db:"monto" json:"monto" example:"200"`
	Motivo         *string `db:"motivo" json:"motivo,omitempty"`
	PlanillaID     *int    `db:"planilla_id" json:"planilla_id,omitempty"`
	Anulado        bool    `db:"anulado" json:"anulado" example:"false"`
}

// Planilla es el sueldo de un empleado en un periodo, con su desglose.
type Planilla struct {
	ID               int     `db:"id" json:"id" example:"1"`
	EmpleadoID       int     `db:"empleado_id" json:"empleado_id" example:"1"`
	EmpleadoNombre   string  `db:"empleado_nombre" json:"empleado_nombre" example:"Maria Rojas"`
	Cargo            *string `db:"cargo" json:"cargo,omitempty"`
	LocalID          *int    `db:"local_id" json:"local_id,omitempty"`
	LocalNombre      *string `db:"local_nombre" json:"local_nombre,omitempty"`
	Tipo             string  `db:"tipo" json:"tipo" example:"SUELDO"`
	Periodo          string  `db:"periodo" json:"periodo" example:"2026-10-01"`
	FechaPago        string  `db:"fecha_pago" json:"fecha_pago" example:"2026-10-05"`
	SueldoBase       float64 `db:"sueldo_base" json:"sueldo_base" example:"3000"`
	Bonos            float64 `db:"bonos" json:"bonos" example:"150"`
	DiasFalta        float64 `db:"dias_falta" json:"dias_falta" example:"1"`
	DescuentoFaltas  float64 `db:"descuento_faltas" json:"descuento_faltas" example:"100"`
	TotalGanado      float64 `db:"total_ganado" json:"total_ganado" example:"3050"`
	AporteLaboral    float64 `db:"aporte_laboral" json:"aporte_laboral" example:"387.7"`
	Adelantos        float64 `db:"adelantos" json:"adelantos" example:"200"`
	OtrosDescuentos  float64 `db:"otros_descuentos" json:"otros_descuentos" example:"0"`
	NetoPagar        float64 `db:"neto_pagar" json:"neto_pagar" example:"2462.3"`
	AportePatronal   float64 `db:"aporte_patronal" json:"aporte_patronal" example:"509.65"`
	CostoEmpresa     float64 `db:"costo_empresa" json:"costo_empresa" example:"3559.65"`
	Estado           string  `db:"estado" json:"estado" example:"PENDIENTE"`
	MotivoAnulacion  *string `db:"motivo_anulacion" json:"motivo_anulacion,omitempty"`
	Notas            *string `db:"notas" json:"notas,omitempty"`
	MovimientoCodigo *string `db:"movimiento_codigo" json:"movimiento_codigo,omitempty" example:"MOV-000012"`
	// BonosDetalle son las lineas de bono que suman el campo bonos; se cargan aparte en el repositorio.
	BonosDetalle []PlanillaBono `db:"-" json:"bonos_detalle"`
}

// TipoBono es un concepto de bono del catalogo (ej. Produccion, Puntualidad).
type TipoBono struct {
	ID          int     `db:"id" json:"id" example:"1"`
	Nombre      string  `db:"nombre" json:"nombre" example:"Producción"`
	Descripcion *string `db:"descripcion" json:"descripcion,omitempty" example:"Bono por metas del mes"`
	Activo      bool    `db:"activo" json:"activo" example:"true"`
}

// PlanillaBono es una linea de bono de una planilla: de que tipo es, cuanto y por que.
type PlanillaBono struct {
	ID         int     `db:"id" json:"id" example:"1"`
	PlanillaID int     `db:"planilla_id" json:"-"`
	TipoBonoID int     `db:"tipo_bono_id" json:"tipo_bono_id" example:"1"`
	TipoNombre string  `db:"tipo_nombre" json:"tipo_nombre" example:"Producción"`
	Monto      float64 `db:"monto" json:"monto" example:"150"`
	Nota       *string `db:"nota" json:"nota,omitempty" example:"Meta de octubre"`
}

// PlanillaResumen son los totales de un periodo (planillas no anuladas).
type PlanillaResumen struct {
	Periodo        string  `json:"periodo" example:"2026-10"`
	Empleados      int     `json:"empleados" example:"5"`
	TotalGanado    float64 `json:"total_ganado" example:"15000"`
	AporteLaboral  float64 `json:"aporte_laboral" example:"1906.5"`
	Adelantos      float64 `json:"adelantos" example:"200"`
	Descuentos     float64 `json:"descuentos" example:"100"`
	NetoPagar      float64 `json:"neto_pagar" example:"12793.5"`
	AportePatronal float64 `json:"aporte_patronal" example:"2506.5"`
	CostoEmpresa   float64 `json:"costo_empresa" example:"17506.5"`
	Pagado         float64 `json:"pagado" example:"5000"`
	Pendiente      float64 `json:"pendiente" example:"12506.5"`
}

// PlanillaPendiente es una planilla vencida que aun no se genero (vista previa de "generar").
type PlanillaPendiente struct {
	EmpleadoID     int     `json:"empleado_id" example:"1"`
	EmpleadoNombre string  `json:"empleado_nombre" example:"Maria Rojas"`
	Periodo        string  `json:"periodo" example:"2026-09"`
	FechaPago      string  `json:"fecha_pago" example:"2026-09-05"`
	SueldoBase     float64 `json:"sueldo_base" example:"3000"`
}
