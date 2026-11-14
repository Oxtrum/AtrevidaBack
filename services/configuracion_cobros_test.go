package services

import "testing"

func TestDiasMaximoFechaPasadaPorDefectoNoPoneLimite(t *testing.T) {
	if got := configPorDefecto().DiasMaximoFechaPasada; got != 0 {
		t.Fatalf("el default es 0 (sin limite), obtuvo %d", got)
	}
}

func TestValidarPatchAceptaDiasMaximoFechaPasada(t *testing.T) {
	for _, n := range []float64{0, 30, 365} {
		valores, err := validarPatchConfiguracion(map[string]interface{}{CfgDiasMaximoFechaPasada: n})
		if err != nil {
			t.Fatalf("%v: %v", n, err)
		}
		if valores[CfgDiasMaximoFechaPasada] == "" {
			t.Fatalf("%v: no se serializo", n)
		}
	}
}

func TestValidarPatchRechazaDiasMaximoFechaPasadaInvalido(t *testing.T) {
	for _, v := range []interface{}{-1.0, 366.0, 2.5, "30", true} {
		if _, err := validarPatchConfiguracion(map[string]interface{}{CfgDiasMaximoFechaPasada: v}); err == nil {
			t.Fatalf("%v debe rechazarse", v)
		}
	}
}
