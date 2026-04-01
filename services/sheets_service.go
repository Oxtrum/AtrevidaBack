package services

import (
	"context"
	"log"
	"strings"

	"atrevida-agenda-api/config"
	"atrevida-agenda-api/models"
	"atrevida-agenda-api/utils"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// --- CONFIG LECTURA ---
const blocksRange = "!A1:G200" // Espacio de lectura

// --- SERVICIOS ---

func GetAllReservas() []models.LocalReservas {
	var resultado []models.LocalReservas

	for _, sheetName := range config.App.SheetsDisponibles {
		data := GetSheetData(sheetName)

		semanas := ParseSemanas(data)

		resultado = append(resultado, models.LocalReservas{
			Local:   sheetName,
			Semanas: semanas,
		})
	}

	return resultado
}

func GetSheetData(sheetName string) [][]interface{} {
	ctx := context.Background()

	srv, err := sheets.NewService(ctx,
		option.WithCredentialsFile(config.App.CredentialsPath),
	)
	if err != nil {
		log.Fatalf("error al crear servicio de Sheets: %v", err)
	}

	readRange := sheetName + blocksRange

	resp, err := srv.Spreadsheets.Values.Get(config.App.SpreadsheetID, readRange).Do()
	if err != nil {
		log.Fatalf("error al leer sheet '%s': %v", sheetName, err)
	}

	return resp.Values
}

func ParseReservas(data [][]interface{}) []models.ReservaSlot {
	var resultado []models.ReservaSlot

	headers := data[0]

	for i := 1; i < len(data); i++ {
		fila := data[i]

		hora, ok := fila[0].(string)
		if !ok {
			continue
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

			celda, ok := fila[j].(string)
			if !ok {
				continue
			}

			items := utils.ParseCelda(celda)

			slot.Dias[dia] = items
		}

		resultado = append(resultado, slot)
	}

	return resultado
}

func ParseSemanas(data [][]interface{}) []models.Semana {
	var semanas []models.Semana

	var semanaActual *models.Semana
	var headers []interface{}

	for _, fila := range data {
		if len(fila) == 0 {
			continue
		}

		val, ok := fila[0].(string)
		if !ok {
			continue
		}

		val = strings.TrimSpace(val)

		if strings.Contains(val, "SEMANA") {
			semana := models.Semana{
				Titulo: val,
			}

			semanas = append(semanas, semana)
			semanaActual = &semanas[len(semanas)-1]

			headers = nil
			continue
		}

		if val == "HORAS" {
			headers = fila
			continue
		}

		if semanaActual == nil || headers == nil {
			continue
		}

		slot, ok := utils.ParseFilaSlot(fila, headers)
		if !ok {
			continue
		}

		semanaActual.Reservas = append(semanaActual.Reservas, slot)
	}

	return semanas
}
