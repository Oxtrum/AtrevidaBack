package services

import (
	"strings"
	"testing"

	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

func decimal(v float64) *float64 { return &v }

func TestCrearItemRechazaPrecioDeVentaNegativo(t *testing.T) {
	s := &InventarioService{}

	_, err := s.CrearItem(pgsqlrepo.ItemInput{Nombre: "Anestesia", UnidadMedida: "UNIDAD", PrecioVenta: decimal(-1)})

	if err == nil || !strings.Contains(err.Error(), "precio_venta") {
		t.Fatalf("error = %v, want precio_venta negativo", err)
	}
}

func TestActualizarItemRechazaPrecioDeVentaNegativo(t *testing.T) {
	s := &InventarioService{}

	err := s.ActualizarItem(3, pgsqlrepo.ItemPatch{PrecioVenta: decimal(-5), PrecioVentaSet: true})

	if err == nil || !strings.Contains(err.Error(), "precio_venta") {
		t.Fatalf("error = %v, want precio_venta negativo", err)
	}
}

func TestActualizarItemSoloConPrecioDeVentaCuentaComoCambio(t *testing.T) {
	s := &InventarioService{}

	err := s.ActualizarItem(3, pgsqlrepo.ItemPatch{PrecioVenta: decimal(-5), PrecioVentaSet: true})

	if err != nil && strings.Contains(err.Error(), "no se enviaron campos") {
		t.Fatalf("un patch solo con precio_venta es un cambio valido: %v", err)
	}
}
