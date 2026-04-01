package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	SpreadsheetID     string
	SheetsDisponibles []string
	CredentialsPath   string
}

var App *Config

func Load() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	sheetsRaw := os.Getenv("SHEETS_DISPONIBLES")
	sheets := strings.Split(sheetsRaw, ",")

	App = &Config{
		SpreadsheetID:     os.Getenv("SPREADSHEET_ID"),
		SheetsDisponibles: sheets,
		CredentialsPath:   os.Getenv("CREDENTIALS_PATH"),
	}
}
