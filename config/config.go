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
	// Turnstile valida el captcha del formulario publico de reservas.
	Turnstile TurnstileConfig
}

// TurnstileConfig son los datos para validar tokens de Cloudflare Turnstile (siteverify).
// Sin SecretKey o sin AllowedHostnames, las reservas publicas se rechazan (falla cerrado), salvo Disabled.
type TurnstileConfig struct {
	// SecretKey es la secret key del widget cuya site key usa el front.
	SecretKey string
	// AllowedHostnames: dominios aceptados, separados por coma (TURNSTILE_ALLOWED_HOSTNAMES).
	AllowedHostnames []string
	// Disabled omite la verificacion; solo para desarrollo local (TURNSTILE_DISABLED=true).
	Disabled bool
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
		Turnstile: TurnstileConfig{
			SecretKey:        strings.TrimSpace(os.Getenv("TURNSTILE_SECRET_KEY")),
			AllowedHostnames: splitCSV(os.Getenv("TURNSTILE_ALLOWED_HOSTNAMES")),
			Disabled:         strings.EqualFold(strings.TrimSpace(os.Getenv("TURNSTILE_DISABLED")), "true"),
		},
	}
}

// splitCSV separa una lista por comas, sin espacios ni elementos vacios.
func splitCSV(value string) []string {
	out := []string{}
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
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
