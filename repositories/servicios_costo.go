package repository

import (
	"errors"
	"fmt"
	"math"
)

var ErrCostoServicioInvalido = errors.New("costo de servicio invalido")

// NormalizarCostoServicio conserva el costo ausente para clientes anteriores.
// Los servicios variables siempre guardan cero, nunca un precio referencial.
func NormalizarCostoServicio(costo *float64, variable bool) (*float64, error) {
	if costo != nil {
		if math.IsNaN(*costo) || math.IsInf(*costo, 0) || *costo < 0 || *costo > 99999999.99 {
			return nil, fmt.Errorf("%w: costo debe estar entre 0 y 99999999.99", ErrCostoServicioInvalido)
		}
	}
	if variable {
		cero := 0.0
		return &cero, nil
	}
	if costo == nil {
		return nil, nil
	}
	valor := math.Round(*costo*100) / 100
	return &valor, nil
}

// ResolverCostoServicio aplica un PATCH sobre la modalidad actual bloqueada.
func ResolverCostoServicio(variableActual bool, costo *float64, variable *bool) (*float64, error) {
	variableFinal := variableActual
	if variable != nil {
		variableFinal = *variable
		if variableActual && !variableFinal && costo == nil {
			return nil, fmt.Errorf("%w: costo es requerido al pasar de variable a fijo", ErrCostoServicioInvalido)
		}
	}
	return NormalizarCostoServicio(costo, variableFinal)
}
