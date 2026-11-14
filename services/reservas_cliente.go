package services

import (
	"errors"
	"log"
)

// clienteDeLaReserva decide con que cliente queda enlazada una reserva nueva.
//
// Panel interno: el cajero elige al cliente y es obligatorio. Formulario publico: nada se adivina; solo se
// enlaza si el telefono coincide exacto con un unico cliente, y si no la reserva queda por vincular.
func (s *ReservasPGService) clienteDeLaReserva(input CrearReservaPGInput, telefonoE164 *string) (*int, error) {
	if input.Autenticado {
		if input.ClienteID == nil || *input.ClienteID <= 0 {
			return nil, errors.New("cliente_id es requerido: elija o cree al cliente de la reserva")
		}
		if s.clientes != nil {
			existe, err := s.clientes.Existe(*input.ClienteID)
			if err != nil {
				return nil, err
			}
			if !existe {
				return nil, errors.New("cliente no encontrado")
			}
		}
		return input.ClienteID, nil
	}

	if s.clientes == nil || telefonoE164 == nil {
		return nil, nil
	}
	ids, err := s.clientes.IDsPorTelefono(*telefonoE164)
	if err != nil {
		// La reserva publica no se pierde por no poder vincular: queda por vincular.
		log.Printf("[reservas] no se pudo buscar al cliente por telefono: %v", err)
		return nil, nil
	}
	return unicoCliente(ids), nil
}

// unicoCliente devuelve el cliente solo si hay exactamente uno.
func unicoCliente(ids []int) *int {
	if len(ids) != 1 {
		return nil
	}
	id := ids[0]
	return &id
}
