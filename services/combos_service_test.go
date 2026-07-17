package services

import (
	"errors"
	"testing"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type fakeCombosRepo struct {
	createInput   repository.CrearComboInput
	createErr     error
	servicesInput []repository.ComboServicioCatalogoInput
}

func (f *fakeCombosRepo) ListCombos(repository.FiltroCombos) ([]models.ComboCatalogoPG, int, error) {
	return nil, 0, nil
}

func (f *fakeCombosRepo) GetComboByID(int, bool) (*models.ComboCatalogoPG, error) {
	return nil, nil
}

func (f *fakeCombosRepo) CreateCombo(input repository.CrearComboInput) (int, error) {
	f.createInput = input
	return 1, f.createErr
}

func (f *fakeCombosRepo) UpdateCombo(repository.ActualizarComboInput) error { return nil }
func (f *fakeCombosRepo) SetComboActivo(int, bool) error                    { return nil }
func (f *fakeCombosRepo) SetComboLocales(int, []int) error                  { return nil }
func (f *fakeCombosRepo) SetComboImagen(int, *string) error                 { return nil }

func (f *fakeCombosRepo) ReplaceComboServicios(_ int, input []repository.ComboServicioCatalogoInput) error {
	f.servicesInput = input
	return nil
}

func TestCrearComboPorItemsPropagaCostoFaltanteComoErrorDeValidacion(t *testing.T) {
	service := NewCombosService(&fakeCombosRepo{createErr: repository.ErrComboDatosInvalidos})
	_, err := service.CrearCombo(CrearComboCatalogoInput{
		Nombre: "Promo", TipoPrecio: TipoPrecioPorItems, LocalIDs: []int{1},
		Servicios: []repository.ComboServicioCatalogoInput{{ServicioTexto: "Manual", Sesiones: 1, Orden: 0}},
	})
	if !errors.Is(err, ErrComboInvalido) {
		t.Fatalf("CrearCombo() error = %v, want ErrComboInvalido", err)
	}
}

func TestCrearComboPorItemsPermiteMaterializarCostoDesdeServicioOrigen(t *testing.T) {
	servicioID := 8
	repo := &fakeCombosRepo{}
	service := NewCombosService(repo)
	_, err := service.CrearCombo(CrearComboCatalogoInput{
		Nombre: "Promo", TipoPrecio: TipoPrecioPorItems, LocalIDs: []int{1},
		Servicios: []repository.ComboServicioCatalogoInput{{ServicioID: &servicioID, Sesiones: 1, Orden: 0}},
	})
	if err != nil {
		t.Fatalf("CrearCombo() error = %v", err)
	}
	if repo.createInput.Servicios[0].Costo != nil {
		t.Fatalf("el servicio debe llegar sin costo para materializarlo desde el origen")
	}
}

func TestCrearComboPrecioPaqueteNoAceptaPrecioDeItemsContradictorio(t *testing.T) {
	precio := 100.0
	service := NewCombosService(&fakeCombosRepo{})
	_, err := service.CrearCombo(CrearComboCatalogoInput{
		Nombre: "Promo", TipoPrecio: TipoPrecioPorItems, PrecioPaquete: &precio, LocalIDs: []int{1},
		Servicios: []repository.ComboServicioCatalogoInput{{ServicioTexto: "Manual", Costo: &precio, Sesiones: 1, Orden: 0}},
	})
	if !errors.Is(err, ErrComboInvalido) {
		t.Fatalf("CrearCombo() error = %v, want ErrComboInvalido", err)
	}
}

func TestReemplazarServiciosNormalizaCostoYEvitaOrdenDuplicado(t *testing.T) {
	precio := 10.129
	repo := &fakeCombosRepo{}
	service := NewCombosService(repo)
	err := service.ReemplazarServicios(1, []repository.ComboServicioCatalogoInput{{
		ServicioTexto: "Manual", Costo: &precio, Sesiones: 1, Orden: 0,
	}})
	if err != nil {
		t.Fatalf("ReemplazarServicios() error = %v", err)
	}
	if got := *repo.servicesInput[0].Costo; got != 10.13 {
		t.Fatalf("costo normalizado = %v, want 10.13", got)
	}
	err = service.ReemplazarServicios(1, []repository.ComboServicioCatalogoInput{
		{ServicioTexto: "A", Sesiones: 1, Orden: 0},
		{ServicioTexto: "B", Sesiones: 1, Orden: 0},
	})
	if !errors.Is(err, ErrComboServicioInvalido) {
		t.Fatalf("ReemplazarServicios() error = %v, want ErrComboServicioInvalido", err)
	}
}
