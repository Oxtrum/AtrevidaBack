package services

import (
	"testing"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type fakePagosRepo struct {
	pago *models.PagoCompletoPG
}

func (f *fakePagosRepo) GetPagos(filtro repository.FiltroPagos) ([]models.PagoPG, error) {
	return nil, nil
}

func (f *fakePagosRepo) GetPagoByCodigo(codigoPago string) (*models.PagoCompletoPG, error) {
	return f.pago, nil
}

func (f *fakePagosRepo) CreatePago(input repository.CrearPagoInput) (string, error) {
	return "", nil
}

func (f *fakePagosRepo) UpdatePago(input repository.ActualizarPagoInput) error {
	return nil
}

func (f *fakePagosRepo) DeletePago(codigoPago string) error {
	return nil
}

func (f *fakePagosRepo) GetResumenPagos(filtro repository.FiltroResumenPagos) (repository.PagoResumenAgregado, error) {
	return repository.PagoResumenAgregado{}, nil
}

func TestGetPagoByCodigoAplicaLocalID(t *testing.T) {
	service := NewPagosService(&fakePagosRepo{
		pago: &models.PagoCompletoPG{
			PagoPG: models.PagoPG{
				CodigoPago:        "PAGO-000001",
				LocalID:           1,
				LocalNombre:       "SAN MARTIN",
				ClienteNombre:     "Maria Lopez",
				TipoPago:          "efectivo",
				Estado:            "PAGADO",
				Activo:            true,
				FechaCreacion:     time.Date(2026, time.May, 23, 0, 0, 0, 0, time.UTC),
				FechaModificacion: time.Date(2026, time.May, 23, 0, 0, 0, 0, time.UTC),
			},
		},
	})

	scopeLocalID := 2
	if _, err := service.GetPagoByCodigo("PAGO-000001", &scopeLocalID); err == nil {
		t.Fatal("GetPagoByCodigo() error = nil, want pago no encontrado")
	}

	scopeLocalID = 1
	if _, err := service.GetPagoByCodigo("PAGO-000001", &scopeLocalID); err != nil {
		t.Fatalf("GetPagoByCodigo() error = %v, want nil", err)
	}
}
