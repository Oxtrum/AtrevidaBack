package repository

import "atrevida-agenda-api/models"

type ReservasRepository interface {
	// Lectura
	GetAllReservas() []models.LocalReservas
	GetSheetData(sheetName string) [][]interface{}

	// Escritura
	GetCeldaRaw(sheetName, a1Range string) (string, error)
	WriteCelda(sheetName, a1Range, contenido string) error
	ResolverCoordenada(sheetName, semana, dia, hora string) (string, error)
}
