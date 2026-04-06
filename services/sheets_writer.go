package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"atrevida-agenda-api/config"
	"atrevida-agenda-api/models"
	"atrevida-agenda-api/utils"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Lectura y escritura de celdas

// GetCeldaRaw lee el contenido crudo de una celda
func GetCeldaRaw(sheetName string, a1Range string) (string, error) {
	ctx := context.Background()
	srv, err := newSheetsService(ctx)
	if err != nil {
		return "", err
	}

	fullRange := sheetName + "!" + a1Range
	resp, err := srv.Spreadsheets.Values.Get(config.App.SpreadsheetID, fullRange).Do()
	if err != nil {
		return "", fmt.Errorf("error al leer celda %s: %w", fullRange, err)
	}

	if len(resp.Values) == 0 || len(resp.Values[0]) == 0 {
		return "", nil
	}

	val, _ := resp.Values[0][0].(string)
	return val, nil
}

// WriteCelda escribe el contenido de una celda en el shee
func WriteCelda(sheetName string, a1Range string, contenido string) error {
	ctx := context.Background()
	srv, err := newSheetsService(ctx)
	if err != nil {
		return err
	}

	fullRange := sheetName + "!" + a1Range
	vr := &sheets.ValueRange{
		Values: [][]interface{}{{contenido}},
	}

	_, err = srv.Spreadsheets.Values.Update(config.App.SpreadsheetID, fullRange, vr).
		ValueInputOption("RAW").
		Do()
	if err != nil {
		return fmt.Errorf("error al escribir celda %s: %w", fullRange, err)
	}

	return nil
}

// Reconstrucción de celda

// ReconstruirCelda convierte una lista de ReservaItem al formato de texto del sheet
func ReconstruirCelda(items []models.ReservaItem) string {
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

func tipoToLetra(tipo string) string {
	switch strings.ToLower(tipo) {
	case "mesa":
		return "M"
	case "bicicleta":
		return "B"
	}
	return ""
}

// Validaciones de slot

// TiposDisponibles retorna cuántos espacios libres hay por letra (M, B).
func TiposDisponibles(items []models.ReservaItem) map[string]int {
	libres := map[string]int{}
	for _, item := range items {
		letra := tipoToLetra(item.Tipo)
		if letra != "" && item.Cliente == "" {
			libres[letra]++
		}
	}
	return libres
}

// TiposExistentes retorna las letras que existen en el slot (libres u ocupados).
func TiposExistentes(items []models.ReservaItem) map[string]bool {
	tipos := map[string]bool{}
	for _, item := range items {
		letra := tipoToLetra(item.Tipo)
		if letra != "" {
			tipos[letra] = true
		}
	}
	return tipos
}

// ResolverCoordenada busca en el sheet la celda (A1) correspondiente a semana+día+hora.
func ResolverCoordenada(sheetName string, semana string, dia string, hora string) (string, error) {
	data := GetSheetData(sheetName)

	enSemana := false
	headersRow := []interface{}{}

	for rowIdx, fila := range data {
		if len(fila) == 0 {
			continue
		}

		val, ok := fila[0].(string)
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)

		if strings.Contains(strings.ToUpper(val), "SEMANA") {
			enSemana = strings.Contains(strings.ToUpper(val), strings.ToUpper(semana))
			headersRow = []interface{}{}
			continue
		}

		if !enSemana {
			continue
		}

		if val == "HORAS" {
			headersRow = fila
			continue
		}

		if len(headersRow) == 0 {
			continue
		}

		horaSlot := strings.TrimSpace(val)
		horaBuscar := strings.TrimSpace(hora)
		horaInicioSlot := strings.TrimSpace(strings.SplitN(horaSlot, " a ", 2)[0])
		if !strings.EqualFold(horaSlot, horaBuscar) && !strings.EqualFold(horaInicioSlot, horaBuscar) {
			continue
		}

		for colIdx, header := range headersRow {
			h, ok := header.(string)
			if !ok {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(h), strings.TrimSpace(dia)) {
				sheetRow := rowIdx + 1
				sheetCol := colIdxToLetter(colIdx)
				return fmt.Sprintf("%s%d", sheetCol, sheetRow), nil
			}
		}

		return "", fmt.Errorf("día '%s' no encontrado en la semana '%s'", dia, semana)
	}

	return "", fmt.Errorf("hora '%s' no encontrada en semana '%s' del local '%s'", hora, semana, sheetName)
}

func colIdxToLetter(idx int) string {
	result := ""
	idx++
	for idx > 0 {
		idx--
		result = string(rune('A'+idx%26)) + result
		idx /= 26
	}
	return result
}

func newSheetsService(ctx context.Context) (*sheets.Service, error) {
	srv, err := sheets.NewService(ctx,
		option.WithCredentialsFile(config.App.CredentialsPath),
	)
	if err != nil {
		log.Printf("error al crear servicio de Sheets: %v", err)
		return nil, fmt.Errorf("error al conectar con Google Sheets: %w", err)
	}
	return srv, nil
}

// ParseSlotCelda delega al parser existente de utils.
func ParseSlotCelda(raw string) []models.ReservaItem {
	return utils.ParseCelda(raw)
}
