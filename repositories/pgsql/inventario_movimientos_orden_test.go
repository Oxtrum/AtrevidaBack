package pgsql

import (
	"sort"
	"strings"
	"testing"
	"time"

	repository "atrevida-agenda-api/repositories"
)

func TestMovimientosOrderClauseSinOrdenConservaFechaDescendente(t *testing.T) {
	if got := movimientosOrderClause(repository.FiltroMovimientosInventario{}); got != "m.creado_en DESC, m.id DESC" {
		t.Fatalf("orden por defecto inesperado: %s", got)
	}
}

func TestMovimientosOrderClauseOrdenaPorColumnaPermitida(t *testing.T) {
	got := movimientosOrderClause(repository.FiltroMovimientosInventario{OrdenarPor: "stock_resultante", OrdenDesc: true})
	if got != "m.stock_resultante DESC NULLS LAST, m.id DESC" {
		t.Fatalf("orden por stock inesperado: %s", got)
	}
	got = movimientosOrderClause(repository.FiltroMovimientosInventario{OrdenarPor: "item_nombre"})
	if !strings.Contains(got, "TRANSLATE(LOWER(i.nombre)") {
		t.Fatalf("el texto debe ignorar mayusculas y acentos: %s", got)
	}
}

func TestMovimientosOrdenablesExponeLasColumnasDeLaTabla(t *testing.T) {
	want := []string{"cantidad", "costo_unitario", "creado_en", "item_nombre", "local_nombre", "motivo", "stock_resultante", "tipo", "usuario_nombre"}
	got := make([]string, 0, len(MovimientosOrdenables))
	for key := range MovimientosOrdenables {
		got = append(got, key)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("columnas ordenables = %v, esperadas %v", got, want)
	}
}

func TestBuildMovimientosQuerySinOrdenUsaCursorPorValor(t *testing.T) {
	query, args := buildMovimientosQuery(repository.FiltroMovimientosInventario{
		PageLimit: 51, CursorSet: true, CursorTime: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), CursorID: 9,
	})
	if !strings.Contains(query, "(m.creado_en, m.id) < (") || strings.Contains(query, "OFFSET") {
		t.Fatalf("sin orden debe paginar por valor: %s", query)
	}
	if !strings.Contains(query, "ORDER BY m.creado_en DESC, m.id DESC") || len(args) != 3 {
		t.Fatalf("query/args inesperados: %s %#v", query, args)
	}
}

func TestBuildMovimientosQueryConOrdenPaginaPorDesplazamiento(t *testing.T) {
	query, args := buildMovimientosQuery(repository.FiltroMovimientosInventario{
		OrdenarPor: "cantidad", PageLimit: 51, Offset: 50, CursorSet: true, CursorID: 9,
	})
	if strings.Contains(query, "(m.creado_en, m.id) <") {
		t.Fatalf("con orden no se usa el cursor por valor: %s", query)
	}
	if !strings.Contains(query, "LIMIT $1 OFFSET $2") || len(args) != 2 || args[0] != 51 || args[1] != 50 {
		t.Fatalf("limit/offset inesperados: %s %#v", query, args)
	}
}

func TestBuildMovimientosQueryOrdenadoSinPaginacionNoLlevaOffset(t *testing.T) {
	query, args := buildMovimientosQuery(repository.FiltroMovimientosInventario{OrdenarPor: "cantidad", Offset: 50})
	if strings.Contains(query, "OFFSET") || len(args) != 1 || args[0] != 200 {
		t.Fatalf("sin paginacion se conserva el limite legacy y no hay OFFSET: %s %#v", query, args)
	}
}

func TestBuildMovimientosQueryConservaLosFiltros(t *testing.T) {
	itemID := 4
	query, args := buildMovimientosQuery(repository.FiltroMovimientosInventario{ItemID: &itemID, Tipo: "AJUSTE_POSITIVO", OrdenarPor: "motivo", PageLimit: 11})
	if !strings.Contains(query, "m.item_id = $1") || !strings.Contains(query, "m.tipo = $2") || len(args) != 3 {
		t.Fatalf("los filtros deben mantenerse: %s %#v", query, args)
	}
}
