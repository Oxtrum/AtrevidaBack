package services

import (
	"strings"
	"testing"

	repository "atrevida-agenda-api/repositories"
)

// pagosActorRepo registra a quien anula un pago.
type pagosActorRepo struct {
	pagosFechaRepo
	codigo string
	actor  repository.ConsumoActor
}

func (r *pagosActorRepo) DeletePago(codigo string, actor repository.ConsumoActor) error {
	r.codigo, r.actor = codigo, actor
	return nil
}

func detalleInsumo(item *int, servicioID *int) []CrearDetallePagoInput {
	return []CrearDetallePagoInput{{
		ItemID: item, ServicioID: servicioID, Servicio: "Crema", PrecioUnitario: 20, Cantidad: 2,
	}}
}

func TestCreatePagoPasaElInsumoYCalculaSuSubtotal(t *testing.T) {
	repo := &pagosFechaRepo{}
	item := 7
	in := pagoValido()
	in.Detalle = detalleInsumo(&item, nil)
	if _, err := servicioConFecha(repo, 0).CreatePago(in); err != nil {
		t.Fatal(err)
	}
	d := repo.creado.Detalle[0]
	if d.ItemID == nil || *d.ItemID != 7 || d.Subtotal != 40 {
		t.Fatalf("detalle = %+v", d)
	}
}

func TestCreatePagoRechazaDetalleServicioEInsumoAlMismoTiempo(t *testing.T) {
	item, servicio := 7, 3
	in := pagoValido()
	in.Detalle = detalleInsumo(&item, &servicio)
	_, err := servicioConFecha(&pagosFechaRepo{}, 0).CreatePago(in)
	if err == nil || !strings.Contains(err.Error(), "servicio e insumo") {
		t.Fatalf("error = %v", err)
	}
}

func TestCreatePagoRechazaItemIDInvalido(t *testing.T) {
	item := 0
	in := pagoValido()
	in.Detalle = detalleInsumo(&item, nil)
	_, err := servicioConFecha(&pagosFechaRepo{}, 0).CreatePago(in)
	if err == nil || !strings.Contains(err.Error(), "item_id invalido") {
		t.Fatalf("error = %v", err)
	}
}

func TestCreatePagoPermiteStockNegativoSegunLaConfiguracion(t *testing.T) {
	item := 7
	for _, permitir := range []bool{true, false} {
		repo := &pagosFechaRepo{}
		cfg := configPorDefecto()
		cfg.PermitirStockNegativo = permitir
		s := NewPagosService(repo)
		s.Config = configFija{cfg}
		in := pagoValido()
		in.Detalle = detalleInsumo(&item, nil)
		if _, err := s.CreatePago(in); err != nil {
			t.Fatal(err)
		}
		if repo.creado.PermitirStockNegativo != permitir {
			t.Fatalf("permitir_stock_negativo=%v llego como %v", permitir, repo.creado.PermitirStockNegativo)
		}
	}
}

func TestDeletePagoPasaQuienAnula(t *testing.T) {
	repo := &pagosActorRepo{}
	s := NewPagosService(repo)
	id := 4
	if err := s.DeletePago(" PAGO-000001 ", CajeroAuditoriaInput{ID: &id, Nombre: "Ana"}); err != nil {
		t.Fatal(err)
	}
	if repo.codigo != "PAGO-000001" || repo.actor.UsuarioID == nil || *repo.actor.UsuarioID != 4 || repo.actor.UsuarioNombre != "Ana" {
		t.Fatalf("repo recibio %q %+v", repo.codigo, repo.actor)
	}
}
