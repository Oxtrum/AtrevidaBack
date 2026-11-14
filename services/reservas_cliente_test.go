package services

import (
	"errors"
	"os"
	"strings"
	"testing"

	repository "atrevida-agenda-api/repositories"
)

type reservaClientesFake struct {
	porTelefono map[string][]int
	existentes  map[int]bool
	errTelefono error
}

func (f *reservaClientesFake) IDsPorTelefono(e164 string) ([]int, error) {
	return f.porTelefono[e164], f.errTelefono
}

func (f *reservaClientesFake) Existe(id int) (bool, error) {
	return f.existentes[id], nil
}

type reservasClienteRepo struct {
	reservaCostoUpdateRepo
	ultima  repository.CreateReservaInput
	creadas int
}

func (r *reservasClienteRepo) CreateReserva(input repository.CreateReservaInput) (int, error) {
	r.creadas++
	r.ultima = input
	return 9, nil
}

func entero(n int) *int { return &n }

func inputReservaCliente() CrearReservaPGInput {
	in := inputReservaDia()
	in.Telefono = "+59170011223"
	in.ConfirmarDuplicado = true
	return in
}

func TestCrearReservaDelPanelExigeCliente(t *testing.T) {
	repo := &reservasClienteRepo{}
	svc := &ReservasPGService{repo: repo, clientes: &reservaClientesFake{}}
	in := inputReservaCliente()
	in.ClienteID = nil

	_, err := svc.CrearReservaConAvisos(in)

	if err == nil || !strings.Contains(err.Error(), "cliente_id") {
		t.Fatalf("error = %v, want que pida cliente_id", err)
	}
	if repo.creadas != 0 {
		t.Fatalf("CreateReserva calls = %d, want 0", repo.creadas)
	}
}

func TestCrearReservaDelPanelGuardaElClienteElegido(t *testing.T) {
	repo := &reservasClienteRepo{}
	svc := &ReservasPGService{repo: repo, clientes: &reservaClientesFake{existentes: map[int]bool{5: true}}}
	in := inputReservaCliente()
	in.ClienteID = entero(5)

	if _, err := svc.CrearReservaConAvisos(in); err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if repo.ultima.ClienteID == nil || *repo.ultima.ClienteID != 5 {
		t.Fatalf("ClienteID = %v, want 5", repo.ultima.ClienteID)
	}
}

func TestCrearReservaDelPanelRechazaClienteInexistente(t *testing.T) {
	repo := &reservasClienteRepo{}
	svc := &ReservasPGService{repo: repo, clientes: &reservaClientesFake{existentes: map[int]bool{}}}
	in := inputReservaCliente()
	in.ClienteID = entero(77)

	_, err := svc.CrearReservaConAvisos(in)

	if err == nil || !strings.Contains(err.Error(), "no encontrad") {
		t.Fatalf("error = %v, want cliente no encontrado", err)
	}
	if repo.creadas != 0 {
		t.Fatalf("CreateReserva calls = %d, want 0", repo.creadas)
	}
}

func TestCrearReservaPublicaSeVinculaConTelefonoExactoYUnico(t *testing.T) {
	repo := &reservasClienteRepo{}
	svc := &ReservasPGService{repo: repo, clientes: &reservaClientesFake{
		porTelefono: map[string][]int{"+59170011223": {12}},
	}}
	in := inputReservaCliente()
	in.Autenticado = false

	if _, err := svc.CrearReservaConAvisos(in); err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if repo.ultima.ClienteID == nil || *repo.ultima.ClienteID != 12 {
		t.Fatalf("ClienteID = %v, want 12", repo.ultima.ClienteID)
	}
}

func TestCrearReservaPublicaConTelefonoRepetidoQuedaPorVincular(t *testing.T) {
	repo := &reservasClienteRepo{}
	svc := &ReservasPGService{repo: repo, clientes: &reservaClientesFake{
		porTelefono: map[string][]int{"+59170011223": {12, 13}},
	}}
	in := inputReservaCliente()
	in.Autenticado = false

	if _, err := svc.CrearReservaConAvisos(in); err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if repo.ultima.ClienteID != nil {
		t.Fatalf("ClienteID = %d, want nil: no se adivina entre dos clientes", *repo.ultima.ClienteID)
	}
}

func TestCrearReservaPublicaSinCoincidenciaQuedaPorVincular(t *testing.T) {
	repo := &reservasClienteRepo{}
	svc := &ReservasPGService{repo: repo, clientes: &reservaClientesFake{}}
	in := inputReservaCliente()
	in.Autenticado = false

	if _, err := svc.CrearReservaConAvisos(in); err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if repo.creadas != 1 || repo.ultima.ClienteID != nil {
		t.Fatalf("creadas = %d, ClienteID = %v, want 1 y nil", repo.creadas, repo.ultima.ClienteID)
	}
}

func TestCrearReservaPublicaNoFallaSiLaBusquedaDeClienteFalla(t *testing.T) {
	repo := &reservasClienteRepo{}
	svc := &ReservasPGService{repo: repo, clientes: &reservaClientesFake{errTelefono: errors.New("db caida")}}
	in := inputReservaCliente()
	in.Autenticado = false

	if _, err := svc.CrearReservaConAvisos(in); err != nil {
		t.Fatalf("error = %v, want nil: la reserva publica no se pierde por no poder vincular", err)
	}
	if repo.creadas != 1 || repo.ultima.ClienteID != nil {
		t.Fatalf("creadas = %d, ClienteID = %v, want 1 y nil", repo.creadas, repo.ultima.ClienteID)
	}
}

func TestCrearReservaPublicaIgnoraUnClienteIDQueVengaDelCliente(t *testing.T) {
	repo := &reservasClienteRepo{}
	svc := &ReservasPGService{repo: repo, clientes: &reservaClientesFake{existentes: map[int]bool{5: true}}}
	in := inputReservaCliente()
	in.Autenticado = false
	in.ClienteID = entero(5)

	if _, err := svc.CrearReservaConAvisos(in); err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if repo.ultima.ClienteID != nil {
		t.Fatalf("ClienteID = %d, want nil: el formulario publico no elige cliente", *repo.ultima.ClienteID)
	}
}

func TestLaMigracionDelHistoricoSoloEnlazaTelefonosExactosYUnicos(t *testing.T) {
	sql, err := os.ReadFile("../migrations/000075_reservas_cliente_y_precio_insumos.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	texto := string(sql)
	for _, parte := range []string{"HAVING COUNT(*) = 1", "r.telefono_e164 = c.telefono_e164", "r.cliente_id IS NULL"} {
		if !strings.Contains(texto, parte) {
			t.Fatalf("la migracion debe enlazar solo coincidencias exactas y unicas; falta %q", parte)
		}
	}
	if strings.Contains(texto, "LIKE") || strings.Contains(texto, "RIGHT(") {
		t.Fatal("la migracion no puede enlazar por coincidencias aproximadas")
	}
}
