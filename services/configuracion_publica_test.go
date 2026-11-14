package services

import "testing"

func TestValidarPatch_MostrarPaquetesAceptaBool(t *testing.T) {
	valores, err := validarPatchConfiguracion(map[string]interface{}{CfgMostrarPaquetes: false})
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if valores[CfgMostrarPaquetes] != "false" {
		t.Fatalf("valor guardado = %q, want %q", valores[CfgMostrarPaquetes], "false")
	}
}

func TestValidarPatch_MostrarPaquetesRechazaNoBool(t *testing.T) {
	if _, err := validarPatchConfiguracion(map[string]interface{}{CfgMostrarPaquetes: "si"}); err == nil {
		t.Fatal("esperaba error al enviar un texto en vez de bool")
	}
}

func TestConfigPorDefecto_MuestraPaquetes(t *testing.T) {
	if !configPorDefecto().MostrarPaquetes {
		t.Fatal("la seccion de paquetes debe verse por defecto")
	}
}
