package utils

import (
	"math"
	"strings"

	"atrevida-agenda-api/models"
)

const MinFilaCombos = 78
const MaxFilaCombos = 99

// Parseo a formato de combos
func ParseCombosSheet(data [][]interface{}, minFila, maxFila int) []models.ComboItem {
	var combos []models.ComboItem

	catActual := ""
	localActual := ""
	var comboActual *models.ComboItem

	flush := func() {
		if comboActual != nil {
			combos = append(combos, *comboActual)
			comboActual = nil
		}
	}

	for rowIdx, fila := range data {
		filaNum := rowIdx + 1
		if filaNum < minFila {
			continue
		}
		if filaNum > maxFila {
			break
		}

		if len(fila) == 0 {
			continue
		}

		col0 := strings.TrimSpace(toStr(fila[0]))
		if col0 == "" {
			continue
		}

		col1 := ""
		if len(fila) > 1 {
			col1 = strings.TrimSpace(toStr(fila[1]))
		}
		col2 := ""
		if len(fila) > 2 {
			col2 = strings.TrimSpace(toStr(fila[2]))
		}
		var col3 interface{}
		if len(fila) > 3 {
			col3 = fila[3]
		}

		// ── Header de sección (col1 es TIEMPO/COSTOS/SESIONES) ───────────────
		if esComboCatHeader(col1) {
			flush()
			catActual = col0
			continue
		}

		// ── Fila de local ─────────────────────────────────────────────────────
		if esFilaLocalCombo(col0, col1) {
			localActual = extraerLocal(col0)
			continue
		}

		// ── Fila item (tiene tiempo en col1) ─────────────────────────────────
		if tieneTiempo(col1) {
			if comboActual == nil {
				// item suelto sin nombre de combo encima → combo implícito
				comboActual = &models.ComboItem{
					Nombre:          col0,
					Categoria:       catActual,
					Local:           localActual,
					CostoTotal:      col2,
					SesionesTotales: parseSesionesFloat(col3),
				}
			}
			comboActual.ServiciosIncluidos = append(comboActual.ServiciosIncluidos,
				models.ServicioIncluido{
					Nombre:   col0,
					Tiempo:   col1,
					Costo:    col2,
					Sesiones: parseSesionesFloat(col3),
				},
			)
			continue
		}

		// ── Fila nombre de combo (no es header, no es local, no tiene tiempo) ─
		flush()
		comboActual = &models.ComboItem{
			Nombre:          col0,
			Categoria:       catActual,
			Local:           localActual,
			CostoTotal:      col2,
			SesionesTotales: parseSesionesFloat(col3),
		}
	}

	flush()
	return combos
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func esComboCatHeader(col1 string) bool {
	switch strings.ToUpper(col1) {
	case "TIEMPO", "COSTOS", "COSTO", "SESIONES":
		return true
	}
	return false
}

func esFilaLocalCombo(col0, col1 string) bool {
	if col1 != "" {
		return false
	}
	col0Up := strings.ToUpper(strings.TrimSpace(col0))
	if localesConocidos[col0Up] || strings.Contains(col0Up, "APARTOLOGIA") {
		return true
	}
	return false
}

func tieneTiempo(col1 string) bool {
	col1L := strings.ToLower(col1)
	return strings.Contains(col1L, "min") || strings.Contains(col1L, "hora")
}

func parseSesionesFloat(v interface{}) int {
	if v == nil {
		return 1
	}
	switch n := v.(type) {
	case float64:
		r := int(math.Round(n))
		if r < 1 {
			return 1
		}
		return r
	case int:
		if n < 1 {
			return 1
		}
		return n
	case int64:
		if n < 1 {
			return 1
		}
		return int(n)
	}
	return 1
}
