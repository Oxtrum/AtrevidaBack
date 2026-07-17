package models

import (
	"time"
)

// Locales

type LocalPG struct {
	ID     int    `db:"id" json:"id" example:"3"`
	Nombre string `db:"nombre" json:"nombre" example:"SAN MARTIN"`
	Activo bool   `db:"activo" json:"activo" example:"true"`
}
type TipoEspacioLocal struct {
	TipoEspacio      string `db:"tipo_espacio"       json:"tipo_espacio" example:"M"`
	CantidadEspacios int    `db:"cantidad_espacios"  json:"cantidad_espacios" example:"6"`
}

type LocalConEspacios struct {
	ID       int                `db:"id"     json:"id" example:"3"`
	Nombre   string             `db:"nombre" json:"nombre" example:"SAN MARTIN"`
	Activo   bool               `db:"activo" json:"activo" example:"true"`
	Espacios []TipoEspacioLocal `db:"-"     json:"espacios"`
	Horarios []LocalHorarioPG   `db:"-"     json:"horarios"`
}

type LocalHorarioPG struct {
	ID        int    `db:"id" json:"id" example:"15"`
	LocalID   int    `db:"local_id" json:"local_id" example:"3"`
	DiaSemana int    `db:"dia_semana" json:"dia_semana" example:"1"`
	HoraDesde string `db:"hora_desde" json:"hora_desde" example:"09:00"`
	HoraHasta string `db:"hora_hasta" json:"hora_hasta" example:"18:00"`
	Activo    bool   `db:"activo" json:"activo" example:"true"`
}

// Categorías

type CategoriaPG struct {
	ID     int    `db:"id" json:"id" example:"4"`
	Nombre string `db:"nombre" json:"nombre" example:"Corporal"`
}

// Clientes

type ClientePG struct {
	ID             int    `db:"id" json:"id" example:"12"`
	Nombre         string `db:"nombre" json:"nombre" example:"Maria"`
	Apellido       string `db:"apellido" json:"apellido" example:"Lopez"`
	NumeroTelefono string `db:"numero_telefono" json:"numero_telefono" example:"+59170011223"`
}

// Usuarios

type UsuarioPG struct {
	// ID interno del usuario.
	ID int `db:"id" json:"id" example:"1"`
	// Nombre de usuario.
	Username string `db:"username" json:"username" example:"admin"`
	// Password hasheada; no se expone en respuestas JSON.
	Password string `db:"password" json:"-"`
	// Estado activo del usuario.
	Activo bool `db:"activo" json:"activo" example:"true"`
	// Fecha de registro del usuario.
	FechaRegistro time.Time `db:"fecha_registro" json:"fecha_registro" example:"2026-05-28T14:30:00Z"`
	// ID interno del rol asignado.
	RolID int `db:"rol_id" json:"rol_id" example:"1"`
	// Codigo del rol asignado.
	RolCodigo string `db:"rol_codigo" json:"rol_codigo" example:"admin_sys"`
	// Nombre descriptivo del rol asignado.
	RolNombre string `db:"rol_nombre" json:"rol_nombre" example:"Administrador de sistema"`
	// ID del local asignado al usuario; null para administradores.
	LocalID *int `db:"local_id" json:"local_id,omitempty" example:"1"`
	// Nombre del local asignado al usuario; null para administradores.
	NombreLocal *string `db:"nombre_local" json:"nombre_local,omitempty" example:"SAN MARTIN"`
}

type UsuarioResumenPG struct {
	// Nombre de usuario.
	Username string `db:"username" json:"username" example:"admin"`
	// Estado activo del usuario.
	Activo bool `db:"activo" json:"activo" example:"true"`
	// Fecha de registro del usuario.
	FechaRegistro time.Time `db:"fecha_registro" json:"fecha_registro" example:"2026-05-28T14:30:00Z"`
	// Codigo del rol asignado.
	RolCodigo string `db:"rol_codigo" json:"rol_codigo" example:"admin_sys"`
	// Nombre descriptivo del rol asignado.
	RolNombre string `db:"rol_nombre" json:"rol_nombre" example:"Administrador de sistema"`
	// ID del local asignado al usuario; null para administradores.
	LocalID *int `db:"local_id" json:"local_id,omitempty" example:"1"`
	// Nombre del local asignado al usuario; null para administradores.
	NombreLocal *string `db:"nombre_local" json:"nombre_local,omitempty" example:"SAN MARTIN"`
}

// Servicios

type ServicioPG struct {
	ID                 int      `db:"id"`
	Nombre             string   `db:"nombre"`
	CategoriaID        *int     `db:"categoria_id"`
	Tiempo             *string  `db:"tiempo"`
	Costo              *float64 `db:"costo"`
	Sesiones           int      `db:"sesiones"`
	Activo             bool     `db:"activo"`
	RequiereEvaluacion bool     `db:"requiere_evaluacion"`
}

// ServicioPGConLocal
type ServicioPGConLocal struct {
	ServicioPG
	Categoria string `db:"categoria_nombre"`
	Locales   string `db:"locales"` // nombres separados por coma, agregados con STRING_AGG
}

// Combos

type ComboPG struct {
	ID              int      `db:"id"`
	Nombre          string   `db:"nombre"`
	CategoriaID     *int     `db:"categoria_id"`
	CostoTotal      *float64 `db:"costo_total"`
	SesionesTotales int      `db:"sesiones_totales"`
	Activo          bool     `db:"activo"`
}

// ComboCatalogoPG representa una promocion reutilizable del catalogo.
// No representa una compra ni conserva progreso de clientes.
type ComboCatalogoPG struct {
	ID              int                      `db:"id" json:"id" example:"12"`
	Nombre          string                   `db:"nombre" json:"nombre" example:"Combo Relax"`
	Descripcion     *string                  `db:"descripcion" json:"descripcion,omitempty" example:"Promocion corporal de cuatro sesiones"`
	CategoriaID     *int                     `db:"categoria_id" json:"categoria_id,omitempty" example:"3"`
	Categoria       string                   `db:"categoria" json:"categoria" example:"Corporal"`
	TipoPrecio      string                   `db:"tipo_precio" json:"tipo_precio" example:"PRECIO_PAQUETE"`
	PrecioPaquete   *float64                 `db:"precio_paquete" json:"precio_paquete,omitempty" example:"700"`
	PrecioItems     float64                  `db:"precio_items" json:"precio_items" example:"800"`
	PrecioFinal     float64                  `db:"precio_final" json:"precio_final" example:"700"`
	Moneda          string                   `db:"moneda" json:"moneda" example:"BOB"`
	SesionesTotales int                      `db:"sesiones_totales" json:"sesiones_totales" example:"4"`
	DuracionMin     *int                     `db:"duracion_min" json:"duracion_min,omitempty" example:"90"`
	Activo          bool                     `db:"activo" json:"activo" example:"true"`
	CreadoEn        time.Time                `db:"creado_en" json:"creado_en" example:"2026-07-11T10:00:00Z"`
	ActualizadoEn   time.Time                `db:"actualizado_en" json:"actualizado_en" example:"2026-07-11T10:00:00Z"`
	// Path del objeto de portada en Supabase Storage; interno, no se expone en JSON.
	ImagenPath *string `db:"imagen_path" json:"-"`
	// URL publica de la portada derivada del path; nil si el combo no tiene imagen.
	ImagenURL       *string                  `db:"-" json:"imagen_url,omitempty" example:"https://xxx.supabase.co/storage/v1/object/public/paquetes/combos/12"`
	Locales         []LocalPG                `db:"-" json:"locales"`
	Servicios       []ComboServicioDetallePG `db:"-" json:"servicios"`
}

type ComboServicioPG struct {
	ID             int      `db:"id" json:"id" example:"15"`
	ComboID        int      `db:"combo_id" json:"combo_id" example:"12"`
	ServicioID     *int     `db:"servicio_id" json:"servicio_id,omitempty" example:"8"`
	ServicioTexto  *string  `db:"servicio_texto" json:"servicio_texto,omitempty" example:"Masaje relajante personalizado"`
	Tiempo         *string  `db:"tiempo" json:"tiempo,omitempty" example:"01:00"`
	Costo          *float64 `db:"costo" json:"costo,omitempty" example:"250"`
	Sesiones       int      `db:"sesiones" json:"sesiones" example:"2"`
	Orden          int      `db:"orden" json:"orden" example:"1"`
	ServicioNombre string   `db:"servicio_nombre" json:"servicio_nombre" example:"Masaje relajante personalizado"`
}

type ComboServicioDetallePG struct {
	ID             int      `db:"id" json:"id" example:"15"`
	ComboID        int      `db:"combo_id" json:"combo_id" example:"12"`
	ComboNombre    string   `db:"combo_nombre" json:"combo_nombre" example:"Combo Relax"`
	ServicioID     *int     `db:"servicio_id" json:"servicio_id,omitempty" example:"8"`
	ServicioTexto  *string  `db:"servicio_texto" json:"servicio_texto,omitempty" example:"Masaje relajante personalizado"`
	ServicioNombre string   `db:"servicio_nombre" json:"servicio_nombre" example:"Masaje relajante personalizado"`
	Tiempo         *string  `db:"tiempo" json:"tiempo,omitempty" example:"01:00"`
	Costo          *float64 `db:"costo" json:"costo,omitempty" example:"250"`
	Sesiones       int      `db:"sesiones" json:"sesiones" example:"2"`
	SesionNumero   int      `db:"sesion_numero" json:"sesion_numero" example:"1"`
	Orden          int      `db:"orden" json:"orden" example:"1"`
	Activo         bool     `db:"activo" json:"activo" example:"true"`
}

// Planes

type PlanPG struct {
	ID                    int        `db:"id" json:"id" example:"21"`
	Codigo                string     `db:"codigo" json:"codigo" example:"PLAN-000001"`
	Cliente               string     `db:"cliente" json:"cliente" example:"Maria Lopez"`
	LocalID               *int       `db:"local_id" json:"local_id,omitempty" example:"1"`
	ComboID               *int       `db:"combo_id" json:"combo_id,omitempty" example:"12"`
	ComboNombre           *string    `db:"combo_nombre" json:"combo_nombre,omitempty" example:"Combo Relax"`
	SesionesTotales       int        `db:"sesiones_totales" json:"sesiones_totales" example:"4"`
	SesionesUsadas        int        `db:"sesiones_usadas" json:"sesiones_usadas" example:"2"`
	CostoTotal            *float64   `db:"costo_total" json:"costo_total,omitempty" example:"700"`
	Notas                 *string    `db:"notas" json:"notas,omitempty" example:"Cliente frecuente"`
	Activo                bool       `db:"activo" json:"activo" example:"true"`
	CreadoEn              time.Time  `db:"creado_en" json:"creado_en" example:"2026-07-11T10:00:00Z"`
	ClienteID             *int       `db:"cliente_id" json:"cliente_id,omitempty" example:"12"`
	ClienteNombreTexto    string     `db:"cliente_nombre_texto" json:"cliente_nombre_texto" example:"Maria Lopez"`
	LocalNombreTexto      string     `db:"local_nombre_texto" json:"local_nombre_texto" example:"SAN MARTIN"`
	ComboIDOrigen         *int       `db:"combo_id_origen" json:"combo_id_origen,omitempty" example:"12"`
	ComboNombreTexto      *string    `db:"combo_nombre_texto" json:"combo_nombre_texto,omitempty" example:"Combo Relax"`
	FechaInicio           *time.Time `db:"fecha_inicio" json:"fecha_inicio,omitempty" example:"2026-07-15"`
	FechaFin              *time.Time `db:"fecha_fin" json:"fecha_fin,omitempty" example:"2026-08-14"`
	Estado                string     `db:"estado" json:"estado" example:"ACTIVO"`
	EstadoCobranza        string     `db:"estado_cobranza" json:"estado_cobranza" example:"PENDIENTE"`
	TipoPago              string     `db:"tipo_pago" json:"tipo_pago" example:"UNICO"`
	Subtotal              float64    `db:"subtotal" json:"subtotal" example:"800"`
	Descuento             float64    `db:"descuento" json:"descuento" example:"100"`
	PrecioTotal           float64    `db:"precio_total" json:"precio_total" example:"700"`
	Moneda                string     `db:"moneda" json:"moneda" example:"BOB"`
	CreadoPor             *int       `db:"creado_por" json:"creado_por,omitempty" example:"1"`
	ActualizadoPor        *int       `db:"actualizado_por" json:"actualizado_por,omitempty" example:"1"`
	ActualizadoEn         *time.Time `db:"actualizado_en" json:"actualizado_en,omitempty" example:"2026-07-11T12:00:00Z"`
}

type PlanServicioPG struct {
	ID                  int        `db:"id" json:"id" example:"35"`
	PlanID              int        `db:"plan_id" json:"plan_id" example:"21"`
	ServicioIDOrigen    *int       `db:"servicio_id_origen" json:"servicio_id_origen,omitempty" example:"8"`
	NombreTexto         string     `db:"nombre_texto" json:"nombre_texto" example:"Masaje relajante"`
	TiempoTexto         *string    `db:"tiempo_texto" json:"tiempo_texto,omitempty" example:"01:00"`
	PrecioUnitarioTexto *float64   `db:"precio_unitario_texto" json:"precio_unitario_texto,omitempty" example:"200"`
	SesionesContratadas int        `db:"sesiones_contratadas" json:"sesiones_contratadas" example:"2"`
	Orden               int        `db:"orden" json:"orden" example:"0"`
	SesionNumero        int        `db:"sesion_numero" json:"sesion_numero" example:"1"`
	Realizado           bool       `db:"realizado" json:"realizado" example:"false"`
	FechaRealizado      *time.Time `db:"fecha_realizado" json:"fecha_realizado,omitempty" example:"2026-07-12T15:04:05Z"`
}

type PlanMovimientoPG struct {
	ID             int       `db:"id" json:"id" example:"50"`
	PlanServicioID int       `db:"plan_servicio_id" json:"plan_servicio_id" example:"35"`
	ReservaID      *int      `db:"reserva_id" json:"reserva_id,omitempty" example:"100"`
	Tipo           string    `db:"tipo" json:"tipo" example:"RESERVA"`
	Cantidad       int       `db:"cantidad" json:"cantidad" example:"1"`
	Motivo         *string   `db:"motivo" json:"motivo,omitempty" example:"Ajuste por promocion"`
	UsuarioID      *int      `db:"usuario_id" json:"usuario_id,omitempty" example:"1"`
	CreadoEn       time.Time `db:"creado_en" json:"creado_en" example:"2026-07-11T10:00:00Z"`
}

type PlanCuotaPG struct {
	ID          int       `db:"id" json:"id" example:"40"`
	PlanID      int       `db:"plan_id" json:"plan_id" example:"21"`
	Numero      int       `db:"numero" json:"numero" example:"1"`
	Vencimiento *string   `db:"vencimiento" json:"vencimiento,omitempty" example:"2026-08-15"`
	Monto       float64   `db:"monto" json:"monto" example:"350"`
	Estado      string    `db:"estado" json:"estado" example:"PENDIENTE"`
	CreadoEn    time.Time `db:"creado_en" json:"creado_en" example:"2026-07-11T10:00:00Z"`
}

type PlanPagoAplicacionPG struct {
	ID            int       `db:"id" json:"id" example:"60"`
	PlanCuotaID   int       `db:"plan_cuota_id" json:"plan_cuota_id" example:"40"`
	PagoID        int       `db:"pago_id" json:"pago_id" example:"15"`
	MontoAplicado float64   `db:"monto_aplicado" json:"monto_aplicado" example:"350"`
	CreadoEn      time.Time `db:"creado_en" json:"creado_en" example:"2026-07-11T10:00:00Z"`
}

// PlanCompletoPG es un plan con sus lineas, cuotas y resumen de pagos.
type PlanCompletoPG struct {
	PlanPG
	Servicios []PlanServicioPG       `db:"-" json:"servicios"`
	Cuotas    []PlanCuotaPG          `db:"-" json:"cuotas"`
	Pagos     []PlanPagoAplicacionPG `db:"-" json:"pagos"`
}

// Reservas

type ReservaPG struct {
	ID                 int       `db:"id"`
	LocalID            *int      `db:"local_id"`
	LocalNombre        string    `db:"local_nombre"`
	TipoEspacio        string    `db:"tipo_espacio"`
	Fecha              time.Time `db:"fecha"`
	HoraDesde          string    `db:"hora_desde"` // TIME → string "09:00:00"
	HoraHasta          string    `db:"hora_hasta"`
	Cliente            string    `db:"cliente"`
	Estado             *string   `db:"estado"`
	NumeroTelefono     *string   `db:"numero_telefono"`
	PlanID             *int      `db:"plan_id"`
	ServicioNombre     *string   `db:"servicio_nombre"`
	ServicioSolicitado *string   `db:"servicio_solicitado"`
	ServicioConfirmado *string   `db:"servicio_confirmado"`
	ServicioTiempo     *string   `db:"servicio_tiempo"`
	Precio             *float64  `db:"precio"`
	Notas              *string   `db:"notas"`
	Activo             bool      `db:"activo"`
	Notificado         bool      `db:"notificado"`
	CreadoEn           time.Time `db:"creado_en"`
	ActualizadoEn      time.Time `db:"actualizado_en"`
}

type DetalleReservaPG struct {
	ID             int      `db:"id"`
	ReservaID      int      `db:"reserva_id"`
	ServicioNombre string   `db:"servicio_nombre"`
	ServicioTiempo *string  `db:"servicio_tiempo"`
	Precio         *float64 `db:"precio"`
	Sesiones       int      `db:"sesiones"`
	Notas          *string  `db:"notas"`
}

type ReservaPGCompleta struct {
	ReservaPG
	Detalle []DetalleReservaPG `db:"-"`
}

// Pagos

type PagoPG struct {
	// ID interno del pago; no se expone en la API.
	ID int `db:"id" json:"-"`
	// Codigo publico incremental del pago.
	CodigoPago string `db:"codigo_pago" json:"codigo_pago" example:"PAGO-000001"`
	// ID del local donde se registra el pago.
	LocalID int `db:"local_id" json:"local_id" example:"1"`
	// Nombre del local al momento de registrar el pago.
	LocalNombre string `db:"local_nombre" json:"local_nombre" example:"SAN MARTIN"`
	// ID opcional del cliente registrado.
	ClienteID *int `db:"cliente_id" json:"cliente_id,omitempty" example:"12"`
	// NIT opcional del cliente; vacio si no se registro.
	ClienteNIT string `db:"cliente_nit" json:"cliente_nit" example:"1234567"`
	// Nombre del cliente al momento de registrar el pago.
	ClienteNombre string `db:"cliente_nombre" json:"cliente_nombre" example:"Maria Lopez"`
	// Subtotal del pago antes del descuento.
	Subtotal float64 `db:"subtotal" json:"subtotal" example:"500"`
	// Descuento aplicado al pago.
	Descuento float64 `db:"descuento" json:"descuento" example:"50"`
	// Total final del pago.
	TotalFinal float64 `db:"total_final" json:"total_final" example:"450"`
	// Tipo de pago utilizado.
	TipoPago string `db:"tipo_pago" json:"tipo_pago" example:"efectivo"`
	// Estado informativo del pago.
	Estado string `db:"estado" json:"estado" example:"PENDIENTE"`
	// Estado activo para borrado logico.
	Activo bool `db:"activo" json:"activo" example:"true"`
	// ID opcional del cajero que registro el pago.
	IDCajero *int `db:"id_cajero" json:"id_cajero,omitempty" example:"1"`
	// Nombre completo del cajero que registro el pago.
	NombreCajero string `db:"nombre_cajero" json:"nombre_cajero" example:"admin"`
	// Username opcional del cajero que registro el pago.
	UsernameCajero *string `db:"username_cajero" json:"username_cajero,omitempty" example:"admin"`
	// ID opcional del cajero que modifico el pago por ultima vez.
	IDCajeroModificacion *int `db:"id_cajero_modificacion" json:"id_cajero_modificacion,omitempty" example:"2"`
	// Nombre completo opcional del cajero que modifico el pago por ultima vez.
	NombreCajeroModificacion *string `db:"nombre_cajero_modificacion" json:"nombre_cajero_modificacion,omitempty" example:"Ana Perez"`
	// Username opcional del cajero que modifico el pago por ultima vez.
	UsernameCajeroModificacion *string `db:"username_cajero_modificacion" json:"username_cajero_modificacion,omitempty" example:"ana"`
	// Fecha de creacion del pago.
	FechaCreacion time.Time `db:"fecha_creacion" json:"fecha_creacion" example:"2026-06-03T10:00:00Z"`
	// Fecha de ultima modificacion del pago.
	FechaModificacion time.Time `db:"fecha_modificacion" json:"fecha_modificacion" example:"2026-06-03T10:30:00Z"`
}

type DetallePagoPG struct {
	// ID del detalle de pago.
	ID int `db:"id" json:"id" example:"25"`
	// ID interno del pago; no se expone en la API.
	PagoID int `db:"pago_id" json:"-"`
	// ID opcional del servicio cobrado.
	ServicioID *int `db:"servicio_id" json:"servicio_id,omitempty" example:"8"`
	// Texto del servicio cobrado.
	Servicio string `db:"servicio" json:"servicio" example:"Limpieza facial"`
	// Precio unitario del servicio.
	PrecioUnitario float64 `db:"precio_unitario" json:"precio_unitario" example:"250"`
	// Cantidad cobrada.
	Cantidad int `db:"cantidad" json:"cantidad" example:"2"`
	// Subtotal de la linea.
	Subtotal float64 `db:"subtotal" json:"subtotal" example:"500"`
}

type PagoCompletoPG struct {
	PagoPG
	// Detalle de servicios cobrados.
	Detalle []DetallePagoPG `db:"-" json:"detalle"`
}
