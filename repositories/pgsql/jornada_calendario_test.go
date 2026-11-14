package pgsql

import (
	"strings"
	"testing"
	"time"

	repository "atrevida-agenda-api/repositories"
)

func TestJornadaWhereDistingueDiaActualConFechaUTC(t *testing.T) {
	loc := time.FixedZone("America/La_Paz", -4*60*60)
	ahora := time.Date(2026, time.November, 11, 10, 0, 0, 0, loc)
	fechaUTC := time.Date(2026, time.November, 11, 0, 0, 0, 0, time.UTC)

	where, args := jornadaWhere(repository.FiltroJornada{Fecha: fechaUTC, Estado: "PROGRAMADA", Ahora: ahora})
	if strings.Contains(where, "FALSE") {
		t.Fatalf("la fecha local de hoy no debe descartarse: %s", where)
	}
	if !strings.Contains(where, "r.hora_hasta >") || len(args) != 2 {
		t.Fatalf("se esperaba comparación por hora para el día actual: where=%s args=%v", where, args)
	}
}

func TestCalendarioPersonalWhereIncluyeTodosLosFiltros(t *testing.T) {
	desde := time.Date(2026, time.November, 9, 0, 0, 0, 0, time.UTC)
	hasta := time.Date(2026, time.November, 14, 0, 0, 0, 0, time.UTC)
	local := 3
	where, args := calendarioPersonalWhere(repository.FiltroCalendarioPersonal{
		FechaDesde: desde, FechaHasta: hasta, LocalID: &local,
		Cliente: "Maria", Servicio: "Masaje", Empleado: "Ana", Q: "cabina",
	})
	for _, fragmento := range []string{"a.fecha_atencion >=", "a.fecha_atencion <=", "a.local_id", "r.cliente", "a.empleado_nombre", "CAST(a.reserva_id AS TEXT)"} {
		if !strings.Contains(where, fragmento) {
			t.Fatalf("falta filtro %q en %s", fragmento, where)
		}
	}
	if len(args) != 7 {
		t.Fatalf("args = %d; se esperaban 7", len(args))
	}
}
