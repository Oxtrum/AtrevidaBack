package pgsql

import (
	"strings"
	"testing"
	"time"

	repository "atrevida-agenda-api/repositories"
)

func TestPagoConditionsIncluyeBusquedaCombinada(t *testing.T) {
	conditions, args := pagoConditions(repository.FiltroPagos{
		Busqueda:    "María 123",
		LocalNombre: "PASEO ARANJUEZ",
	})
	where := strings.Join(conditions, " AND ")

	for _, fragment := range []string{"CONCAT_WS", "p.codigo_pago", "p.cliente_nombre", "p.cliente_nit", "p.nombre_cajero", "TRANSLATE"} {
		if !strings.Contains(where, fragment) {
			t.Fatalf("WHERE de pagos no contiene %q: %s", fragment, where)
		}
	}
	if len(args) != 2 || args[0] != "%María 123%" || args[1] != "%PASEO ARANJUEZ%" {
		t.Fatalf("args inesperados: %#v", args)
	}
}

func TestPlanConditionsIncluyeBusquedaCombinada(t *testing.T) {
	conditions, args := planConditions(repository.FiltroPlanes{
		Busqueda: "relajación",
		Estado:   "ACTIVO",
	})
	where := strings.Join(conditions, " AND ")

	for _, fragment := range []string{"CONCAT_WS", "p.codigo", "p.cliente_nombre_texto", "p.combo_nombre_texto", "p.estado_cobranza", "TRANSLATE"} {
		if !strings.Contains(where, fragment) {
			t.Fatalf("WHERE de planes no contiene %q: %s", fragment, where)
		}
	}
	if len(args) != 2 || args[0] != "%relajación%" || args[1] != "ACTIVO" {
		t.Fatalf("args inesperados: %#v", args)
	}
}

func TestPlanOrderClausePriorizaEstadoAntesDePaginar(t *testing.T) {
	legacy := planOrderClause(repository.FiltroPlanes{})
	if legacy != "p.creado_en DESC, p.id DESC" {
		t.Fatalf("orden legacy inesperado: %s", legacy)
	}

	priority := planOrderClause(repository.FiltroPlanes{OrdenPrioridadEstado: true})
	for _, fragment := range []string{"CASE p.estado", "'RESERVADO' THEN 0", "'ACTIVO' THEN 1", "p.creado_en DESC", "p.id DESC"} {
		if !strings.Contains(priority, fragment) {
			t.Fatalf("orden prioritario no contiene %q: %s", fragment, priority)
		}
	}
}

func TestPlanPriorityCursorContinuaConOrdenMixto(t *testing.T) {
	f := repository.FiltroPlanes{
		OrdenPrioridadEstado: true,
		CursorSet:            true,
		CursorEstadoRank:     1,
		CursorFecha:          time.Date(2026, time.August, 12, 18, 0, 0, 0, time.UTC),
		CursorID:             42,
	}
	condition, args, nextIndex := planCursorCondition(f, 3)
	for _, fragment := range []string{"CASE p.estado", "> $3", "= $3", "(p.creado_en, p.id) < ($4, $5)"} {
		if !strings.Contains(condition, fragment) {
			t.Fatalf("cursor prioritario no contiene %q: %s", fragment, condition)
		}
	}
	if len(args) != 3 || args[0] != 1 || args[1] != f.CursorFecha || args[2] != 42 || nextIndex != 6 {
		t.Fatalf("argumentos del cursor inesperados: %#v, next=%d", args, nextIndex)
	}
}
