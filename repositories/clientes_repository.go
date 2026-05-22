package repository

import "atrevida-agenda-api/models"

type FiltroClientes struct {
	Nombre         string
	Apellido       string
	NumeroTelefono string
}

type ActualizarClienteInput struct {
	ID             int
	Nombre         *string
	Apellido       *string
	NumeroTelefono *string
}

type ClientesRepository interface {
	GetClientes(filtro FiltroClientes) ([]models.ClientePG, error)
	GetClienteByID(id int) (*models.ClientePG, error)
	CreateCliente(nombre, apellido, numeroTelefono string) (int, error)
	UpdateCliente(input ActualizarClienteInput) error
	DeleteCliente(id int) error
}
