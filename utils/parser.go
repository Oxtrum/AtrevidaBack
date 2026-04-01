package utils

import (
	"strings"

	"atrevida-agenda-api/models"
)

func ParseCelda(celda string) []models.ReservaItem {
	celda = strings.TrimSpace(celda)

	if strings.Contains(strings.ToUpper(celda), "FERIADO") {
		return []models.ReservaItem{
			{
				Tipo: "feriado",
			},
		}
	}

	celda = strings.ReplaceAll(celda, "\r\n", "\n")
	lineas := strings.Split(celda, "\n")

	var resultado []models.ReservaItem

	for _, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if linea == "" {
			continue
		}

		if !strings.Contains(linea, "-") {
			continue
		}

		partes := strings.SplitN(linea, "-", 2)

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
		if len(partes) > 1 {
			cliente = strings.TrimSpace(partes[1])
		}

		resultado = append(resultado, models.ReservaItem{
			Tipo:    tipo,
			Cliente: cliente,
		})
	}

	return resultado
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

		items := ParseCelda(celda)
		slot.Dias[dia] = items
	}

	return slot, true
}
