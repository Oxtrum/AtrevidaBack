package pgsql

import (
	"strings"
	"testing"

	repository "atrevida-agenda-api/repositories"
)

func TestPagoProductosJoinResumeSinNMasUno(t *testing.T) {
	join := pagoProductosJoin()
	for _, fragmento := range []string{"LEFT JOIN LATERAL", "ARRAY_AGG", "COUNT(*)", "dp.pago_id = p.id"} {
		if !strings.Contains(join, fragmento) {
			t.Fatalf("join no contiene %q: %s", fragmento, join)
		}
	}
	columnas := pagoSelectColumns()
	for _, columna := range []string{"primer_producto", "cantidad_productos"} {
		if !strings.Contains(columnas, columna) {
			t.Fatalf("select no contiene %q", columna)
		}
	}
}

func TestPagoConditionsFiltraCualquierProducto(t *testing.T) {
	conditions, args := pagoConditions(repository.FiltroPagos{Producto: "masaje"})
	query := strings.Join(conditions, " ")
	for _, fragmento := range []string{"EXISTS", "detalle_pagos", "dp_filtro.pago_id = p.id", "TRANSLATE"} {
		if !strings.Contains(query, fragmento) {
			t.Fatalf("filtro no contiene %q: %s", fragmento, query)
		}
	}
	if len(args) != 1 || args[0] != "%masaje%" {
		t.Fatalf("args = %#v", args)
	}
}
