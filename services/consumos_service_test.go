package services

import (
	"testing"
	"time"
)

func TestNormalizarEstadoConsumo(t *testing.T) {
	casos := map[string]string{
		"":                "",
		"todos":           "",
		"descontado":      "DESCONTADO",
		"PENDIENTE":       "PENDIENTE",
		"pendiente_stock": "PENDIENTE",
	}
	for entrada, esperado := range casos {
		got, err := normalizarEstadoConsumo(entrada)
		if err != nil {
			t.Fatalf("estado %q devolvio error: %v", entrada, err)
		}
		if got != esperado {
			t.Fatalf("estado %q: se esperaba %q, se obtuvo %q", entrada, esperado, got)
		}
	}
	if _, err := normalizarEstadoConsumo("OTRO"); err == nil {
		t.Fatal("se esperaba error para un estado desconocido")
	}
}

func TestNormalizarRangoConsumos(t *testing.T) {
	hoy := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	desde := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	hasta := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)

	inicio, fin, err := normalizarRangoConsumos(nil, nil, hoy)
	if err != nil || !inicio.Equal(hoy) || !fin.Equal(hoy) {
		t.Fatalf("sin fechas se esperaba hoy/hoy, se obtuvo %v/%v (%v)", inicio, fin, err)
	}
	inicio, fin, err = normalizarRangoConsumos(&desde, nil, hoy)
	if err != nil || !inicio.Equal(desde) || !fin.Equal(desde) {
		t.Fatalf("solo desde se esperaba desde/desde, se obtuvo %v/%v (%v)", inicio, fin, err)
	}
	inicio, fin, err = normalizarRangoConsumos(nil, &hasta, hoy)
	if err != nil || !inicio.Equal(hasta) || !fin.Equal(hasta) {
		t.Fatalf("solo hasta se esperaba hasta/hasta, se obtuvo %v/%v (%v)", inicio, fin, err)
	}
	inicio, fin, err = normalizarRangoConsumos(&desde, &hasta, hoy)
	if err != nil || !inicio.Equal(desde) || !fin.Equal(hasta) {
		t.Fatalf("rango completo se esperaba desde/hasta, se obtuvo %v/%v (%v)", inicio, fin, err)
	}
	if _, _, err = normalizarRangoConsumos(&hasta, &desde, hoy); err == nil {
		t.Fatal("se esperaba error para un rango invertido")
	}
}
