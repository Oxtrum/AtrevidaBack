package pgsql

import "testing"

func TestEsTransicionValida_Reservado(t *testing.T) {
	cases := []struct {
		actual, nuevo string
		want          bool
	}{
		{"RESERVADO", "ACTIVO", true},
		{"RESERVADO", "CANCELADO", true},
		{"RESERVADO", "COMPLETADO", false},
		{"BORRADOR", "ACTIVO", false}, // BORRADOR ya no existe como estado valido
		{"ACTIVO", "COMPLETADO", true},
		{"COMPLETADO", "ACTIVO", false},
	}
	for _, c := range cases {
		if got := esTransicionValida(c.actual, c.nuevo); got != c.want {
			t.Errorf("esTransicionValida(%q,%q)=%v want %v", c.actual, c.nuevo, got, c.want)
		}
	}
}
