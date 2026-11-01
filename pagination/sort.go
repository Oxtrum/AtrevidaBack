package pagination

import (
	"fmt"
	"sort"
	"strings"
)

// Columns mapea la clave publica de una columna ordenable a su expresion SQL.
// Las expresiones son literales del codigo, nunca texto del cliente: la lista blanca es la unica puerta.
type Columns map[string]string

// Sort es el orden pedido por el cliente. El valor cero significa orden por defecto.
type Sort struct {
	Key  string
	Desc bool
}

func (s Sort) Active() bool { return s.Key != "" }

// Direction devuelve asc o desc, para ecoar el orden en los filtros del cursor.
func (s Sort) Direction() string {
	if s.Desc {
		return "desc"
	}
	return "asc"
}

// ParseSort valida ordenar_por y sentido contra la lista blanca. Sin ordenar_por no hay orden pedido.
// El sentido no se llama orden porque algunos listados ya usan ese parametro para otra cosa.
func ParseSort(rawKey, rawDirection string, allowed Columns) (Sort, error) {
	key := strings.TrimSpace(rawKey)
	direction := strings.ToLower(strings.TrimSpace(rawDirection))
	if key == "" {
		if direction != "" {
			return Sort{}, fmt.Errorf("%w: sentido requiere ordenar_por", ErrInvalid)
		}
		return Sort{}, nil
	}
	if _, ok := allowed[key]; !ok {
		return Sort{}, fmt.Errorf("%w: ordenar_por debe ser uno de: %s", ErrInvalid, allowedKeys(allowed))
	}
	switch direction {
	case "", "asc":
		return Sort{Key: key}, nil
	case "desc":
		return Sort{Key: key, Desc: true}, nil
	default:
		return Sort{}, fmt.Errorf("%w: sentido debe ser asc o desc", ErrInvalid)
	}
}

// OrderBy arma el cuerpo del ORDER BY. Los vacios quedan al final y `ties` desempatan (en el mismo
// sentido) para que las paginas por desplazamiento no repitan ni salten filas: el ultimo debe ser unico,
// normalmente el id. Sin orden activo (o con una clave fuera de la lista blanca) devuelve `fallback`.
func (s Sort) OrderBy(allowed Columns, fallback string, ties ...string) string {
	expr, ok := allowed[s.Key]
	if !s.Active() || !ok {
		return fallback
	}
	direction := "ASC"
	if s.Desc {
		direction = "DESC"
	}
	parts := []string{fmt.Sprintf("%s %s NULLS LAST", expr, direction)}
	for _, tie := range ties {
		parts = append(parts, tie+" "+direction)
	}
	return strings.Join(parts, ", ")
}

func allowedKeys(allowed Columns) string {
	keys := make([]string, 0, len(allowed))
	for key := range allowed {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

// OffsetPosition es la posicion del cursor cuando hay un orden pedido: cuantas filas ya se entregaron.
// El orden por columna arbitraria no admite cursor por valor (nulos, empates), asi que se avanza por desplazamiento.
type OffsetPosition struct {
	Offset int `json:"o"`
}

func (p OffsetPosition) Valid() bool { return p.Offset >= 0 }
