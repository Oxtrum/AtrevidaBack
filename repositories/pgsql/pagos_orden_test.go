package pgsql

import (
	"sort"
	"strings"
	"testing"
	"time"

	repository "atrevida-agenda-api/repositories"
)

func TestPagoOrderClauseSinOrdenConservaFechaDescendente(t *testing.T) {
	got := pagoOrderClause(repository.FiltroPagos{})
	if got != "p.fecha_creacion DESC, p.id DESC" {
		t.Fatalf("orden por defecto inesperado: %s", got)
	}
}

func TestPagoOrderClauseOrdenaPorColumnaPermitida(t *testing.T) {
	got := pagoOrderClause(repository.FiltroPagos{OrdenarPor: "total_final", OrdenDesc: true})
	if got != "p.total_final DESC NULLS LAST, p.id DESC" {
		t.Fatalf("orden por total inesperado: %s", got)
	}
}

func TestPagoOrderClauseTextoIgnoraMayusculasYAcentos(t *testing.T) {
	got := pagoOrderClause(repository.FiltroPagos{OrdenarPor: "cliente_nombre"})
	for _, fragment := range []string{"TRANSLATE(LOWER(p.cliente_nombre)", "ASC NULLS LAST", "p.id ASC"} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("orden por cliente no contiene %q: %s", fragment, got)
		}
	}
}

func TestPagosOrdenablesExponeLasColumnasDeLaTabla(t *testing.T) {
	want := []string{
		"cantidad_productos", "cliente_nit", "cliente_nombre", "codigo_pago", "descuento", "estado", "fecha_creacion",
		"local_nombre", "nombre_cajero", "primer_producto", "subtotal", "tipo_pago", "total_final",
	}
	got := make([]string, 0, len(PagosOrdenables))
	for key := range PagosOrdenables {
		got = append(got, key)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("columnas ordenables = %v, esperadas %v", got, want)
	}
}

func TestBuildPagosQuerySinOrdenUsaCursorPorValor(t *testing.T) {
	query, args := buildPagosQuery(repository.FiltroPagos{
		PageLimit: 51, CursorSet: true, CursorFecha: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), CursorID: 7,
	})
	if !strings.Contains(query, "(p.fecha_creacion, p.id) < (") || strings.Contains(query, "OFFSET") {
		t.Fatalf("sin orden debe paginar por valor: %s", query)
	}
	if !strings.Contains(query, "ORDER BY p.fecha_creacion DESC, p.id DESC") || len(args) != 3 {
		t.Fatalf("query/args inesperados: %s %#v", query, args)
	}
}

func TestBuildPagosQueryConOrdenPaginaPorDesplazamiento(t *testing.T) {
	query, args := buildPagosQuery(repository.FiltroPagos{
		OrdenarPor: "cliente_nombre", PageLimit: 51, Offset: 50,
		// Un cursor por valor no debe mezclarse con un orden pedido.
		CursorSet: true, CursorID: 7,
	})
	if strings.Contains(query, "(p.fecha_creacion, p.id) <") {
		t.Fatalf("con orden no se usa el cursor por valor: %s", query)
	}
	if !strings.Contains(query, "LIMIT $1 OFFSET $2") || len(args) != 2 || args[0] != 51 || args[1] != 50 {
		t.Fatalf("limit/offset inesperados: %s %#v", query, args)
	}
	if !strings.Contains(query, "ORDER BY TRANSLATE(LOWER(p.cliente_nombre)") {
		t.Fatalf("ORDER BY inesperado: %s", query)
	}
}

func TestBuildPagosQueryPrimeraPaginaOrdenadaNoLlevaOffset(t *testing.T) {
	query, args := buildPagosQuery(repository.FiltroPagos{OrdenarPor: "total_final", OrdenDesc: true, PageLimit: 51})
	if strings.Contains(query, "OFFSET") || len(args) != 1 {
		t.Fatalf("la primera pagina no lleva OFFSET: %s %#v", query, args)
	}
}
