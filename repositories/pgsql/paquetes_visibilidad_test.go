package pgsql

import (
	"strings"
	"testing"

	repository "atrevida-agenda-api/repositories"
)

func TestPaqueteConditions_SoloVisiblesPublico(t *testing.T) {
	conds, args := paqueteConditions(repository.FiltroPaquetes{SoloVisiblesPublico: true})
	if !strings.Contains(strings.Join(conds, " AND "), "p.visible_publico = TRUE") {
		t.Fatalf("falta el filtro de visibilidad publica: %v", conds)
	}
	if len(args) != 0 {
		t.Fatalf("el filtro de visibilidad no lleva argumentos, got %v", args)
	}
}

func TestPaqueteConditions_AdminVeTodos(t *testing.T) {
	conds, _ := paqueteConditions(repository.FiltroPaquetes{})
	if strings.Contains(strings.Join(conds, " AND "), "visible_publico") {
		t.Fatalf("sin SoloVisiblesPublico no se debe filtrar por visibilidad: %v", conds)
	}
}
