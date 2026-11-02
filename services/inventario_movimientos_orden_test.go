package services

import (
	"strings"
	"testing"

	repository "atrevida-agenda-api/repositories"
)

func TestListarMovimientosRechazaOrdenFueraDeLaListaBlanca(t *testing.T) {
	s := &InventarioService{}
	for _, clave := range []string{"password", "m.id", "cantidad; DROP TABLE inventario_movimientos"} {
		_, err := s.ListarMovimientos(repository.FiltroMovimientosInventario{OrdenarPor: clave})
		if err == nil || !strings.Contains(err.Error(), "ordenar_por invalido") {
			t.Fatalf("%q: error = %v", clave, err)
		}
	}
}

func TestListarMovimientosRechazaDesplazamientoNegativo(t *testing.T) {
	s := &InventarioService{}
	_, err := s.ListarMovimientos(repository.FiltroMovimientosInventario{OrdenarPor: "cantidad", Offset: -1})
	if err == nil || !strings.Contains(err.Error(), "offset invalido") {
		t.Fatalf("error = %v", err)
	}
}
