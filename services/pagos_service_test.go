package services

import (
	"testing"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type fakePagosRepo struct {
	pago        *models.PagoCompletoPG
	createdPago *repository.CrearPagoInput
}

func (f *fakePagosRepo) CountPagos(repository.FiltroPagos) (int, error) { return 0, nil }

func (f *fakePagosRepo) GetPagos(filtro repository.FiltroPagos) ([]models.PagoPG, error) {
	return nil, nil
}

func (f *fakePagosRepo) GetPagoByCodigo(codigoPago string) (*models.PagoCompletoPG, error) {
	return f.pago, nil
}

func (f *fakePagosRepo) CreatePago(input repository.CrearPagoInput) (string, error) {
	f.createdPago = &input
	return "PAGO-000001", nil
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

func TestCreatePagoCalculaDetalleDesdePrecioYCantidad(t *testing.T) {
	repo := &fakePagosRepo{}
	service := NewPagosService(repo)

	_, err := service.CreatePago(CrearPagoInput{
		LocalID:       1,
		LocalNombre:   "SAN MARTIN",
		ClienteNombre: "Maria Lopez",
		Descuento:     floatPtr(10),
		TipoPago:      "efectivo",
		Estado:        "PAGADO",
		Activo:        true,
		Cajero:        CajeroAuditoriaInput{Nombre: "admin"},
		Detalle: []CrearDetallePagoInput{{
			Servicio:       "Servicio reevaluado",
			PrecioUnitario: 99.999,
			Cantidad:       3,
			Subtotal:       1,
		}},
	})
	if err != nil {
		t.Fatalf("CreatePago() error = %v", err)
	}
	if repo.createdPago == nil {
		t.Fatal("CreatePago() did not call repository")
	}
	if got := repo.createdPago.Detalle[0].PrecioUnitario; got != 100 {
		t.Fatalf("precio_unitario = %v, want 100", got)
	}
	if got := repo.createdPago.Detalle[0].Subtotal; got != 300 {
		t.Fatalf("subtotal detalle = %v, want 300", got)
	}
	if got := *repo.createdPago.Subtotal; got != 300 {
		t.Fatalf("subtotal pago = %v, want 300", got)
	}
	if got := *repo.createdPago.TotalFinal; got != 290 {
		t.Fatalf("total_final = %v, want 290", got)
	}
}

func TestCreatePagoRechazaSubtotalDeCabeceraInconsistente(t *testing.T) {
	service := NewPagosService(&fakePagosRepo{})

	_, err := service.CreatePago(CrearPagoInput{
		LocalID:       1,
		LocalNombre:   "SAN MARTIN",
		ClienteNombre: "Maria Lopez",
		Subtotal:      floatPtr(1),
		Descuento:     floatPtr(0),
		TipoPago:      "qr",
		Estado:        "PAGADO",
		Activo:        true,
		Cajero:        CajeroAuditoriaInput{Nombre: "admin"},
		Detalle: []CrearDetallePagoInput{{
			Servicio:       "Servicio reevaluado",
			PrecioUnitario: 100,
			Cantidad:       1,
			Subtotal:       100,
		}},
	})
	if err == nil || err.Error() != "subtotal no coincide con el detalle del pago" {
		t.Fatalf("CreatePago() error = %v, want subtotal inconsistente", err)
	}
}

func floatPtr(value float64) *float64 {
	return &value
}

func TestPagoMixtoCantidadesYDescuentos(t *testing.T) {
	for _, descuento := range []float64{0, 10.25, 490.5, -1, 491} {
		repo := &fakePagosRepo{}
		svc := NewPagosService(repo)
		_, err := svc.CreatePago(CrearPagoInput{
			LocalID: 1, LocalNombre: "SAN MARTIN", ClienteNombre: "Prueba", Descuento: &descuento,
			TipoPago: "qr", Estado: "PAGADO", Activo: true, Cajero: CajeroAuditoriaInput{Nombre: "admin"},
			Detalle: []CrearDetallePagoInput{
				{Servicio: "Fijo", PrecioUnitario: 70.25, Cantidad: 2},
				{Servicio: "Variable acordado cero", PrecioUnitario: 0, Cantidad: 1},
				{Servicio: "Paquete", PrecioUnitario: 350, Cantidad: 1},
			},
		})
		valido := descuento >= 0 && descuento <= 490.5
		if (err == nil) != valido {
			t.Fatalf("descuento %v: %v", descuento, err)
		}
		if valido {
			if *repo.createdPago.Subtotal != 490.5 || *repo.createdPago.TotalFinal != redondearMoneda(490.5-descuento) {
				t.Fatal("total incorrecto")
			}
		} else if repo.createdPago != nil {
			t.Fatal("pago invalido llego al repositorio")
		}
	}
}
