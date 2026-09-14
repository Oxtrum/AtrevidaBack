package pgsql

import "testing"

// Los costos de referencia variables se materializan como cero, no como NULL.
// No debe cambiar la validacion ni el calculo existente de POR_ITEMS.
func TestComboReferenciaVariableCeroConservaSesiones(t *testing.T) {
	cero, fijo := 0.0, 70.0
	servicios := []servicioMaterializado{
		{ServicioTexto: "Variable", Costo: &cero, Sesiones: 3},
		{ServicioTexto: "Fijo", Costo: &fijo, Sesiones: 2},
	}
	if err := validarPrecioPorItems("POR_ITEMS", servicios); err != nil {
		t.Fatal(err)
	}
	importe, sesiones := resumenServicios(servicios)
	if importe != 140 || sesiones != 5 {
		t.Fatalf("resumen = (%v, %v), esperado (140, 5)", importe, sesiones)
	}
}

func TestComboPrecioPaqueteNoExigeCostoIndividual(t *testing.T) {
	servicios := []servicioMaterializado{{ServicioTexto: "Variable", Sesiones: 2}}
	if err := validarPrecioPorItems("PRECIO_PAQUETE", servicios); err != nil {
		t.Fatal(err)
	}
	if err := validarPrecioPorItems("POR_ITEMS", servicios); err == nil {
		t.Fatal("un costo ausente no equivale a cero para POR_ITEMS")
	}
}
