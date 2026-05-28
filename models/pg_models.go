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
	ID       int    `db:"id" json:"id" example:"1"`
	Username string `db:"username" json:"username" example:"admin"`
	Password string `db:"password" json:"-"`
	Activo   bool   `db:"activo" json:"activo" example:"true"`
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
	Orden          int      `db:"orden" json:"orden" example:"1"`
}

// Planes

type PlanPG struct {
	ID              int       `db:"id"`
	Cliente         string    `db:"cliente"`
	LocalID         *int      `db:"local_id"`
	ComboID         *int      `db:"combo_id"`
	ComboNombre     *string   `db:"combo_nombre"`
	SesionesTotales int       `db:"sesiones_totales"`
	SesionesUsadas  int       `db:"sesiones_usadas"`
	CostoTotal      *float64  `db:"costo_total"`
	Notas           *string   `db:"notas"`
	Activo          bool      `db:"activo"`
	CreadoEn        time.Time `db:"creado_en"`
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
