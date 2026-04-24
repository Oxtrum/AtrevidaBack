package importacion

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"

	sheetsrepo "atrevida-agenda-api/repositories/sheets"
	"atrevida-agenda-api/utils"
)

// ResultadoImportacion contiene el resumen del proceso.
type ResultadoImportacion struct {
	Categorias      int `json:"categorias"`
	Servicios       int `json:"servicios"`
	ServicioLocales int `json:"servicio_locales"`
	Combos          int `json:"combos"`
	ComboLocales    int `json:"combo_locales"`
	ComboServicios  int `json:"combo_servicios"`
}

// ImportService orquesta la importación completa desde Google Sheets a PostgreSQL.
type ImportService struct {
	db   *sqlx.DB
	repo *sheetsrepo.ReservasRepo
}

func NewImportService(db *sqlx.DB, repo *sheetsrepo.ReservasRepo) *ImportService {
	return &ImportService{db: db, repo: repo}
}

// Pipeline
//  1. Lee el sheet de servicios y combos
//  2. Drop + Create tablas temporales
//  3. Carga datos en temporales
//  4. Trunca reales y migra desde temporales (en una tx)
//  5. Limpia temporales
func (s *ImportService) Ejecutar() (ResultadoImportacion, error) {
	var resultado ResultadoImportacion

	log.Println("[Import] Iniciando importación desde Google Sheets")

	// 1. Leer sheets
	log.Println("[Import] Leyendo sheet de servicios y combos...")

	sheetData := s.repo.GetSheetData("SERVICIOS")
	if len(sheetData) == 0 {
		return resultado, fmt.Errorf("sheet SERVICIOS vacío o inaccesible")
	}

	servicios := utils.ParseServiciosSheet(sheetData, utils.MaxFilaServicios)
	combos := utils.ParseCombosSheet(sheetData, utils.MinFilaCombos, utils.MaxFilaCombos)

	log.Printf("[Import] Leídos: %d servicios, %d combos", len(servicios), len(combos))

	// 2. Preparar staging
	st := newStaging(s.db)

	log.Println("[Import] Preparando tablas temporales...")
	if err := st.Drop(); err != nil {
		return resultado, fmt.Errorf("error al limpiar tablas temporales previas: %w", err)
	}
	if err := st.Create(); err != nil {
		return resultado, fmt.Errorf("error al crear tablas temporales: %w", err)
	}
	if err := st.Truncate(); err != nil {
		return resultado, fmt.Errorf("error al truncar tablas temporales: %w", err)
	}

	// 3. Cargar en temporales
	ld := newLoader(s.db)

	log.Println("[Import] Cargando servicios en staging...")
	if err := ld.CargarServicios(servicios); err != nil {
		_ = st.Drop()
		return resultado, fmt.Errorf("error al cargar servicios en staging: %w", err)
	}

	log.Println("[Import] Cargando combos en staging...")
	if err := ld.CargarCombos(combos); err != nil {
		_ = st.Drop()
		return resultado, fmt.Errorf("error al cargar combos en staging: %w", err)
	}

	// 4. Migrar a tablas reales
	log.Println("[Import] Migrando a tablas reales...")
	mg := newMigracion(s.db)

	stats, err := mg.Ejecutar()
	if err != nil {
		_ = st.Drop()
		return resultado, fmt.Errorf("error durante la migración: %w", err)
	}

	// 5. Limpiar temporales
	log.Println("[Import] Limpiando tablas temporales...")
	if err := st.Drop(); err != nil {
		// no es fatal — loguear y continuar
		log.Printf("[Import] WARN: error al limpiar temporales: %v", err)
	}

	resultado = ResultadoImportacion{
		Categorias:      stats.Categorias,
		Servicios:       stats.Servicios,
		ServicioLocales: stats.ServicioLocales,
		Combos:          stats.Combos,
		ComboLocales:    stats.ComboLocales,
		ComboServicios:  stats.ComboServicios,
	}

	log.Printf("[Import] Completado — categorías: %d, servicios: %d, combos: %d",
		resultado.Categorias, resultado.Servicios, resultado.Combos)

	return resultado, nil
}
