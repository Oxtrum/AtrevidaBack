package repository

import (
	"errors"

	"atrevida-agenda-api/models"
)

// ErrComboNoEncontrado permite a las capas superiores distinguir una referencia
// inexistente de una falla inesperada de PostgreSQL.
var ErrComboNoEncontrado = errors.New("combo o referencia no encontrado")

// ErrComboDatosInvalidos identifica inconsistencias de dominio detectadas una
// vez materializados los snapshots dentro de una transacción.
var ErrComboDatosInvalidos = errors.New("datos de combo invalidos")

type FiltroCombos struct {
	Nombre    string
	Categoria string
	Local     string
	LocalID   *int
	Activo    *bool
}

// ComboServicioCatalogoInput es una linea del catalogo. ServicioID es solo
// trazabilidad; los campos de texto, tiempo y costo quedan como snapshot.
type ComboServicioCatalogoInput struct {
	ServicioID    *int
	ServicioTexto string
	Tiempo        *string
	Costo         *float64
	Sesiones      int
	SesionNumero  int
	Orden         int
}

type CrearComboInput struct {
	Nombre        string
	Descripcion   *string
	CategoriaID   *int
	TipoPrecio    string
	PrecioPaquete *float64
	Moneda        string
	DuracionMin   *int
	LocalIDs      []int
	Servicios     []ComboServicioCatalogoInput
}

type ActualizarComboInput struct {
	ID            int
	Nombre        *string
	Descripcion   *string
	CategoriaID   *int
	TipoPrecio    *string
	PrecioPaquete *float64
	Moneda        *string
	DuracionMin   *int
}

type CombosRepository interface {
	ListCombos(filtro FiltroCombos) ([]models.ComboCatalogoPG, int, error)
	GetComboByID(id int, incluirInactivo bool) (*models.ComboCatalogoPG, error)
	CreateCombo(input CrearComboInput) (int, error)
	UpdateCombo(input ActualizarComboInput) error
	SetComboActivo(id int, activo bool) error
	SetComboLocales(comboID int, localIDs []int) error
	ReplaceComboServicios(comboID int, servicios []ComboServicioCatalogoInput) error
}
