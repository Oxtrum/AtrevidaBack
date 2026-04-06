package utils

import (
	"strings"

	"atrevida-agenda-api/models"
)

func ParseCelda(celda string) []models.ReservaItem {
	celda = strings.TrimSpace(celda)

	if strings.Contains(strings.ToUpper(celda), "FERIADO") {
		return []models.ReservaItem{{Tipo: "feriado"}}
	}

	celda = strings.ReplaceAll(celda, "\r\n", "\n")

	var lineas []string
	if strings.Contains(celda, "\n") {
		lineas = strings.Split(celda, "\n")
	} else {
		lineas = tokenizarLineaUnica(celda)
	}

	var resultado []models.ReservaItem

	for _, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if linea == "" {
			continue
		}

		// normalizar "M -Cliente" → "M - Cliente" (sin espacio tras el guión)
		if strings.Contains(linea, " -") && !strings.Contains(linea, " - ") {
			linea = strings.Replace(linea, " -", " - ", 1)
		}

		if !strings.Contains(linea, " - ") {
			continue
		}

		partes := strings.SplitN(linea, " - ", 2)
		tipoRaw := strings.TrimSpace(partes[0])

		var tipo string
		switch tipoRaw {
		case "M":
			tipo = "mesa"
		case "B":
			tipo = "bicicleta"
		default:
			continue
		}

		cliente := ""
		servicio := ""

		if len(partes) > 1 {
			resto := strings.TrimSpace(partes[1])
			cliente, servicio = extraerServicio(resto)
		}

		resultado = append(resultado, models.ReservaItem{
			Tipo:     tipo,
			Cliente:  cliente,
			Servicio: servicio,
		})
	}

	return resultado
}

// extraerServicio separa el nombre del cliente del servicio entre paréntesis.
// "Carlos Rios (meso)"  → "Carlos Rios", "meso"
// "Carlos Rios"         → "Carlos Rios", ""
// ""                    → "", ""
func extraerServicio(s string) (cliente, servicio string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}

	if idx := strings.LastIndex(s, "("); idx != -1 && strings.HasSuffix(s, ")") {
		servicio = strings.TrimSpace(s[idx+1 : len(s)-1])
		cliente = strings.TrimSpace(s[:idx])
		return
	}

	return s, ""
}

// tokenizarLineaUnica divide una celda de una sola línea en sus items.
// "M - Carlos (meso) M - B -" → ["M - Carlos (meso)", "M - ", "B -"]
func tokenizarLineaUnica(celda string) []string {
	var resultado []string
	resto := celda

	for len(resto) > 0 {
		resto = strings.TrimSpace(resto)
		if len(resto) < 3 {
			break
		}

		tipo := string(resto[0])
		if tipo != "M" && tipo != "B" {
			break
		}

		siguiente := encontrarSiguienteItem(resto[1:])
		if siguiente == -1 {
			resultado = append(resultado, strings.TrimSpace(resto))
			break
		}

		corte := siguiente + 1 // +1 por buscar desde resto[1:]
		resultado = append(resultado, strings.TrimSpace(resto[:corte]))
		resto = resto[corte:]
	}

	return resultado
}

// encontrarSiguienteItem busca la posición del próximo "M -" o "B -" en el string.
func encontrarSiguienteItem(s string) int {
	for i := 0; i < len(s)-2; i++ {
		c := s[i]
		if (c == 'M' || c == 'B') && s[i+1] == ' ' && s[i+2] == '-' {
			if i == 0 || s[i-1] == ' ' {
				return i
			}
		}
	}
	return -1
}

func ParseFilaSlot(fila []interface{}, headers []interface{}) (models.ReservaSlot, bool) {
	if len(fila) == 0 {
		return models.ReservaSlot{}, false
	}

	hora, ok := fila[0].(string)
	if !ok || strings.TrimSpace(hora) == "" {
		return models.ReservaSlot{}, false
	}

	if hora == "HORAS" {
		return models.ReservaSlot{}, false
	}

	slot := models.ReservaSlot{
		Hora: hora,
		Dias: make(map[string][]models.ReservaItem),
	}

	for j := 1; j < len(fila) && j < len(headers); j++ {
		dia, ok := headers[j].(string)
		if !ok {
			continue
		}

		celda := ""
		if val, ok := fila[j].(string); ok {
			celda = val
		} else {
			continue
		}

		slot.Dias[dia] = ParseCelda(celda)
	}

	return slot, true
}
