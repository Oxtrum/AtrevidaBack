package importacion

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	sheetsrepo "atrevida-agenda-api/repositories/sheets"
)

// ResultadoImportacion contiene el resumen del proceso legacy.
type ResultadoImportacion struct {
	Categorias      int `json:"categorias"`
	Servicios       int `json:"servicios"`
	ServicioLocales int `json:"servicio_locales"`
	Combos          int `json:"combos"`
	ComboLocales    int `json:"combo_locales"`
	ComboServicios  int `json:"combo_servicios"`
}

// ImportService queda como stub legacy para evitar dependencias activas con Google Sheets.
type ImportService struct{}

func NewImportService(_ *sqlx.DB, _ *sheetsrepo.ReservasRepo) *ImportService {
	return &ImportService{}
}

func (s *ImportService) Ejecutar() (ResultadoImportacion, error) {
	return ResultadoImportacion{}, fmt.Errorf("Google Sheets ya no soportado")
}
