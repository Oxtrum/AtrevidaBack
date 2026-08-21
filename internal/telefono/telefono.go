// Package telefono centraliza el parseo de numeros para la transición entre
// formato legacy y E.164. No persiste ni modifica datos por si mismo.
package telefono

import (
	"fmt"
	"strings"

	"github.com/nyaruka/phonenumbers/v2"
)

const DefaultRegion = "BO"

// NormalizeE164 valida una entrada explícitamente internacional y devuelve su
// representación E.164. Un campo telefono_e164 sin '+' es ambiguo y se rechaza.
func NormalizeE164(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("telefono_e164 no puede estar vacio")
	}
	if !strings.HasPrefix(raw, "+") {
		return "", fmt.Errorf("telefono_e164 debe comenzar con '+'")
	}
	return normalize(raw, DefaultRegion)
}

// TryNormalizeLegacy intenta enriquecer un valor histórico sin convertir un
// fallo de parseo en error de compatibilidad para consumidores anteriores.
func TryNormalizeLegacy(raw string) (string, bool) {
	value, err := normalize(raw, DefaultRegion)
	return value, err == nil
}

// ResolveE164 prioriza un valor explícito E.164. Si el consumidor legacy no
// lo envía, intenta enriquecer su número sin convertir un fallo en ruptura de
// compatibilidad.
func ResolveE164(legacy string, explicit *string) (*string, error) {
	if explicit != nil {
		value, err := NormalizeE164(*explicit)
		if err != nil {
			return nil, err
		}
		return &value, nil
	}
	if value, ok := TryNormalizeLegacy(legacy); ok {
		return &value, nil
	}
	return nil, nil
}

func normalize(raw, region string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("telefono vacio")
	}
	number, err := phonenumbers.Parse(raw, region)
	if err != nil || !phonenumbers.IsValidNumber(number) {
		return "", fmt.Errorf("telefono invalido")
	}
	return phonenumbers.Format(number, phonenumbers.E164), nil
}

// DigitsForWhatsApp devuelve la forma internacional sin '+'.
func DigitsForWhatsApp(e164 string) string {
	return strings.TrimPrefix(strings.TrimSpace(e164), "+")
}
