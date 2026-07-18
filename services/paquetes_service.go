package services

import (
	"errors"
	"fmt"
	"strings"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

var (
	ErrPaqueteInvalido     = errors.New("datos de paquete invalidos")
	ErrPaqueteNoEncontrado = errors.New("paquete no encontrado")
)

// CrearPaqueteInput es el payload de alta de un paquete, tal como lo arma el
// handler a partir del request. Se normaliza y valida antes de delegar al
// repositorio.
type CrearPaqueteInput struct {
	Nombre        string
	Descripcion   *string
	CategoriaID   *int
	Moneda        string
	LocalIDs      []int
	ServiciosBase []repository.PaqueteServicioInput
	Tiers         []models.PaqueteTierInput
}

// ActualizarPaqueteInput es el payload de edicion de un paquete.
type ActualizarPaqueteInput struct {
	ID            int
	Nombre        string
	Descripcion   *string
	CategoriaID   *int
	Moneda        string
	LocalIDs      []int
	ServiciosBase []repository.PaqueteServicioInput
	Tiers         []models.PaqueteTierInput
}

type PaquetesService struct {
	repo repository.PaquetesRepository
	// Storage es opcional: si es nil, la gestion de imagenes queda deshabilitada
	// y las respuestas simplemente no incluyen imagen_url. Se inyecta en app.go.
	Storage *SupabaseStorage
}

func NewPaquetesService(repo repository.PaquetesRepository) *PaquetesService {
	return &PaquetesService{repo: repo}
}

// rutaImagenPaquete es el path determinista del objeto en el bucket. Reemplazar
// la imagen sobreescribe el mismo path, evitando objetos huerfanos.
func rutaImagenPaquete(paqueteID int) string {
	return fmt.Sprintf("paquetes/%d", paqueteID)
}

// rellenarImagenURLPaquete deriva la URL publica desde el path guardado.
func (s *PaquetesService) rellenarImagenURLPaquete(detalle *models.PaqueteDetalle) {
	if detalle == nil || s.Storage == nil || detalle.Paquete.ImagenPath == nil || *detalle.Paquete.ImagenPath == "" {
		return
	}
	url := s.Storage.URLPublica(*detalle.Paquete.ImagenPath)
	detalle.Paquete.ImagenURL = &url
}

// GenerarURLSubidaImagen valida el paquete y emite una URL firmada de subida.
func (s *PaquetesService) GenerarURLSubidaImagen(paqueteID int) (SubidaFirmada, error) {
	if paqueteID < 1 {
		return SubidaFirmada{}, fmt.Errorf("id debe ser un entero positivo: %w", ErrPaqueteInvalido)
	}
	if s.Storage == nil {
		return SubidaFirmada{}, ErrAlmacenamientoNoConfigurado
	}
	if _, err := s.repo.GetPaqueteByID(paqueteID, true); err != nil {
		return SubidaFirmada{}, err
	}
	return s.Storage.CrearURLSubida(rutaImagenPaquete(paqueteID))
}

// ConfirmarImagen persiste el path tras una subida exitosa y devuelve la URL publica.
func (s *PaquetesService) ConfirmarImagen(paqueteID int) (string, error) {
	if paqueteID < 1 {
		return "", fmt.Errorf("id debe ser un entero positivo: %w", ErrPaqueteInvalido)
	}
	if s.Storage == nil {
		return "", ErrAlmacenamientoNoConfigurado
	}
	path := rutaImagenPaquete(paqueteID)
	if err := s.repo.SetPaqueteImagen(paqueteID, &path); err != nil {
		return "", err
	}
	return s.Storage.URLPublica(path), nil
}

// EliminarImagen borra el objeto del bucket y limpia el path del paquete.
func (s *PaquetesService) EliminarImagen(paqueteID int) error {
	if paqueteID < 1 {
		return fmt.Errorf("id debe ser un entero positivo: %w", ErrPaqueteInvalido)
	}
	if s.Storage == nil {
		return ErrAlmacenamientoNoConfigurado
	}
	if err := s.Storage.Eliminar(rutaImagenPaquete(paqueteID)); err != nil {
		return err
	}
	return s.repo.SetPaqueteImagen(paqueteID, nil)
}

func (s *PaquetesService) Listar(f repository.FiltroPaquetes) ([]models.PaqueteDetalle, error) {
	paquetes, err := s.repo.ListPaquetes(f)
	if err != nil {
		return nil, err
	}
	for i := range paquetes {
		s.rellenarImagenURLPaquete(&paquetes[i])
	}
	return paquetes, nil
}

func (s *PaquetesService) Obtener(id int) (*models.PaqueteDetalle, error) {
	if id < 1 {
		return nil, fmt.Errorf("id debe ser un entero positivo: %w", ErrPaqueteInvalido)
	}
	paquete, err := s.repo.GetPaqueteByID(id, false)
	if err != nil {
		return nil, err
	}
	s.rellenarImagenURLPaquete(paquete)
	return paquete, nil
}

func (s *PaquetesService) Crear(input CrearPaqueteInput) (int, error) {
	normalizado, err := normalizarCrearPaquete(input)
	if err != nil {
		return 0, err
	}
	return s.repo.CrearPaquete(normalizado)
}

func (s *PaquetesService) Actualizar(input ActualizarPaqueteInput) error {
	if input.ID < 1 {
		return fmt.Errorf("id debe ser un entero positivo: %w", ErrPaqueteInvalido)
	}
	normalizado, err := normalizarPaquete(input.Nombre, input.LocalIDs, input.Tiers, input.Moneda)
	if err != nil {
		return err
	}
	input.Nombre = normalizado.nombre
	input.Moneda = normalizado.moneda
	if input.Descripcion != nil {
		v := strings.TrimSpace(*input.Descripcion)
		input.Descripcion = &v
	}
	return s.repo.ActualizarPaquete(repository.ActualizarPaqueteInput{
		ID:            input.ID,
		Nombre:        input.Nombre,
		Descripcion:   input.Descripcion,
		CategoriaID:   input.CategoriaID,
		Moneda:        input.Moneda,
		LocalIDs:      input.LocalIDs,
		ServiciosBase: input.ServiciosBase,
		Tiers:         input.Tiers,
	})
}

func (s *PaquetesService) Eliminar(id int) error {
	if id < 1 {
		return fmt.Errorf("id debe ser un entero positivo: %w", ErrPaqueteInvalido)
	}
	return s.repo.EliminarPaquete(id)
}

type paqueteNormalizado struct {
	nombre string
	moneda string
}

// normalizarPaquete valida los campos comunes a alta y edicion: nombre,
// locales, tiers y moneda. Devuelve el nombre y la moneda ya normalizados
// (trim/upper, con default de moneda) para que el llamador los reasigne.
func normalizarPaquete(nombre string, localIDs []int, tiers []models.PaqueteTierInput, moneda string) (paqueteNormalizado, error) {
	nombreLimpio := strings.TrimSpace(nombre)
	if nombreLimpio == "" {
		return paqueteNormalizado{}, fmt.Errorf("nombre es requerido: %w", ErrPaqueteInvalido)
	}
	if len(localIDs) == 0 || contieneDuplicados(localIDs) {
		return paqueteNormalizado{}, fmt.Errorf("al menos un local valido es requerido: %w", ErrPaqueteInvalido)
	}
	for _, id := range localIDs {
		if id < 1 {
			return paqueteNormalizado{}, fmt.Errorf("local_id debe ser un entero positivo: %w", ErrPaqueteInvalido)
		}
	}
	if len(tiers) == 0 {
		return paqueteNormalizado{}, fmt.Errorf("al menos un tier es requerido: %w", ErrPaqueteInvalido)
	}
	for _, tier := range tiers {
		if tier.Sesiones <= 0 {
			return paqueteNormalizado{}, fmt.Errorf("sesiones debe ser mayor a cero: %w", ErrPaqueteInvalido)
		}
		if tier.PrecioContado < 0 {
			return paqueteNormalizado{}, fmt.Errorf("precio_contado no puede ser negativo: %w", ErrPaqueteInvalido)
		}
		if tier.PrecioRegular != nil && *tier.PrecioRegular <= tier.PrecioContado {
			return paqueteNormalizado{}, fmt.Errorf("precio_regular debe ser mayor a precio_contado: %w", ErrPaqueteInvalido)
		}
	}
	monedaLimpia := strings.ToUpper(strings.TrimSpace(moneda))
	if monedaLimpia == "" {
		monedaLimpia = "BOB"
	}
	if len(monedaLimpia) != 3 {
		return paqueteNormalizado{}, fmt.Errorf("moneda debe tener tres caracteres: %w", ErrPaqueteInvalido)
	}
	return paqueteNormalizado{nombre: nombreLimpio, moneda: monedaLimpia}, nil
}

func normalizarCrearPaquete(input CrearPaqueteInput) (repository.CrearPaqueteInput, error) {
	normalizado, err := normalizarPaquete(input.Nombre, input.LocalIDs, input.Tiers, input.Moneda)
	if err != nil {
		return repository.CrearPaqueteInput{}, err
	}
	var descripcion *string
	if input.Descripcion != nil {
		v := strings.TrimSpace(*input.Descripcion)
		descripcion = &v
	}
	return repository.CrearPaqueteInput{
		Nombre:        normalizado.nombre,
		Descripcion:   descripcion,
		CategoriaID:   input.CategoriaID,
		Moneda:        normalizado.moneda,
		LocalIDs:      input.LocalIDs,
		ServiciosBase: input.ServiciosBase,
		Tiers:         input.Tiers,
	}, nil
}
