package services

import (
	"errors"

	"atrevida-agenda-api/models"
)

const (
	vinculoPaginaPorDefecto = 50
	vinculoPaginaMaxima     = 100
)

// ReservasVinculoStore es lo que el servicio necesita del almacen de reservas por vincular.
type ReservasVinculoStore interface {
	PorVincular(limit, offset int) ([]models.ReservaPorVincularPG, int, error)
	Sugerencias(reservaID int) ([]models.ClienteSugeridoPG, error)
	Vincular(reservaID, clienteID int, mismoTelefono bool) (int, error)
}

// ReservasVinculoService enlaza reservas con clientes registrados a pedido del cajero.
type ReservasVinculoService struct {
	store ReservasVinculoStore
}

func NewReservasVinculoService(store ReservasVinculoStore) *ReservasVinculoService {
	return &ReservasVinculoService{store: store}
}

// PorVincular lista las reservas vigentes sin cliente, mas recientes primero.
func (s *ReservasVinculoService) PorVincular(limit, offset int) ([]models.ReservaPorVincularPG, int, error) {
	if limit <= 0 {
		limit = vinculoPaginaPorDefecto
	}
	if limit > vinculoPaginaMaxima {
		limit = vinculoPaginaMaxima
	}
	if offset < 0 {
		offset = 0
	}
	return s.store.PorVincular(limit, offset)
}

// Sugerencias propone clientes para una reserva por vincular. Es una pista: nunca vincula.
func (s *ReservasVinculoService) Sugerencias(reservaID int) ([]models.ClienteSugeridoPG, error) {
	if reservaID <= 0 {
		return nil, errors.New("reserva_id es requerido")
	}
	return s.store.Sugerencias(reservaID)
}

// Vincular enlaza la reserva con el cliente elegido. Con mismoTelefono tambien enlaza las demas reservas
// sin cliente que tengan exactamente ese telefono.
func (s *ReservasVinculoService) Vincular(reservaID, clienteID int, mismoTelefono bool) (models.ResultadoVinculo, error) {
	if reservaID <= 0 {
		return models.ResultadoVinculo{}, errors.New("reserva_id es requerido")
	}
	if clienteID <= 0 {
		return models.ResultadoVinculo{}, errors.New("cliente_id es requerido")
	}
	n, err := s.store.Vincular(reservaID, clienteID, mismoTelefono)
	if err != nil {
		return models.ResultadoVinculo{}, err
	}
	return models.ResultadoVinculo{Vinculadas: n}, nil
}
