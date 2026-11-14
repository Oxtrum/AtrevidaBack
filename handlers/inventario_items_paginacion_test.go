package handlers

import "testing"

// El listado de items sigue devolviendo todo (hasta 500) a quien manda un limit grande o ninguno; solo pagina con
// cursor o con un limit que cabe en una pagina (1 a 100).
func TestItemsSePaginanSoloConCursorOLimitDePagina(t *testing.T) {
	casos := []struct {
		limit, cursor string
		quiero        bool
	}{
		{"", "", false},
		{"200", "", false},
		{"500", "", false},
		{"101", "", false},
		{"100", "", true},
		{"50", "", true},
		{"1", "", true},
		{"", "abc", true},
		{"500", "abc", true},
		{"0", "", true}, // limite invalido: lo rechaza el paginador con 400
		{"x", "", true},
	}
	for _, c := range casos {
		if got := itemsSePaginan(c.limit, c.cursor); got != c.quiero {
			t.Errorf("limit=%q cursor=%q: itemsSePaginan = %v, quiero %v", c.limit, c.cursor, got, c.quiero)
		}
	}
}
