package services

import (
	"testing"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type fakeClientesRepo struct {
	crearInput      repository.CrearClienteInput
	actualizarInput repository.ActualizarClienteInput
}

func (f *fakeClientesRepo) GetClientes(filtro repository.FiltroClientes) ([]models.ClientePG, error) {
	return nil, nil
}

func (f *fakeClientesRepo) CountClientes(filtro repository.FiltroClientes) (int, error) {
	return 0, nil
}

func (f *fakeClientesRepo) GetClienteByID(id int) (*models.ClientePG, error) {
	return nil, nil
}

func (f *fakeClientesRepo) CreateCliente(input repository.CrearClienteInput) (int, error) {
	f.crearInput = input
	return 7, nil
}

func (f *fakeClientesRepo) UpdateCliente(input repository.ActualizarClienteInput) error {
	f.actualizarInput = input
	return nil
}

func (f *fakeClientesRepo) DeleteCliente(id int) error {
	return nil
}

func TestCreateClienteRecortaEspaciosDeCIyNIT(t *testing.T) {
	repo := &fakeClientesRepo{}
	service := NewClientesService(repo)

	id, err := service.CreateCliente(CrearClienteInput{
		Nombre:         "  Maria  ",
		Apellido:       " Lopez ",
		NumeroTelefono: " 70011223 ",
		CI:             "  8765432  ",
		NIT:            " 1234567 ",
	})
	if err != nil {
		t.Fatalf("CreateCliente() error = %v, want nil", err)
	}
	if id != 7 {
		t.Fatalf("CreateCliente() id = %d, want 7", id)
	}
	if repo.crearInput.CI != "8765432" {
		t.Errorf("CI = %q, want %q", repo.crearInput.CI, "8765432")
	}
	if repo.crearInput.NIT != "1234567" {
		t.Errorf("NIT = %q, want %q", repo.crearInput.NIT, "1234567")
	}
	if repo.crearInput.Nombre != "Maria" {
		t.Errorf("Nombre = %q, want %q", repo.crearInput.Nombre, "Maria")
	}
}

func TestUpdateClientePermiteVaciarElNIT(t *testing.T) {
	repo := &fakeClientesRepo{}
	service := NewClientesService(repo)

	vacio := ""
	if err := service.UpdateCliente(ActualizarClienteInput{ID: 3, NIT: &vacio}); err != nil {
		t.Fatalf("UpdateCliente() error = %v, want nil", err)
	}
	if repo.actualizarInput.NIT == nil {
		t.Fatal("NIT = nil, want puntero a cadena vacia")
	}
	if *repo.actualizarInput.NIT != "" {
		t.Errorf("NIT = %q, want cadena vacia", *repo.actualizarInput.NIT)
	}
	if repo.actualizarInput.CI != nil {
		t.Errorf("CI = %v, want nil (no se envio)", repo.actualizarInput.CI)
	}
}
