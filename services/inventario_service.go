package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

var unidadesMedida = map[string]bool{"UNIDAD": true, "ML": true, "GR": true, "CAJA": true}

type InventarioService struct {
	repo   *pgsqlrepo.InventarioRepo
	config *ConfiguracionService
	// Storage es opcional: si es nil, la gestion de imagenes queda deshabilitada.
	Storage ObjectStorage
}

func NewInventarioService(repo *pgsqlrepo.InventarioRepo, config *ConfiguracionService) *InventarioService {
	return &InventarioService{repo: repo, config: config}
}

func rutaImagenItem(id int) string { return fmt.Sprintf("inventario/%d/imagen", id) }

func (s *InventarioService) completar(item *models.InventarioItem) {
	item.BajoStock = item.StockActual <= item.StockMinimo
	if s.Storage != nil && item.ImagenKey != nil && *item.ImagenKey != "" {
		url := s.Storage.URLPublica(*item.ImagenKey)
		item.ImagenURL = &url
	}
}

// ─── Items ────────────────────────────────────────────────────────────────────

func (s *InventarioService) ListarItems(f repository.FiltroItemsInventario) ([]models.InventarioItem, error) {
	items, err := s.repo.ListItems(f)
	if err != nil {
		return nil, err
	}
	for i := range items {
		s.completar(&items[i])
	}
	return items, nil
}

func (s *InventarioService) ObtenerItem(id int) (*models.InventarioItem, error) {
	item, err := s.repo.GetItem(id)
	if err != nil {
		return nil, err
	}
	s.completar(item)
	return item, nil
}

// ServiciosQueUsan devuelve los servicios que consumen el item (para el detalle).
func (s *InventarioService) ServiciosQueUsan(itemID int) ([]string, error) {
	return s.repo.ServiciosQueUsanItem(itemID)
}

func validarItem(nombre string, unidad string, costo *float64, minimo float64) error {
	if strings.TrimSpace(nombre) == "" {
		return errors.New("nombre es requerido")
	}
	if len(nombre) > 200 {
		return errors.New("nombre no puede superar 200 caracteres")
	}
	if !unidadesMedida[unidad] {
		return errors.New("unidad_medida invalida, valores permitidos: UNIDAD, ML, GR, CAJA")
	}
	if costo != nil && *costo < 0 {
		return errors.New("costo_referencia no puede ser negativo")
	}
	if minimo < 0 {
		return errors.New("stock_minimo no puede ser negativo")
	}
	return nil
}

func (s *InventarioService) CrearItem(in pgsqlrepo.ItemInput) (*models.InventarioItem, error) {
	in.Nombre = strings.TrimSpace(in.Nombre)
	in.UnidadMedida = strings.ToUpper(strings.TrimSpace(in.UnidadMedida))
	if in.UnidadMedida == "" {
		in.UnidadMedida = "UNIDAD"
	}
	if err := validarItem(in.Nombre, in.UnidadMedida, in.CostoReferencia, in.StockMinimo); err != nil {
		return nil, err
	}
	id, err := s.repo.CreateItem(in)
	if err != nil {
		return nil, err
	}
	return s.ObtenerItem(id)
}

func (s *InventarioService) ActualizarItem(id int, p pgsqlrepo.ItemPatch) error {
	if p.Nombre != nil {
		n := strings.TrimSpace(*p.Nombre)
		if n == "" || len(n) > 200 {
			return errors.New("nombre es requerido y no puede superar 200 caracteres")
		}
		p.Nombre = &n
	}
	if p.UnidadMedida != nil {
		u := strings.ToUpper(strings.TrimSpace(*p.UnidadMedida))
		if !unidadesMedida[u] {
			return errors.New("unidad_medida invalida, valores permitidos: UNIDAD, ML, GR, CAJA")
		}
		p.UnidadMedida = &u
	}
	if p.CostoReferenciaSet && p.CostoReferencia != nil && *p.CostoReferencia < 0 {
		return errors.New("costo_referencia no puede ser negativo")
	}
	if p.StockMinimo != nil && *p.StockMinimo < 0 {
		return errors.New("stock_minimo no puede ser negativo")
	}
	if p.Nombre == nil && p.UnidadMedida == nil && p.StockMinimo == nil && p.Activo == nil &&
		!p.DescripcionSet && !p.CategoriaSet && !p.CostoReferenciaSet {
		return errors.New("no se enviaron campos para actualizar")
	}
	return s.repo.UpdateItem(id, p)
}

// DesactivarItem lo oculta del catalogo sin borrar historial (RN-07).
func (s *InventarioService) DesactivarItem(id int) error {
	inactivo := false
	return s.repo.UpdateItem(id, pgsqlrepo.ItemPatch{Activo: &inactivo})
}

// ─── Imagen ───────────────────────────────────────────────────────────────────

func (s *InventarioService) GenerarURLSubidaImagen(id int) (SubidaFirmada, error) {
	if ok, err := s.repo.ItemExiste(id); err != nil {
		return SubidaFirmada{}, err
	} else if !ok {
		return SubidaFirmada{}, repository.ErrItemNoEncontrado
	}
	if s.Storage == nil {
		return SubidaFirmada{}, ErrAlmacenamientoNoConfigurado
	}
	return s.Storage.CrearURLSubida(rutaImagenItem(id))
}

// ConfirmarImagen guarda la clave tras una subida exitosa y devuelve la URL.
func (s *InventarioService) ConfirmarImagen(id int) (string, error) {
	if s.Storage == nil {
		return "", ErrAlmacenamientoNoConfigurado
	}
	key := rutaImagenItem(id)
	if err := s.repo.SetImagenKey(id, &key); err != nil {
		return "", err
	}
	return s.Storage.URLPublica(key), nil
}

func (s *InventarioService) EliminarImagen(id int) error {
	if s.Storage == nil {
		return ErrAlmacenamientoNoConfigurado
	}
	if err := s.Storage.Eliminar(rutaImagenItem(id)); err != nil {
		return err
	}
	return s.repo.SetImagenKey(id, nil)
}

// ─── Categorias ───────────────────────────────────────────────────────────────

func (s *InventarioService) ListarCategorias() ([]models.CategoriaInventario, error) {
	return s.repo.ListCategorias()
}

func (s *InventarioService) CrearCategoria(nombre string) (int, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" || len(nombre) > 100 {
		return 0, errors.New("nombre es requerido y no puede superar 100 caracteres")
	}
	return s.repo.CreateCategoria(nombre)
}

func (s *InventarioService) ActualizarCategoria(id int, nombre *string, activo *bool) error {
	if nombre == nil && activo == nil {
		return errors.New("no se enviaron campos para actualizar")
	}
	if nombre != nil {
		n := strings.TrimSpace(*nombre)
		if n == "" || len(n) > 100 {
			return errors.New("nombre es requerido y no puede superar 100 caracteres")
		}
		nombre = &n
	}
	return s.repo.UpdateCategoria(id, nombre, activo)
}

// ─── Kardex y ajustes ─────────────────────────────────────────────────────────

func (s *InventarioService) ListarMovimientos(f repository.FiltroMovimientosInventario) ([]models.MovimientoInventario, error) {
	if f.Tipo != "" {
		switch f.Tipo {
		case repository.MovEntradaCompra, repository.MovSalidaServicio, repository.MovReversionServicio,
			repository.MovAjustePositivo, repository.MovAjusteNegativo:
		default:
			return nil, errors.New("tipo de movimiento invalido")
		}
	}
	return s.repo.ListMovimientos(f)
}

type AjusteInput struct {
	ItemID        int
	LocalID       int
	Tipo          string
	Cantidad      float64
	Motivo        string
	UsuarioID     *int
	UsuarioNombre string
}

// Ajustar registra una merma, vencimiento o conteo fisico. El motivo es obligatorio (RN-03).
func (s *InventarioService) Ajustar(in AjusteInput) (float64, error) {
	if in.Tipo != repository.MovAjustePositivo && in.Tipo != repository.MovAjusteNegativo {
		return 0, errors.New("tipo invalido, valores permitidos: AJUSTE_POSITIVO, AJUSTE_NEGATIVO")
	}
	if in.Cantidad <= 0 {
		return 0, errors.New("cantidad debe ser mayor a 0")
	}
	if strings.TrimSpace(in.Motivo) == "" {
		return 0, errors.New("motivo es requerido para ajustes de stock")
	}
	if in.ItemID < 1 || in.LocalID < 1 {
		return 0, errors.New("item_id y local_id son requeridos")
	}
	return s.repo.Ajustar(repository.MovimientoStockInput{
		ItemID: in.ItemID, LocalID: in.LocalID, Tipo: in.Tipo, Cantidad: in.Cantidad,
		Motivo: strings.TrimSpace(in.Motivo), UsuarioID: in.UsuarioID, UsuarioNombre: in.UsuarioNombre,
		// Un ajuste manual nunca deja el stock negativo: para eso esta el descuento por servicio.
		PermitirNegativo: false,
	})
}

// ─── Alertas ──────────────────────────────────────────────────────────────────

func (s *InventarioService) Alertas(localID *int) (*models.AlertasInventario, error) {
	items, err := s.ListarItems(repository.FiltroItemsInventario{LocalID: localID, BajoStock: true, Activo: boolPtr(true)})
	if err != nil {
		return nil, err
	}
	reservas, err := s.repo.ReservasConAlerta(localID)
	if err != nil {
		return nil, err
	}
	cfg, err := s.config.Snapshot()
	if err != nil {
		return nil, err
	}
	for i := range reservas {
		// Se recalcula el detalle en vivo: el flag solo indica que hubo faltante al agendar.
		warns, err := s.advertenciasReserva(reservas[i].ReservaID, cfg.ConsiderarComprometido)
		if err == nil {
			reservas[i].Advertencias = warns
		}
	}
	return &models.AlertasInventario{ItemsBajoStock: items, ReservasConAlerta: reservas}, nil
}

func (s *InventarioService) advertenciasReserva(reservaID int, comprometido bool) ([]models.AdvertenciaInventario, error) {
	return s.repo.RecalcularAlerta(reservaID, comprometido)
}

// ─── Insumos por servicio ─────────────────────────────────────────────────────

func (s *InventarioService) ListarInsumos(servicioID int) ([]models.ServicioInsumo, error) {
	return s.repo.ListInsumos(servicioID)
}

// ReemplazarInsumos guarda la lista completa de insumos del servicio (PUT).
func (s *InventarioService) ReemplazarInsumos(servicioID int, insumos []models.ServicioInsumo) error {
	if ok, err := s.repo.ServicioExiste(servicioID); err != nil {
		return err
	} else if !ok {
		return errors.New("servicio no encontrado")
	}
	vistos := map[int]bool{}
	for _, in := range insumos {
		if in.ItemID < 1 {
			return errors.New("item_id invalido en insumos")
		}
		if in.CantidadPorSesion <= 0 {
			return errors.New("cantidad_por_sesion debe ser mayor a 0")
		}
		if vistos[in.ItemID] {
			return fmt.Errorf("el item %d esta repetido en los insumos", in.ItemID)
		}
		vistos[in.ItemID] = true
		if ok, err := s.repo.ItemExiste(in.ItemID); err != nil {
			return err
		} else if !ok {
			return fmt.Errorf("%w: id %d", repository.ErrItemNoEncontrado, in.ItemID)
		}
	}
	return s.repo.ReplaceInsumos(servicioID, insumos)
}

func boolPtr(b bool) *bool { return &b }

// ParseFechaOpcional parsea YYYY-MM-DD; vacio devuelve nil.
func ParseFechaOpcional(valor, campo string) (*time.Time, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", valor)
	if err != nil {
		return nil, fmt.Errorf("%s invalida, use formato YYYY-MM-DD", campo)
	}
	return &t, nil
}
