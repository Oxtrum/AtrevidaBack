package pgsql

import (
	"strings"
	"testing"
)

func TestElListadoDeItemsTraeElPrecioDeVenta(t *testing.T) {
	if !strings.Contains(itemsSelectBase, "i.precio_venta") {
		t.Fatalf("el item debe devolver precio_venta: %s", itemsSelectBase)
	}
}
