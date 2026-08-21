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
	if repo.crearInput.TelefonoE164 == nil || *repo.crearInput.TelefonoE164 != "+59170011223" {
		t.Errorf("TelefonoE164 = %v, want +59170011223", repo.crearInput.TelefonoE164)
	}
}

func TestCreateClienteAceptaE164Explicito(t *testing.T) {
	repo := &fakeClientesRepo{}
	service := NewClientesService(repo)
	e164 := "+5491123456789"
	if _, err := service.CreateCliente(CrearClienteInput{
		Nombre: "Ana", Apellido: "Perez", NumeroTelefono: "1123456789", TelefonoE164: &e164,
	}); err != nil {
		t.Fatalf("CreateCliente() error = %v", err)
	}
	if repo.crearInput.TelefonoE164 == nil || *repo.crearInput.TelefonoE164 != e164 {
		t.Fatalf("TelefonoE164 = %v, want %q", repo.crearInput.TelefonoE164, e164)
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

func TestUpdateClienteLegacyNoNormalizableLimpiaE164(t *testing.T) {
	repo := &fakeClientesRepo{}
	service := NewClientesService(repo)
	legacy := "1234567"
	if err := service.UpdateCliente(ActualizarClienteInput{ID: 3, NumeroTelefono: &legacy}); err != nil {
		t.Fatalf("UpdateCliente() error = %v", err)
	}
	if !repo.actualizarInput.TelefonoE164Set {
		t.Fatal("TelefonoE164Set = false, want true")
	}
	if repo.actualizarInput.TelefonoE164 != nil {
		t.Fatalf("TelefonoE164 = %v, want nil", *repo.actualizarInput.TelefonoE164)
	}
}
