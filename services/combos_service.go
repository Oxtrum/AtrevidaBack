package services

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

const (
	TipoPrecioPorItems   = "POR_ITEMS"
	TipoPrecioPorPaquete = "PRECIO_PAQUETE"
)

var (
	ErrComboInvalido               = errors.New("datos de combo invalidos")
	ErrComboNoEncontrado           = errors.New("combo no encontrado")
	ErrComboReferenciaNoEncontrada = errors.New("referencia de combo no encontrada")
	ErrComboServicioInvalido       = errors.New("servicio de combo invalido")
)

type FiltroCombos struct {
	Nombre    string
	Categoria string
	Local     string
	LocalID   *int
}

type CrearComboCatalogoInput struct {
	Nombre        string
	Descripcion   *string
	CategoriaID   *int
	TipoPrecio    string
	PrecioPaquete *float64
	Moneda        string
	DuracionMin   *int
	LocalIDs      []int
	Servicios     []repository.ComboServicioCatalogoInput
}

type ActualizarComboCatalogoInput struct {
	ID            int
	Nombre        *string
	Descripcion   *string
	CategoriaID   *int
	TipoPrecio    *string
	PrecioPaquete *float64
	Moneda        *string
	DuracionMin   *int
}

type CombosService struct {
	repo repository.CombosRepository
}

func NewCombosService(repo repository.CombosRepository) *CombosService {
	return &CombosService{repo: repo}
}

func (s *CombosService) ListarCombos(f FiltroCombos) ([]models.ComboCatalogoPG, int, error) {
	activo := true
	combos, total, err := s.repo.ListCombos(repository.FiltroCombos{
		Nombre: strings.TrimSpace(f.Nombre), Categoria: strings.TrimSpace(f.Categoria),
		Local: strings.TrimSpace(f.Local), LocalID: f.LocalID, Activo: &activo,
	})
	return combos, total, traducirErrorRepositorioCombo(err)
}

func (s *CombosService) ObtenerCombo(id int) (*models.ComboCatalogoPG, error) {
	if id < 1 {
		return nil, fmt.Errorf("id debe ser un entero positivo: %w", ErrComboInvalido)
	}
	combo, err := s.repo.GetComboByID(id, false)
	return combo, traducirErrorRepositorioCombo(err)
}

func (s *CombosService) CrearCombo(input CrearComboCatalogoInput) (int, error) {
	normalizado, err := normalizarCrearCombo(input)
	if err != nil {
		return 0, err
	}
	id, err := s.repo.CreateCombo(normalizado)
	return id, traducirErrorRepositorioCombo(err)
}

func (s *CombosService) ActualizarCombo(input ActualizarComboCatalogoInput) error {
	if input.ID < 1 {
		return fmt.Errorf("id debe ser un entero positivo: %w", ErrComboInvalido)
	}
	if input.Nombre == nil && input.Descripcion == nil && input.CategoriaID == nil && input.TipoPrecio == nil && input.PrecioPaquete == nil && input.Moneda == nil && input.DuracionMin == nil {
		return fmt.Errorf("debe especificarse al menos un campo a modificar: %w", ErrComboInvalido)
	}
	if input.DuracionMin != nil && *input.DuracionMin < 0 {
		return fmt.Errorf("duracion_min no puede ser negativa: %w", ErrComboInvalido)
	}
	if input.Nombre != nil {
		v := strings.TrimSpace(*input.Nombre)
		if v == "" {
			return fmt.Errorf("nombre es requerido: %w", ErrComboInvalido)
		}
		input.Nombre = &v
	}
	if input.Descripcion != nil {
		v := strings.TrimSpace(*input.Descripcion)
		input.Descripcion = &v
	}
	if input.CategoriaID != nil && *input.CategoriaID < 1 {
		return fmt.Errorf("categoria_id debe ser un entero positivo: %w", ErrComboInvalido)
	}
	if input.TipoPrecio != nil {
		v := strings.ToUpper(strings.TrimSpace(*input.TipoPrecio))
		if v != TipoPrecioPorItems && v != TipoPrecioPorPaquete {
			return fmt.Errorf("tipo_precio debe ser POR_ITEMS o PRECIO_PAQUETE: %w", ErrComboInvalido)
		}
		input.TipoPrecio = &v
	}
	if input.PrecioPaquete != nil && *input.PrecioPaquete < 0 {
		return fmt.Errorf("precio_paquete no puede ser negativo: %w", ErrComboInvalido)
	}
	if input.Moneda != nil {
		v := strings.ToUpper(strings.TrimSpace(*input.Moneda))
		if len(v) != 3 {
			return fmt.Errorf("moneda debe tener tres caracteres: %w", ErrComboInvalido)
		}
		input.Moneda = &v
	}
	return traducirErrorRepositorioCombo(s.repo.UpdateCombo(repository.ActualizarComboInput(input)))
}

func (s *CombosService) DesactivarCombo(id int) error {
	if id < 1 {
		return fmt.Errorf("id debe ser un entero positivo: %w", ErrComboInvalido)
	}
	return traducirErrorRepositorioCombo(s.repo.SetComboActivo(id, false))
}

func (s *CombosService) ReemplazarLocales(comboID int, localIDs []int) error {
	if comboID < 1 || len(localIDs) == 0 || contieneDuplicados(localIDs) {
		return fmt.Errorf("combo_id y locales validos son requeridos: %w", ErrComboInvalido)
	}
	for _, id := range localIDs {
		if id < 1 {
			return fmt.Errorf("local_id debe ser un entero positivo: %w", ErrComboInvalido)
		}
	}
	return traducirErrorRepositorioCombo(s.repo.SetComboLocales(comboID, localIDs))
}

func (s *CombosService) ReemplazarServicios(comboID int, servicios []repository.ComboServicioCatalogoInput) error {
	if comboID < 1 {
		return fmt.Errorf("combo_id debe ser un entero positivo: %w", ErrComboInvalido)
	}
	normalizados, err := normalizarServicios(servicios)
	if err != nil {
		return err
	}
	return traducirErrorRepositorioCombo(s.repo.ReplaceComboServicios(comboID, normalizados))
}

func traducirErrorRepositorioCombo(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrComboNoEncontrado) {
		return fmt.Errorf("%w: %v", ErrComboReferenciaNoEncontrada, err)
	}
	if errors.Is(err, repository.ErrComboDatosInvalidos) {
		return fmt.Errorf("%w: %v", ErrComboInvalido, err)
	}
	return err
}

func normalizarCrearCombo(input CrearComboCatalogoInput) (repository.CrearComboInput, error) {
	nombre := strings.TrimSpace(input.Nombre)
	if nombre == "" || len(input.LocalIDs) == 0 || contieneDuplicados(input.LocalIDs) {
		return repository.CrearComboInput{}, fmt.Errorf("nombre y al menos un local son requeridos: %w", ErrComboInvalido)
	}
	for _, id := range input.LocalIDs {
		if id < 1 {
			return repository.CrearComboInput{}, fmt.Errorf("local_id debe ser un entero positivo: %w", ErrComboInvalido)
		}
	}
	if input.CategoriaID != nil && *input.CategoriaID < 1 {
		return repository.CrearComboInput{}, fmt.Errorf("categoria_id debe ser un entero positivo: %w", ErrComboInvalido)
	}
	tipoPrecio := strings.ToUpper(strings.TrimSpace(input.TipoPrecio))
	if tipoPrecio != TipoPrecioPorItems && tipoPrecio != TipoPrecioPorPaquete {
		return repository.CrearComboInput{}, fmt.Errorf("tipo_precio debe ser POR_ITEMS o PRECIO_PAQUETE: %w", ErrComboInvalido)
	}
	if input.PrecioPaquete != nil && *input.PrecioPaquete < 0 {
		return repository.CrearComboInput{}, fmt.Errorf("precio_paquete no puede ser negativo: %w", ErrComboInvalido)
	}
	if tipoPrecio == TipoPrecioPorPaquete && input.PrecioPaquete == nil {
		return repository.CrearComboInput{}, fmt.Errorf("precio_paquete es requerido para PRECIO_PAQUETE: %w", ErrComboInvalido)
	}
	if tipoPrecio == TipoPrecioPorItems && input.PrecioPaquete != nil {
		return repository.CrearComboInput{}, fmt.Errorf("precio_paquete no aplica para POR_ITEMS: %w", ErrComboInvalido)
	}
	moneda := strings.ToUpper(strings.TrimSpace(input.Moneda))
	if moneda == "" {
		moneda = "BOB"
	}
	if len(moneda) != 3 {
		return repository.CrearComboInput{}, fmt.Errorf("moneda debe tener tres caracteres: %w", ErrComboInvalido)
	}
	if input.DuracionMin != nil && *input.DuracionMin < 0 {
		return repository.CrearComboInput{}, fmt.Errorf("duracion_min no puede ser negativa: %w", ErrComboInvalido)
	}
	servicios, err := normalizarServicios(input.Servicios)
	if err != nil {
		return repository.CrearComboInput{}, err
	}
	var descripcion *string
	if input.Descripcion != nil {
		v := strings.TrimSpace(*input.Descripcion)
		descripcion = &v
	}
	return repository.CrearComboInput{Nombre: nombre, Descripcion: descripcion, CategoriaID: input.CategoriaID, TipoPrecio: tipoPrecio, PrecioPaquete: input.PrecioPaquete, Moneda: moneda, DuracionMin: input.DuracionMin, LocalIDs: input.LocalIDs, Servicios: servicios}, nil
}

func normalizarServicios(servicios []repository.ComboServicioCatalogoInput) ([]repository.ComboServicioCatalogoInput, error) {
	if len(servicios) == 0 {
		return nil, fmt.Errorf("servicios es requerido: %w", ErrComboServicioInvalido)
	}
	ordenes := map[int]bool{}
	resultado := make([]repository.ComboServicioCatalogoInput, 0, len(servicios))
	for _, servicio := range servicios {
		servicio.ServicioTexto = strings.TrimSpace(servicio.ServicioTexto)
		if servicio.ServicioID != nil && *servicio.ServicioID < 1 {
			return nil, fmt.Errorf("servicio_id debe ser un entero positivo: %w", ErrComboServicioInvalido)
		}
		if servicio.ServicioID == nil && servicio.ServicioTexto == "" {
			return nil, fmt.Errorf("cada servicio requiere servicio_id o servicio_texto: %w", ErrComboServicioInvalido)
		}
		// Servicio como referencia: sin sesiones por línea, default 1 (constraint sesiones > 0).
		if servicio.Sesiones < 1 {
			servicio.Sesiones = 1
		}
		if servicio.SesionNumero < 1 {
			servicio.SesionNumero = 1
		}
		if servicio.Orden < 0 || ordenes[servicio.Orden] {
			return nil, fmt.Errorf("orden debe ser no negativo y unico: %w", ErrComboServicioInvalido)
		}
		if servicio.Costo != nil {
			if *servicio.Costo < 0 {
				return nil, fmt.Errorf("costo no puede ser negativo: %w", ErrComboServicioInvalido)
			}
			v := redondearImporte(*servicio.Costo)
			servicio.Costo = &v
		}
		if servicio.Tiempo != nil {
			v := strings.TrimSpace(*servicio.Tiempo)
			servicio.Tiempo = &v
		}
		ordenes[servicio.Orden] = true
		resultado = append(resultado, servicio)
	}
	return resultado, nil
}

func contieneDuplicados(ids []int) bool {
	vistos := map[int]bool{}
	for _, id := range ids {
		if vistos[id] {
			return true
		}
		vistos[id] = true
	}
	return false
}
func redondearImporte(v float64) float64 { return math.Round(v*100) / 100 }
