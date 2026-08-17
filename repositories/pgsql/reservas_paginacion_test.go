package pgsql

import (
	"strings"
	"testing"
	"time"

	repository "atrevida-agenda-api/repositories"
)

func TestReservaConditionsIncluyeBusquedaYVigenciaAntesDePaginar(t *testing.T) {
	fecha := time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC)
	conditions, args := reservaConditions(repository.FiltroReservasPG{
		Busqueda:           "Maria",
		ExcluirEstado:      "COMPLETADO",
		SoloActivas:        true,
		VigenteFecha:       &fecha,
		VigenteHora:        "15:30",
		VigenciaPendientes: true,
	})
	where := strings.Join(conditions, " AND ")
	for _, fragment := range []string{
		"CONCAT_WS",
		"r.estado <> $2",
		"r.activo = TRUE",
		"r.estado <> 'PENDIENTE' OR",
		"r.hora_hasta >= $4",
	} {
		if !strings.Contains(where, fragment) {
			t.Fatalf("WHERE no contiene %q: %s", fragment, where)
		}
	}
	if len(args) != 4 || args[0] != "%Maria%" || args[1] != "COMPLETADO" || args[3] != "15:30" {
		t.Fatalf("args inesperados: %#v", args)
	}
}

func TestReservaOrderClauseConservaLegacyYPermiteCronologico(t *testing.T) {
	legacy := reservaOrderClause(repository.FiltroReservasPG{})
	if legacy != "r.local_nombre, r.fecha, r.hora_desde, r.id" {
		t.Fatalf("orden legacy = %q", legacy)
	}
	cronologico := reservaOrderClause(repository.FiltroReservasPG{OrdenCronologico: true})
	if cronologico != "r.fecha, r.hora_desde, r.local_nombre, r.id" {
		t.Fatalf("orden cronologico = %q", cronologico)
	}
}
