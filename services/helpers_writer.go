package services

import (
	"fmt"
	"strings"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/utils"
)

// parseSlotCelda delega al parser existente de utils.
func parseSlotCelda(raw string) []models.ReservaItem {
	return utils.ParseCelda(raw)
}

// reconstruirCelda convierte una lista de ReservaItem al formato de texto del sheet.
func reconstruirCelda(items []models.ReservaItem) string {
	var lineas []string
	for _, item := range items {
		letra := tipoToLetra(item.Tipo)
		if letra == "" {
			continue
		}
		if item.Cliente == "" {
			lineas = append(lineas, letra+" - ")
		} else if item.Servicio != "" {
			lineas = append(lineas, fmt.Sprintf("%s - %s (%s)", letra, item.Cliente, item.Servicio))
		} else {
			lineas = append(lineas, fmt.Sprintf("%s - %s", letra, item.Cliente))
		}
	}
	return strings.Join(lineas, "\n")
}

// tiposExistentes retorna las letras que existen en el slot (libres u ocupados).
func tiposExistentes(items []models.ReservaItem) map[string]bool {
	tipos := map[string]bool{}
	for _, item := range items {
		letra := tipoToLetra(item.Tipo)
		if letra != "" {
			tipos[letra] = true
		}
	}
	return tipos
}

// letraToTipoNombre convierte "M" → "mesa", "B" → "bicicleta".
func letraToTipoNombre(letra string) string {
	switch strings.ToUpper(letra) {
	case "M":
		return "mesa"
	case "B":
		return "bicicleta"
	}
	return ""
}

// convierte "mesa" → "M", "bicicleta" → "B".
func tipoToLetra(tipo string) string {
	switch strings.ToLower(tipo) {
	case "mesa":
		return "M"
	case "bicicleta":
		return "B"
	}
	return ""
}
