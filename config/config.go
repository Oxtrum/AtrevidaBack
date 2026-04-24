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
	DB                DBConfig
}

type DBConfig struct {
	Host         string
	Port         string
	User         string
	Password     string
	Name         string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
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
		DB: DBConfig{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     os.Getenv("DB_NAME"),
			SSLMode:  os.Getenv("DB_SSLMODE"),
		},
	}
}
