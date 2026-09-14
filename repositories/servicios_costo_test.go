package repository

import (
	"errors"
	"math"
	"testing"
)

func TestCostoServicioModalidadesYCompatibilidad(t *testing.T) {
	cero, costo, negativo, limite, exceso := 0.0, 125.555, -1.0, 99999999.99, 100000000.0
	nan, infinito := math.NaN(), math.Inf(1)
	fijo, variable := false, true
	tests := []struct {
		nombre    string
		actual    bool
		costo     *float64
		modalidad *bool
		want      *float64
		invalido  bool
	}{
		{"creacion antigua sin costo", false, nil, nil, nil, false},
		{"precio fijo redondeado", false, &costo, nil, costoServicioTestPtr(125.56), false},
		{"servicio gratuito", false, &cero, nil, &cero, false},
		{"activar variable sin costo", false, nil, &variable, &cero, false},
		{"activar variable descarta referencia", false, &costo, &variable, &cero, false},
		{"cliente antiguo no desactiva variable", true, &costo, nil, &cero, false},
		{"variable existente conserva cero", true, nil, &variable, &cero, false},
		{"volver a fijo exige costo", true, nil, &fijo, nil, true},
		{"volver a fijo con costo", true, &costo, &fijo, costoServicioTestPtr(125.56), false},
		{"volver a gratuito explicitamente", true, &cero, &fijo, &cero, false},
		{"limite de numeric diez dos", false, &limite, nil, &limite, false},
		{"rechaza negativo", false, &negativo, nil, nil, true},
		{"rechaza exceso", false, &exceso, nil, nil, true},
		{"rechaza NaN", false, &nan, nil, nil, true},
		{"rechaza infinito incluso variable", true, &infinito, nil, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.nombre, func(t *testing.T) {
			got, err := ResolverCostoServicio(tt.actual, tt.costo, tt.modalidad)
			if tt.invalido {
				if !errors.Is(err, ErrCostoServicioInvalido) {
					t.Fatalf("error = %v, want costo invalido", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.want == nil {
				if got != nil {
					t.Fatalf("costo = %v, want nil", *got)
				}
			} else if got == nil || *got != *tt.want {
				t.Fatalf("costo = %v, want %v", got, *tt.want)
			}
		})
	}
}

func costoServicioTestPtr(value float64) *float64 { return &value }
