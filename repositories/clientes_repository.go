package repository

import "context"

import "atrevida-agenda-api/models"

type FiltroClientes struct {
	Context        context.Context
	Nombre         string
	Apellido       string
	NumeroTelefono string
	Busqueda       string
	PageLimit      int
	CursorSet      bool
	CursorApellido string
	CursorNombre   string
	CursorID       int
}

type CrearClienteInput struct {
	Nombre         string
	Apellido       string
	NumeroTelefono string
	CI             string
	NIT            string
}

type ActualizarClienteInput struct {
	ID             int
	Nombre         *string
	Apellido       *string
	NumeroTelefono *string
	CI             *string
	NIT            *string
}

type ClientesRepository interface {
	GetClientes(filtro FiltroClientes) ([]models.ClientePG, error)
	CountClientes(filtro FiltroClientes) (int, error)
	GetClienteByID(id int) (*models.ClientePG, error)
	CreateCliente(input CrearClienteInput) (int, error)
	UpdateCliente(input ActualizarClienteInput) error
	DeleteCliente(id int) error
}
