package pgsql

import (
	"strings"
	"testing"
	"time"

	repository "atrevida-agenda-api/repositories"
)

func TestBuildConsumosEventosQueryAplicaFiltros(t *testing.T) {
	itemID := 14
	localID := 1
	desde := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	hasta := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	query, args := buildConsumosEventosQuery(repository.FiltroConsumos{
		LocalID: &localID, FechaDesde: desde, FechaHasta: hasta, ItemID: &itemID,
		EstadoDescuento: "PENDIENTE", Q: "maria", PageLimit: 51, Offset: 50,
	})
	for _, fragmento := range []string{
		"a.activo = TRUE",
		"a.fecha_atencion >= $1",
		"a.fecha_atencion <= $2",
		"a.local_id = $3",
		"ai.item_id = $4",
		"ai.estado_descuento <> 'DESCONTADO'",
		"ORDER BY a.fecha_atencion ASC, COALESCE(a.hora_inicio, r.hora_desde) ASC NULLS LAST, a.id ASC",
		"LIMIT $6 OFFSET $7",
	} {
		if !strings.Contains(query, fragmento) {
			t.Fatalf("la consulta no contiene %q:\n%s", fragmento, query)
		}
	}
	if len(args) != 7 {
		t.Fatalf("se esperaban 7 argumentos, se obtuvieron %d", len(args))
	}
}

func TestBuildConsumosResumenQueryFiltraLineas(t *testing.T) {
	itemID := 15
	desde := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	hasta := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)
	query, args := buildConsumosResumenQuery(repository.FiltroConsumos{
		FechaDesde: desde, FechaHasta: hasta, ItemID: &itemID, EstadoDescuento: "DESCONTADO",
	})
	for _, fragmento := range []string{
		"ai.item_id = $3",
		"ai.estado_descuento = 'DESCONTADO'",
		"SUM(GREATEST(ai.cantidad_usada - ai.cantidad_descontada, 0))",
		"COUNT(DISTINCT ai.atencion_id)",
		"GROUP BY ai.item_id, ai.item_nombre, ai.unidad_medida",
	} {
		if !strings.Contains(query, fragmento) {
			t.Fatalf("la consulta no contiene %q:\n%s", fragmento, query)
		}
	}
	if len(args) != 3 {
		t.Fatalf("se esperaban 3 argumentos, se obtuvieron %d", len(args))
	}
}

func TestBuildConsumosEmpleadosQueryAgrupaPorEmpleado(t *testing.T) {
	empleadoID := 4
	localID := 2
	desde := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	hasta := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)
	query, args := buildConsumosEmpleadosQuery(repository.FiltroConsumos{
		LocalID: &localID, FechaDesde: desde, FechaHasta: hasta,
		EmpleadoID: &empleadoID, EstadoDescuento: "PENDIENTE",
	})
	for _, fragmento := range []string{
		"a.local_id = $3",
		"a.empleado_id = $4",
		"ai.estado_descuento <> 'DESCONTADO'",
		"COUNT(DISTINCT a.id) AS atenciones",
		"COUNT(ai.id) AS insumos_registrados",
		"COUNT(DISTINCT (ai.item_id, ai.item_nombre, ai.unidad_medida)) FILTER (WHERE ai.id IS NOT NULL)",
		"COUNT(ai.id) FILTER (WHERE ai.estado_descuento <> 'DESCONTADO')",
		"COUNT(DISTINCT a.id) FILTER (WHERE ai.estado_descuento <> 'DESCONTADO')",
		"GROUP BY a.empleado_id, CASE WHEN a.empleado_id IS NULL THEN a.empleado_nombre ELSE '' END",
		"ORDER BY empleado_nombre ASC",
	} {
		if !strings.Contains(query, fragmento) {
			t.Fatalf("la consulta no contiene %q:\n%s", fragmento, query)
		}
	}
	if len(args) != 4 {
		t.Fatalf("se esperaban 4 argumentos, se obtuvieron %d", len(args))
	}
}
