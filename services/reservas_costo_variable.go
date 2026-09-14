package services

import (
	"strings"

	"atrevida-agenda-api/models"
)

// Reprogramar o reenviar el mismo servicio conserva la modalidad historica.
// Una confirmacion explicita puede fijar la modalidad de una reserva antigua.
func (s *ReservasPGService) modalidadCostoParaCambio(current *models.ReservaPGCompleta, nuevoServicio *string, confirmacion bool) (*bool, bool) {
	if nuevoServicio == nil || strings.TrimSpace(*nuevoServicio) == "" {
		return nil, false
	}
	nombreActual := current.ServicioNombre
	if current.ServicioConfirmado != nil && strings.TrimSpace(*current.ServicioConfirmado) != "" {
		nombreActual = current.ServicioConfirmado
	}
	if nombreActual != nil && normalizarNombreServicio(*nombreActual) == normalizarNombreServicio(*nuevoServicio) {
		if current.CostoVariable != nil || !confirmacion {
			return nil, false
		}
	}
	if s.serviciosRepo == nil {
		return nil, true
	}
	servicio, err := s.serviciosRepo.GetServicioByNombre(strings.TrimSpace(*nuevoServicio))
	if err != nil || servicio == nil {
		return nil, true
	}
	variable := servicio.CostoVariable
	return &variable, true
}
