package services

import (
	"strings"
	"testing"

	"atrevida-agenda-api/models"
)

type vinculoStoreFake struct {
	limit, offset int
	reservaID     int
	clienteID     int
	actualizarTel bool
	llamadas      int
	resultado     models.ResultadoVinculo
}

func (f *vinculoStoreFake) PorVincular(limit, offset int) ([]models.ReservaPorVincularPG, int, error) {
	f.limit, f.offset = limit, offset
	return []models.ReservaPorVincularPG{}, 0, nil
}

func (f *vinculoStoreFake) Sugerencias(reservaID int) ([]models.ClienteSugeridoPG, error) {
	f.reservaID = reservaID
	return []models.ClienteSugeridoPG{}, nil
}

func (f *vinculoStoreFake) Vincular(reservaID, clienteID int, actualizarTelefono bool) (models.ResultadoVinculo, error) {
	f.llamadas++
	f.reservaID, f.clienteID, f.actualizarTel = reservaID, clienteID, actualizarTelefono
	return f.resultado, nil
}

func TestVincularReservaValidaLosIDs(t *testing.T) {
	for _, c := range []struct{ reserva, cliente int }{{0, 5}, {-1, 5}, {7, 0}, {7, -3}} {
		f := &vinculoStoreFake{}
		_, err := NewReservasVinculoService(f).Vincular(c.reserva, c.cliente, false)
		if err == nil || !strings.Contains(err.Error(), "requerido") {
			t.Fatalf("Vincular(%d, %d) error = %v, want requerido", c.reserva, c.cliente, err)
		}
		if f.llamadas != 0 {
			t.Fatalf("Vincular(%d, %d) llego al repositorio", c.reserva, c.cliente)
		}
	}
}

func TestVincularReservaPasaLaOpcionDeActualizarElTelefono(t *testing.T) {
	f := &vinculoStoreFake{resultado: models.ResultadoVinculo{Vinculadas: 1, TelefonoActualizado: true}}

	res, err := NewReservasVinculoService(f).Vincular(7, 5, true)

	if err != nil || res.Vinculadas != 1 || !res.TelefonoActualizado {
		t.Fatalf("res = %+v, err = %v", res, err)
	}
	if f.reservaID != 7 || f.clienteID != 5 || !f.actualizarTel {
		t.Fatalf("store = %+v", f)
	}
}

func TestPorVincularAcotaLaPagina(t *testing.T) {
	casos := []struct{ limit, offset, wantLimit, wantOffset int }{
		{0, 0, 50, 0},
		{-5, -1, 50, 0},
		{20, 40, 20, 40},
		{500, 0, 100, 0},
	}
	for _, c := range casos {
		f := &vinculoStoreFake{}
		if _, _, err := NewReservasVinculoService(f).PorVincular(c.limit, c.offset); err != nil {
			t.Fatal(err)
		}
		if f.limit != c.wantLimit || f.offset != c.wantOffset {
			t.Errorf("PorVincular(%d, %d) pidio limit=%d offset=%d, want %d y %d", c.limit, c.offset, f.limit, f.offset, c.wantLimit, c.wantOffset)
		}
	}
}

func TestSugerenciasExigeUnaReserva(t *testing.T) {
	f := &vinculoStoreFake{}
	if _, err := NewReservasVinculoService(f).Sugerencias(0); err == nil || !strings.Contains(err.Error(), "requerido") {
		t.Fatalf("error = %v, want requerido", err)
	}
}
