package services

import (
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
	Nombre         string
	Apellido       string
	NumeroTelefono string
}

func (s *ClientesService) GetClientes(filtro FiltroClientes) ([]models.ClientePG, error) {
	return s.repo.GetClientes(repository.FiltroClientes{
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
}

func (s *ClientesService) CreateCliente(input CrearClienteInput) (int, error) {
	return s.repo.CreateCliente(
		strings.TrimSpace(input.Nombre),
		strings.TrimSpace(input.Apellido),
		strings.TrimSpace(input.NumeroTelefono),
	)
}

type ActualizarClienteInput struct {
	ID             int
	Nombre         *string
	Apellido       *string
	NumeroTelefono *string
}

func (s *ClientesService) UpdateCliente(input ActualizarClienteInput) error {
	var nombre *string
	if input.Nombre != nil {
		value := strings.TrimSpace(*input.Nombre)
		nombre = &value
	}

	var apellido *string
	if input.Apellido != nil {
		value := strings.TrimSpace(*input.Apellido)
		apellido = &value
	}

	var numeroTelefono *string
	if input.NumeroTelefono != nil {
		value := strings.TrimSpace(*input.NumeroTelefono)
		numeroTelefono = &value
	}

	return s.repo.UpdateCliente(repository.ActualizarClienteInput{
		ID:             input.ID,
		Nombre:         nombre,
		Apellido:       apellido,
		NumeroTelefono: numeroTelefono,
	})
}

func (s *ClientesService) DeleteCliente(id int) error {
	return s.repo.DeleteCliente(id)
}
