package sheets

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

const blocksRange = "!A1:G200"

type ReservasRepo struct {
	cfg *config.Config
}

func NewReservasRepo(cfg *config.Config) *ReservasRepo {
	return &ReservasRepo{cfg: cfg}
}

// ── Lectura ───────────────────────────────────────────────────────────────────

func (r *ReservasRepo) GetAllReservas() []models.LocalReservas {
	var resultado []models.LocalReservas

	for _, sheetName := range r.cfg.SheetsDisponibles {
		data := r.GetSheetData(sheetName)
		semanas := r.parseSemanas(data)
		resultado = append(resultado, models.LocalReservas{
			Local:   sheetName,
			Semanas: semanas,
		})
	}

	return resultado
}

func (r *ReservasRepo) GetSheetData(sheetName string) [][]interface{} {
	ctx := context.Background()

	srv, err := r.newService(ctx)
	if err != nil {
		log.Fatalf("error al crear servicio de Sheets: %v", err)
	}

	resp, err := srv.Spreadsheets.Values.
		Get(r.cfg.SpreadsheetID, sheetName+blocksRange).
		Do()
	if err != nil {
		log.Fatalf("error al leer sheet '%s': %v", sheetName, err)
	}

	return resp.Values
}

// ── Escritura ─────────────────────────────────────────────────────────────────

func (r *ReservasRepo) GetCeldaRaw(sheetName, a1Range string) (string, error) {
	ctx := context.Background()
	srv, err := r.newService(ctx)
	if err != nil {
		return "", err
	}

	fullRange := sheetName + "!" + a1Range
	resp, err := srv.Spreadsheets.Values.Get(r.cfg.SpreadsheetID, fullRange).Do()
	if err != nil {
		return "", fmt.Errorf("error al leer celda %s: %w", fullRange, err)
	}

	if len(resp.Values) == 0 || len(resp.Values[0]) == 0 {
		return "", nil
	}

	val, _ := resp.Values[0][0].(string)
	return val, nil
}

func (r *ReservasRepo) WriteCelda(sheetName, a1Range, contenido string) error {
	ctx := context.Background()
	srv, err := r.newService(ctx)
	if err != nil {
		return err
	}

	fullRange := sheetName + "!" + a1Range
	vr := &sheets.ValueRange{
		Values: [][]interface{}{{contenido}},
	}

	_, err = srv.Spreadsheets.Values.
		Update(r.cfg.SpreadsheetID, fullRange, vr).
		ValueInputOption("RAW").
		Do()
	if err != nil {
		return fmt.Errorf("error al escribir celda %s: %w", fullRange, err)
	}

	return nil
}

func (r *ReservasRepo) ResolverCoordenada(sheetName, semana, dia, hora string) (string, error) {
	data := r.GetSheetData(sheetName)

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

// ── Parsers internos (solo usados por este repo) ──────────────────────────────

func (r *ReservasRepo) parseSemanas(data [][]interface{}) []models.Semana {
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
			semanas = append(semanas, models.Semana{Titulo: val})
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

// ── Helpers internos ──────────────────────────────────────────────────────────

func (r *ReservasRepo) newService(ctx context.Context) (*sheets.Service, error) {
	srv, err := sheets.NewService(ctx,
		option.WithCredentialsFile(r.cfg.CredentialsPath),
	)
	if err != nil {
		return nil, fmt.Errorf("error al conectar con Google Sheets: %w", err)
	}
	return srv, nil
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
