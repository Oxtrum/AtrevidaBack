package pgsql

import (
	"sort"
	"strings"
	"testing"
)

func TestDeudoresOrdenablesExponeLasColumnasDelListado(t *testing.T) {
	want := []string{"apellido", "deuda", "nombre", "numero_telefono", "total_abonos", "total_cargos"}
	got := make([]string, 0, len(DeudoresOrdenables))
	for key := range DeudoresOrdenables {
		got = append(got, key)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("columnas ordenables = %v, esperadas %v", got, want)
	}
}

func TestBuildDeudoresQuerySinOrdenMuestraLaMayorDeudaPrimero(t *testing.T) {
	query, args := buildDeudoresQuery(FiltroDeudores{Limit: 51})
	if !strings.Contains(query, "ORDER BY (COALESCE(ca.total, 0) - COALESCE(ab.total, 0)) DESC, c.id") {
		t.Fatalf("orden por defecto inesperado: %s", query)
	}
	if !strings.Contains(query, "LIMIT $1") || strings.Contains(query, "OFFSET") || len(args) != 1 || args[0] != 51 {
		t.Fatalf("paginacion inesperada: %s %#v", query, args)
	}
}

func TestBuildDeudoresQuerySoloListaClientesConDeuda(t *testing.T) {
	query, _ := buildDeudoresQuery(FiltroDeudores{Limit: 10})
	if !strings.Contains(query, "COALESCE(ca.total, 0) - COALESCE(ab.total, 0) > 0") {
		t.Fatalf("debe filtrar solo a quienes deben: %s", query)
	}
	for _, fragmento := range []string{"es_abono = TRUE", "activo = TRUE", "estado = 'PAGADO'", "anulado = FALSE"} {
		if !strings.Contains(query, fragmento) {
			t.Fatalf("el saldo debe contar solo %s: %s", fragmento, query)
		}
	}
}

func TestBuildDeudoresQueryConOrdenYDesplazamiento(t *testing.T) {
	query, args := buildDeudoresQuery(FiltroDeudores{OrdenarPor: "apellido", OrdenDesc: true, Limit: 51, Offset: 50})
	if !strings.Contains(query, "ORDER BY TRANSLATE(LOWER(c.apellido)") || !strings.Contains(query, "DESC NULLS LAST, c.id DESC") {
		t.Fatalf("ORDER BY inesperado: %s", query)
	}
	if !strings.Contains(query, "LIMIT $1 OFFSET $2") || len(args) != 2 || args[1] != 50 {
		t.Fatalf("limit/offset inesperados: %s %#v", query, args)
	}
}

func TestBuildDeudoresQueryConBusquedaBuscaPorNombreYTelefono(t *testing.T) {
	query, args := buildDeudoresQuery(FiltroDeudores{Busqueda: "Maria", Limit: 10})
	if !strings.Contains(query, "ILIKE $1") || !strings.Contains(query, "numero_telefono") {
		t.Fatalf("la busqueda debe ser parametrizada: %s", query)
	}
	if len(args) != 2 || args[0] != "%Maria%" || args[1] != 10 {
		t.Fatalf("args = %#v", args)
	}
}
