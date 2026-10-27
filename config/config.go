package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DB      DBConfig
	Auth    AuthConfig
	Storage StorageConfig
	S3      S3Config
	Jobs    JobsConfig
}

// S3Config apunta al bucket de AWS S3 usado para imagenes de inventario.
// Sin Bucket la gestion de imagenes de inventario queda deshabilitada (503).
type S3Config struct {
	Bucket string
	Region string
	// PublicBaseURL es el dominio de lectura (p. ej. CloudFront). Vacio = URL directa del bucket.
	PublicBaseURL string
}

// JobsConfig guarda el secreto con el que un disparador externo (EventBridge) puede ejecutar
// la generacion de gastos fijos sin token de usuario. Vacio = solo admin_sys.
type JobsConfig struct {
	Secret string
}

// StorageConfig apunta al bucket de Supabase Storage usado para imagenes de combos.
type StorageConfig struct {
	SupabaseURL string
	SecretKey   string
	Bucket      string
}

type AuthConfig struct {
	TokenSecret string
	TokenTTL    time.Duration
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

	App = &Config{
		DB: DBConfig{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     os.Getenv("DB_NAME"),
			SSLMode:  os.Getenv("DB_SSLMODE"),
		},
		Auth: AuthConfig{
			TokenSecret: getEnvDefault("AUTH_TOKEN_SECRET", "atrevida-local-dev-secret"),
			TokenTTL:    time.Duration(getEnvIntDefault("AUTH_TOKEN_TTL_MINUTES", 60)) * time.Minute,
		},
		Storage: StorageConfig{
			SupabaseURL: strings.TrimRight(os.Getenv("SUPABASE_URL"), "/"),
			SecretKey:   os.Getenv("SUPABASE_SECRET_KEY"),
			Bucket:      getEnvDefault("SUPABASE_STORAGE_BUCKET", "paquetes"),
		},
		S3: S3Config{
			Bucket:        os.Getenv("S3_BUCKET"),
			Region:        getEnvDefault("S3_REGION", "sa-east-1"),
			PublicBaseURL: os.Getenv("S3_PUBLIC_BASE_URL"),
		},
		Jobs: JobsConfig{
			Secret: os.Getenv("JOB_SECRET"),
		},
	}
}

func getEnvDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvIntDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}
