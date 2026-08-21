package services

import (
	"context"
	"strings"

	"atrevida-agenda-api/internal/telefono"
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
	Busqueda       string
	PageLimit      int
	CursorSet      bool
	CursorApellido string
	CursorNombre   string
	CursorID       int
}

func (s *ClientesService) GetClientes(filtro FiltroClientes) ([]models.ClientePG, error) {
	return s.repo.GetClientes(toRepositoryFiltroClientes(filtro))
}

func (s *ClientesService) CountClientes(filtro FiltroClientes) (int, error) {
	return s.repo.CountClientes(toRepositoryFiltroClientes(filtro))
}

func toRepositoryFiltroClientes(filtro FiltroClientes) repository.FiltroClientes {
	return repository.FiltroClientes{
		Context:        filtro.Context,
		Nombre:         strings.TrimSpace(filtro.Nombre),
		Apellido:       strings.TrimSpace(filtro.Apellido),
		NumeroTelefono: strings.TrimSpace(filtro.NumeroTelefono),
		Busqueda:       strings.TrimSpace(filtro.Busqueda),
		PageLimit:      filtro.PageLimit, CursorSet: filtro.CursorSet,
		CursorApellido: filtro.CursorApellido, CursorNombre: filtro.CursorNombre, CursorID: filtro.CursorID,
	}
}

func (s *ClientesService) GetClienteByID(id int) (*models.ClientePG, error) {
	return s.repo.GetClienteByID(id)
}

type CrearClienteInput struct {
	Nombre         string
	Apellido       string
	NumeroTelefono string
	TelefonoE164   *string
	CI             string
	NIT            string
}

func (s *ClientesService) CreateCliente(input CrearClienteInput) (int, error) {
	e164, err := telefono.ResolveE164(input.NumeroTelefono, input.TelefonoE164)
	if err != nil {
		return 0, err
	}
	return s.repo.CreateCliente(repository.CrearClienteInput{
		Nombre:         strings.TrimSpace(input.Nombre),
		Apellido:       strings.TrimSpace(input.Apellido),
		NumeroTelefono: strings.TrimSpace(input.NumeroTelefono),
		TelefonoE164:   e164,
		CI:             strings.TrimSpace(input.CI),
		NIT:            strings.TrimSpace(input.NIT),
	})
}

type ActualizarClienteInput struct {
	ID             int
	Nombre         *string
	Apellido       *string
	NumeroTelefono *string
	TelefonoE164   *string
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

	var e164 *string
	e164Set := false
	if input.TelefonoE164 != nil {
		value, err := telefono.NormalizeE164(*input.TelefonoE164)
		if err != nil {
			return err
		}
		e164 = &value
		e164Set = true
	} else if input.NumeroTelefono != nil {
		e164Set = true
		if value, ok := telefono.TryNormalizeLegacy(*input.NumeroTelefono); ok {
			e164 = &value
		}
	}

	return s.repo.UpdateCliente(repository.ActualizarClienteInput{
		ID:              input.ID,
		Nombre:          trim(input.Nombre),
		Apellido:        trim(input.Apellido),
		NumeroTelefono:  trim(input.NumeroTelefono),
		TelefonoE164:    e164,
		TelefonoE164Set: e164Set,
		CI:              trim(input.CI),
		NIT:             trim(input.NIT),
	})
}

func (s *ClientesService) DeleteCliente(id int) error {
	return s.repo.DeleteCliente(id)
}
