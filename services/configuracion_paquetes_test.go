package services

import "testing"

func TestSesionPaqueteEnEstadoPorDefectoEsCompletado(t *testing.T) {
	if got := configPorDefecto().SesionPaqueteEnEstado; got != "COMPLETADO" {
		t.Fatalf("el default es COMPLETADO, obtuvo %q", got)
	}
}

func TestValidarPatchAceptaSesionPaqueteEnEstado(t *testing.T) {
	for entrada, esperado := range map[string]string{"COMPLETADO": "COMPLETADO", "AGENDADO": "AGENDADO", "agendado": "AGENDADO"} {
		valores, err := validarPatchConfiguracion(map[string]interface{}{CfgSesionPaqueteEnEstado: entrada})
		if err != nil || valores[CfgSesionPaqueteEnEstado] != esperado {
			t.Fatalf("%q: valores=%v err=%v", entrada, valores, err)
		}
	}
}

func TestValidarPatchRechazaSesionPaqueteEnEstadoInvalido(t *testing.T) {
	for _, v := range []interface{}{"PENDIENTE", "RECHAZADO", "", 1.0, true} {
		if _, err := validarPatchConfiguracion(map[string]interface{}{CfgSesionPaqueteEnEstado: v}); err == nil {
			t.Fatalf("%v debe rechazarse", v)
		}
	}
}

func TestUsarPaqueteDesdePorDefectoEsCuenta(t *testing.T) {
	if got := configPorDefecto().UsarPaqueteDesde; got != "CUENTA" {
		t.Fatalf("el default es CUENTA (se usa apenas se vende), obtuvo %q", got)
	}
}

func TestValidarPatchAceptaUsarPaqueteDesde(t *testing.T) {
	for entrada, esperado := range map[string]string{"CUENTA": "CUENTA", "PAGADO": "PAGADO", "pagado": "PAGADO"} {
		valores, err := validarPatchConfiguracion(map[string]interface{}{CfgUsarPaqueteDesde: entrada})
		if err != nil || valores[CfgUsarPaqueteDesde] != esperado {
			t.Fatalf("%q: valores=%v err=%v", entrada, valores, err)
		}
	}
}

func TestValidarPatchRechazaUsarPaqueteDesdeInvalido(t *testing.T) {
	for _, v := range []interface{}{"AGENDADO", "", 1.0, true} {
		if _, err := validarPatchConfiguracion(map[string]interface{}{CfgUsarPaqueteDesde: v}); err == nil {
			t.Fatalf("%v debe rechazarse", v)
		}
	}
}
