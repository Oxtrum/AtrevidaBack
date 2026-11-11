package models

import "testing"

func TestTiposSeguimientoMantienenLaSeparacionOperativa(t *testing.T) {
	casos := []TipoSeguimiento{
		SeguimientoDeduccionPendiente,
		SeguimientoInasistencia,
		SeguimientoCancelacion,
		SeguimientoPendienteCierre,
	}
	vistos := map[TipoSeguimiento]bool{}
	for _, caso := range casos {
		if caso == "" {
			t.Fatal("un tipo de seguimiento no puede ser vacío")
		}
		if vistos[caso] {
			t.Fatalf("tipo de seguimiento repetido: %s", caso)
		}
		vistos[caso] = true
	}
	if vistos["CANCELADA"] || vistos["ATENDIDA"] {
		t.Fatal("seguimiento no debe reutilizar estados de reserva o atención")
	}
}
