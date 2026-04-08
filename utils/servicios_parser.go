package utils

import (
	"math"
	"strings"

	"atrevida-agenda-api/models"
)

// Se identifican cabeceras presentes
var headersSecciones = map[string]bool{
	"TIEMPO": true,
	"COSTO":  true,
	"COSTOS": true,
}

// localesConocidos, idenitifica celdas de locales
var localesConocidos = map[string]bool{
	"ARANJUEZ":          true,
	"CENTRO":            true,
	"ARANJUEZ + CENTRO": true,
}

const MaxFilaServicios = 77

// Parseo de formato de hoja de servicios
func ParseServiciosSheet(data [][]interface{}, maxFila int) []models.ServicioItem {
	var resultado []models.ServicioItem

	categoriaActual := ""
	localActual := ""
	hayColumnasSesiones := false

	for rowIdx, fila := range data {
		filaNum := rowIdx + 1
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

		// Fila de categoría
		if esTituloCategoria(col0, col1, fila) {
			categoriaActual = col0
			localActual = ""
			hayColumnasSesiones = tieneSesiones(fila)
			continue
		}

		// Fila de local
		if esFilaLocal(col0, col1) {
			localActual = extraerLocal(col0)
			hayColumnasSesiones = tieneSesiones(fila)
			continue
		}

		// Fila de headers de columnas (+ sesiones)
		if headersSecciones[strings.ToUpper(col0)] {
			hayColumnasSesiones = tieneSesiones(fila)
			continue
		}

		// Fila de dato real (servicio)
		if categoriaActual == "" || localActual == "" {
			continue
		}

		item := parsearFilaServicio(fila, categoriaActual, localActual, hayColumnasSesiones)
		if item == nil {
			continue
		}

		resultado = append(resultado, *item)
	}

	return resultado
}

//  Clasificadores de fila

func esTituloCategoria(col0, col1 string, fila []interface{}) bool {
	col0Up := strings.ToUpper(col0)
	col1Up := strings.ToUpper(col1)

	// MANEJO DE SESIONES

	// - "SERVICIOS" / "COMBOS" = "1 Sesion"
	if col1Up == "1 SESION" {
		return true
	}

	// - "INYECCIONES TIEMPO COSTOS SESIONES" = Campo sesiones
	if col1Up == "TIEMPO" && tieneSesiones(fila) {
		return true
	}

	// "SERVICIOS Manuales", "LIMPIEZA FACIAL" → solo cuando col1 está vacío
	if col1 == "" {
		titulos := []string{"SERVICIOS MANUALES", "LIMPIEZA FACIAL"}
		for _, t := range titulos {
			if strings.HasPrefix(col0Up, t) {
				return true
			}
		}
	}

	return false
}

func esFilaLocal(col0, col1 string) bool {
	col0Up := strings.ToUpper(strings.TrimSpace(col0))
	col1Up := strings.ToUpper(strings.TrimSpace(col1))

	// Fila header / servicio
	if col1 == "" || col1Up == "TIEMPO" || col1Up == "COSTO" || col1Up == "COSTOS" {
		if localesConocidos[col0Up] {
			return true
		}
		if strings.Contains(col0Up, "APARTOLOGIA") {
			return true
		}
	}
	return false
}

func tieneSesiones(fila []interface{}) bool {
	for _, v := range fila {
		if strings.ToUpper(strings.TrimSpace(toStr(v))) == "SESIONES" {
			return true
		}
	}
	return false
}

// Extracción de datos

func extraerLocal(raw string) string {
	rawUp := strings.ToUpper(strings.TrimSpace(raw))
	if strings.Contains(rawUp, "ARANJUEZ") && strings.Contains(rawUp, "CENTRO") {
		return "ARANJUEZ + CENTRO"
	}
	if strings.Contains(rawUp, "ARANJUEZ") {
		return "ARANJUEZ"
	}
	if strings.Contains(rawUp, "CENTRO") {
		return "CENTRO"
	}
	return strings.ToUpper(raw)
}

func parsearFilaServicio(fila []interface{}, categoria, local string, hayColumnasSesiones bool) *models.ServicioItem {
	nombre := strings.TrimSpace(toStr(fila[0]))
	if nombre == "" {
		return nil
	}

	tiempo := ""
	if len(fila) > 1 {
		tiempo = strings.TrimSpace(toStr(fila[1]))
	}

	costo := ""
	if len(fila) > 2 {
		costo = strings.TrimSpace(toStr(fila[2]))
	}

	sesiones := 1
	if hayColumnasSesiones && len(fila) > 3 && fila[3] != nil {
		switch n := fila[3].(type) {
		case float64:
			sesiones = int(math.Round(n))
		case int:
			sesiones = n
		case int64:
			sesiones = int(n)
		}
	}

	return &models.ServicioItem{
		Nombre:    nombre,
		Categoria: categoria,
		Local:     local,
		Tiempo:    tiempo,
		Costo:     costo,
		Sesiones:  sesiones,
	}
}

func toStr(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
