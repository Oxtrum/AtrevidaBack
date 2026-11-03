package services

import (
	"strings"
	"testing"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type pagosFechaRepo struct {
	creado *repository.CrearPagoInput
}

func (r *pagosFechaRepo) GetPagos(repository.FiltroPagos) ([]models.PagoPG, error) { return nil, nil }
func (r *pagosFechaRepo) CountPagos(repository.FiltroPagos) (int, error)          { return 0, nil }
func (r *pagosFechaRepo) GetPagoByCodigo(string) (*models.PagoCompletoPG, error)  { return nil, nil }
func (r *pagosFechaRepo) CreatePago(in repository.CrearPagoInput) (string, error) {
	r.creado = &in
	return "PAGO-000001", nil
}
func (r *pagosFechaRepo) UpdatePago(repository.ActualizarPagoInput) error { return nil }
func (r *pagosFechaRepo) DeletePago(string, repository.ConsumoActor) error { return nil }
func (r *pagosFechaRepo) GetResumenPagos(repository.FiltroResumenPagos) (repository.PagoResumenAgregado, error) {
	return repository.PagoResumenAgregado{}, nil
}
func (r *pagosFechaRepo) ListVentasDetalle(repository.FiltroResumenPagos, int) ([]repository.VentaDetalleRow, error) {
	return nil, nil
}

type configFija struct{ cfg ConfigTrastienda }

func (c configFija) Snapshot() (ConfigTrastienda, error) { return c.cfg, nil }

func pagoValido() CrearPagoInput {
	return CrearPagoInput{
		LocalID: 1, LocalNombre: "SAN MARTIN", ClienteNombre: "Maria Lopez",
		TipoPago: "efectivo", Estado: "PAGADO", Activo: true,
		Cajero:  CajeroAuditoriaInput{Nombre: "Juan"},
		Detalle: []CrearDetallePagoInput{{Servicio: "Limpieza facial", PrecioUnitario: 100, Cantidad: 1}},
	}
}

func servicioConFecha(repo *pagosFechaRepo, maxDias int) *PagosService {
	cfg := configPorDefecto()
	cfg.DiasMaximoFechaPasada = maxDias
	s := NewPagosService(repo)
	s.Config = configFija{cfg}
	s.Ahora = func() time.Time { return ahoraPrueba }
	return s
}

func TestCreatePagoSinFechaHechoDejaQueLaBaseUseElMomentoActual(t *testing.T) {
	repo := &pagosFechaRepo{}
	if _, err := servicioConFecha(repo, 0).CreatePago(pagoValido()); err != nil {
		t.Fatal(err)
	}
	if repo.creado == nil || repo.creado.FechaHecho != nil {
		t.Fatalf("sin fecha_hecho no se envia fecha: %#v", repo.creado)
	}
}

func TestCreatePagoConFechaHechoPasadaLaGuardaAMediodiaLocal(t *testing.T) {
	repo := &pagosFechaRepo{}
	in := pagoValido()
	in.FechaHecho = "2026-10-25"
	if _, err := servicioConFecha(repo, 0).CreatePago(in); err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 10, 25, 12, 0, 0, 0, laPaz)
	if repo.creado == nil || repo.creado.FechaHecho == nil || !repo.creado.FechaHecho.Equal(want) {
		t.Fatalf("fecha enviada = %v, esperada %v", repo.creado, want)
	}
}

func TestCreatePagoRechazaFechaHechoFuturaYLimiteDeDias(t *testing.T) {
	casos := []struct {
		fecha   string
		maxDias int
		texto   string
	}{
		{"2026-10-29", 0, "no puede ser futura"},
		{"2026-10-20", 3, "no puede ser anterior a 3 dias"},
		{"28/10/2026", 0, "formato"},
	}
	for _, c := range casos {
		repo := &pagosFechaRepo{}
		in := pagoValido()
		in.FechaHecho = c.fecha
		_, err := servicioConFecha(repo, c.maxDias).CreatePago(in)
		if err == nil || !strings.Contains(err.Error(), c.texto) {
			t.Fatalf("%s: error = %v", c.fecha, err)
		}
		if repo.creado != nil {
			t.Fatalf("%s: un pago con fecha invalida no debe llegar al repositorio", c.fecha)
		}
	}
}
