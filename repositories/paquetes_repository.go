package repository

import (
	"context"
	"errors"

	"atrevida-agenda-api/models"
)

// ErrPaqueteNoEncontrado permite a las capas superiores distinguir una
// referencia inexistente de una falla inesperada de PostgreSQL.
var ErrPaqueteNoEncontrado = errors.New("paquete o referencia no encontrado")

// ErrPaqueteDatosInvalidos identifica inconsistencias de dominio detectadas
// al materializar los tiers de un paquete.
var ErrPaqueteDatosInvalidos = errors.New("datos de paquete invalidos")

type FiltroPaquetes struct {
	Context      context.Context
	Nombre       string
	Categoria    string
	Local        string
	LocalID      *int
	Activo       *bool
	PageLimit    int
	CursorSet    bool
	CursorNombre string
	CursorID     int
}

// PaqueteServicioInput es una linea del catalogo base de servicios de un
// paquete. ServicioID es solo trazabilidad; costo queda como snapshot.
type PaqueteServicioInput struct {
	ServicioID    *int
	ServicioTexto *string
	Costo         float64
	Orden         int
}

type CrearPaqueteInput struct {
	Nombre        string
	Descripcion   *string
	CategoriaID   *int
	Moneda        string
	LocalIDs      []int
	ServiciosBase []PaqueteServicioInput
	Tiers         []models.PaqueteTierInput
}

type ActualizarPaqueteInput struct {
	ID            int
	Nombre        string
	Descripcion   *string
	CategoriaID   *int
	Moneda        string
	LocalIDs      []int
	ServiciosBase []PaqueteServicioInput
	Tiers         []models.PaqueteTierInput
}

// PaquetesRepository persiste el catalogo de paquetes. Los tiers (precio por
// cantidad de sesiones) se materializan como filas de `combos` enlazadas por
// `combos.paquete_id`, de modo que planes/caja siguen operando sobre combos
// sin cambios.
type PaquetesRepository interface {
	ListPaquetes(f FiltroPaquetes) ([]models.PaqueteDetalle, error)
	CountPaquetes(f FiltroPaquetes) (int, error)
	GetPaqueteByID(id int, incluirInactivo bool) (*models.PaqueteDetalle, error)
	CrearPaquete(in CrearPaqueteInput) (int, error)
	ActualizarPaquete(in ActualizarPaqueteInput) error
	EliminarPaquete(id int) error
	// SetPaqueteImagen actualiza el path de portada (nil lo limpia).
	SetPaqueteImagen(id int, path *string) error
}
