package pgsql

import (
	"sort"
	"strings"
	"testing"
	"time"

	repository "atrevida-agenda-api/repositories"
)

func TestPlanOrderClauseConOrdenPedidoIgnoraLaPrioridadDeEstado(t *testing.T) {
	got := planOrderClause(repository.FiltroPlanes{OrdenarPor: "precio_total", OrdenDesc: true, OrdenPrioridadEstado: true})
	if got != "p.precio_total DESC NULLS LAST, p.id DESC" {
		t.Fatalf("el orden pedido manda sobre la prioridad: %s", got)
	}
}

func TestPlanOrderClauseTextoIgnoraMayusculasYAcentos(t *testing.T) {
	got := planOrderClause(repository.FiltroPlanes{OrdenarPor: "cliente"})
	if !strings.Contains(got, "TRANSLATE(LOWER(p.cliente)") || !strings.HasSuffix(got, "ASC NULLS LAST, p.id ASC") {
		t.Fatalf("orden por cliente inesperado: %s", got)
	}
}

func TestPlanesOrdenablesExponeLasColumnasDeLaTabla(t *testing.T) {
	want := []string{"cliente", "codigo", "combo_nombre_texto", "creado_en", "estado", "estado_cobranza", "local_nombre_texto", "precio_total"}
	got := make([]string, 0, len(PlanesOrdenables))
	for key := range PlanesOrdenables {
		got = append(got, key)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("columnas ordenables = %v, esperadas %v", got, want)
	}
}

func TestBuildPlanesQuerySinOrdenConservaElCursorPrioritario(t *testing.T) {
	query, args := buildPlanesQuery(repository.FiltroPlanes{
		OrdenPrioridadEstado: true, PageLimit: 51, CursorSet: true, CursorEstadoRank: 1,
		CursorFecha: time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC), CursorID: 42,
	})
	if !strings.Contains(query, "(p.creado_en, p.id) < (") || strings.Contains(query, "OFFSET") || len(args) != 4 {
		t.Fatalf("sin orden pedido se conserva el cursor por valor: %s %#v", query, args)
	}
}

func TestBuildPlanesQueryConOrdenPaginaPorDesplazamiento(t *testing.T) {
	query, args := buildPlanesQuery(repository.FiltroPlanes{
		OrdenarPor: "estado", PageLimit: 51, Offset: 50, CursorSet: true, CursorID: 42,
	})
	if strings.Contains(query, "(p.creado_en, p.id) <") {
		t.Fatalf("con orden no se usa el cursor por valor: %s", query)
	}
	if !strings.Contains(query, "LIMIT $1 OFFSET $2") || len(args) != 2 || args[0] != 51 || args[1] != 50 {
		t.Fatalf("limit/offset inesperados: %s %#v", query, args)
	}
	if !strings.Contains(query, "ORDER BY p.estado ASC NULLS LAST, p.id ASC") {
		t.Fatalf("ORDER BY inesperado: %s", query)
	}
}

func TestBuildPlanesQueryPrimeraPaginaOrdenadaNoLlevaOffset(t *testing.T) {
	query, args := buildPlanesQuery(repository.FiltroPlanes{OrdenarPor: "precio_total", PageLimit: 51})
	if strings.Contains(query, "OFFSET") || len(args) != 1 {
		t.Fatalf("la primera pagina no lleva OFFSET: %s %#v", query, args)
	}
}
