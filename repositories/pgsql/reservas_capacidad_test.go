package pgsql

import (
	"strings"
	"testing"
)

// La validacion de capacidad debe medir la concurrencia maxima en cualquier
// instante dentro del rango solicitado, no contar reservas de sub-bloques
// distintos sobre todo el rango. Dos reservas consecutivas de 30min
// (12:00-12:30 y 12:30-13:00) no deben bloquear una nueva de 60min
// (12:00-13:00) cuando hay capacidad para 2 mesas: cada instante tiene 1
// ocupada de 2.
func TestConsultaMaxConcurrencia_MideInstanteNoRangoCompleto(t *testing.T) {
	sql := consultaMaxConcurrencia()

	for _, frag := range []string{
		"MAX(depth)",
		"r.hora_desde <= t.instant",
		"r.hora_hasta > t.instant",
		"$4::time AS instant",
		"UNION",
	} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("la consulta no contiene %q:\n%s", frag, sql)
		}
	}

	old := "SELECT COUNT(*) FROM reservas\n" +
		"\t\tWHERE local_id = $1 AND tipo_espacio = $2 AND fecha = $3\n" +
		"\t\t  AND activo = TRUE\n" +
		"\t\t  AND hora_desde < $5::time AND hora_hasta > $4::time"
	if strings.Contains(sql, old) {
		t.Fatalf("quedo el conteo antiguo sobre el rango completo:\n%s", sql)
	}
}