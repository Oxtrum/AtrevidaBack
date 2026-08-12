package services

import (
	"context"
	"strings"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type ClientesService struct {
	repo repository.ClientesRepository
}

func NewClientesService(repo repository.ClientesRepository) *ClientesService {
	return &ClientesService{repo: repo}
}

type FiltroClientes struct {
	Context        context.Context
	Nombre         string
	Apellido       string
	NumeroTelefono string
}

func (s *ClientesService) GetClientes(filtro FiltroClientes) ([]models.ClientePG, error) {
	return s.repo.GetClientes(repository.FiltroClientes{
		Context:        filtro.Context,
		Nombre:         strings.TrimSpace(filtro.Nombre),
		Apellido:       strings.TrimSpace(filtro.Apellido),
		NumeroTelefono: strings.TrimSpace(filtro.NumeroTelefono),
	})
}

func (s *ClientesService) GetClienteByID(id int) (*models.ClientePG, error) {
	return s.repo.GetClienteByID(id)
}

type CrearClienteInput struct {
	Nombre         string
	Apellido       string
	NumeroTelefono string
	CI             string
	NIT            string
}

func (s *ClientesService) CreateCliente(input CrearClienteInput) (int, error) {
	return s.repo.CreateCliente(repository.CrearClienteInput{
		Nombre:         strings.TrimSpace(input.Nombre),
		Apellido:       strings.TrimSpace(input.Apellido),
		NumeroTelefono: strings.TrimSpace(input.NumeroTelefono),
		CI:             strings.TrimSpace(input.CI),
		NIT:            strings.TrimSpace(input.NIT),
	})
}

type ActualizarClienteInput struct {
	ID             int
	Nombre         *string
	Apellido       *string
	NumeroTelefono *string
	CI             *string
	NIT            *string
}

func (s *ClientesService) UpdateCliente(input ActualizarClienteInput) error {
	trim := func(value *string) *string {
		if value == nil {
			return nil
		}
		recortado := strings.TrimSpace(*value)
		return &recortado
	}

	return s.repo.UpdateCliente(repository.ActualizarClienteInput{
		ID:             input.ID,
		Nombre:         trim(input.Nombre),
		Apellido:       trim(input.Apellido),
		NumeroTelefono: trim(input.NumeroTelefono),
		CI:             trim(input.CI),
		NIT:            trim(input.NIT),
	})
}

func (s *ClientesService) DeleteCliente(id int) error {
	return s.repo.DeleteCliente(id)
}
