package services

import (
	"errors"
	"testing"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type fakePaquetesRepo struct {
	createInput repository.CrearPaqueteInput
	createErr   error
}

func (f *fakePaquetesRepo) ListPaquetes(repository.FiltroPaquetes) ([]models.PaqueteDetalle, error) {
	return nil, nil
}

func (f *fakePaquetesRepo) GetPaqueteByID(int, bool) (*models.PaqueteDetalle, error) {
	return nil, nil
}

func (f *fakePaquetesRepo) CrearPaquete(in repository.CrearPaqueteInput) (int, error) {
	f.createInput = in
	return 1, f.createErr
}

func (f *fakePaquetesRepo) ActualizarPaquete(repository.ActualizarPaqueteInput) error { return nil }
func (f *fakePaquetesRepo) EliminarPaquete(int) error                                 { return nil }
func (f *fakePaquetesRepo) SetPaqueteImagen(int, *string) error                       { return nil }

func TestCrearPaquete_ValidaTierSesiones(t *testing.T) {
	svc := NewPaquetesService(&fakePaquetesRepo{})
	_, err := svc.Crear(CrearPaqueteInput{
		Nombre: "X", LocalIDs: []int{1},
		Tiers: []models.PaqueteTierInput{{Sesiones: 0, PrecioContado: 10}},
	})
	if !errors.Is(err, ErrPaqueteInvalido) {
		t.Fatalf("esperaba ErrPaqueteInvalido, got %v", err)
	}
}

func TestCrearPaquete_RequiereAlMenosUnTier(t *testing.T) {
	svc := NewPaquetesService(&fakePaquetesRepo{})
	_, err := svc.Crear(CrearPaqueteInput{Nombre: "X", LocalIDs: []int{1}, Tiers: nil})
	if !errors.Is(err, ErrPaqueteInvalido) {
		t.Fatalf("esperaba ErrPaqueteInvalido, got %v", err)
	}
}

func TestCrearPaquete_RequiereLocal(t *testing.T) {
	svc := NewPaquetesService(&fakePaquetesRepo{})
	_, err := svc.Crear(CrearPaqueteInput{
		Nombre:   "X",
		LocalIDs: nil,
		Tiers:    []models.PaqueteTierInput{{Sesiones: 1, PrecioContado: 10}},
	})
	if !errors.Is(err, ErrPaqueteInvalido) {
		t.Fatalf("esperaba ErrPaqueteInvalido, got %v", err)
	}
}

func TestCrearPaquete_PrecioRegularMayorQueContado(t *testing.T) {
	precioRegular := 10.0
	svc := NewPaquetesService(&fakePaquetesRepo{})
	_, err := svc.Crear(CrearPaqueteInput{
		Nombre:   "X",
		LocalIDs: []int{1},
		Tiers: []models.PaqueteTierInput{{
			Sesiones: 1, PrecioContado: 10, PrecioRegular: &precioRegular,
		}},
	})
	if !errors.Is(err, ErrPaqueteInvalido) {
		t.Fatalf("esperaba ErrPaqueteInvalido, got %v", err)
	}
}

func TestCrearPaquete_Valido_DelegaAlRepo(t *testing.T) {
	repo := &fakePaquetesRepo{}
	svc := NewPaquetesService(repo)
	id, err := svc.Crear(CrearPaqueteInput{
		Nombre:   "X",
		LocalIDs: []int{1},
		Tiers:    []models.PaqueteTierInput{{Sesiones: 4, PrecioContado: 100}},
	})
	if err != nil {
		t.Fatalf("Crear() error = %v", err)
	}
	if id != 1 {
		t.Fatalf("Crear() id = %v, want 1", id)
	}
	if repo.createInput.Nombre != "X" {
		t.Fatalf("el repo no recibio el input esperado: %+v", repo.createInput)
	}
}
