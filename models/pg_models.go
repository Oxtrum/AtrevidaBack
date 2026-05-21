package models

import (
	"time"
)

// Locales

type LocalPG struct {
	ID     int    `db:"id"`
	Nombre string `db:"nombre"`
	Activo bool   `db:"activo"`
}
type TipoEspacioLocal struct {
	TipoEspacio      string `db:"tipo_espacio"       json:"tipo_espacio"`
	CantidadEspacios int    `db:"cantidad_espacios"  json:"cantidad_espacios"`
}

type LocalConEspacios struct {
	ID       int                `db:"id"     json:"id"`
	Nombre   string             `db:"nombre" json:"nombre"`
	Activo   bool               `db:"activo" json:"activo"`
	Espacios []TipoEspacioLocal `db:"-"     json:"espacios"`
}

// Categorías

type CategoriaPG struct {
	ID     int    `db:"id"`
	Nombre string `db:"nombre"`
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
	ID             int      `db:"id"`
	ComboID        int      `db:"combo_id"`
	ServicioID     int      `db:"servicio_id"`
	Tiempo         *string  `db:"tiempo"`
	Costo          *float64 `db:"costo"`
	Sesiones       int      `db:"sesiones"`
	Orden          int      `db:"orden"`
	ServicioNombre string   `db:"servicio_nombre"`
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
