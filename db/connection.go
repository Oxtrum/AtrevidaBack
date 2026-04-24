package db

import (
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"atrevida-agenda-api/config"
)

// Connect
func Connect(cfg *config.Config) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.User,
		cfg.DB.Password, cfg.DB.Name, cfg.DB.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("error al conectar con PostgreSQL: %w", err)
	}

	db.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	db.SetMaxIdleConns(cfg.DB.MaxIdleConns)

	log.Println("[DB] Conexión establecida con PostgreSQL")
	return db, nil
}

// RunMigrations ejecuta todas las migraciones pendientes
// Path: "file://migrations"
func RunMigrations(db *sqlx.DB, migrationsPath string) error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("error al crear driver de migraciones: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "postgres", driver)
	if err != nil {
		return fmt.Errorf("error al inicializar migraciones: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("error al aplicar migraciones: %w", err)
	}

	version, dirty, _ := m.Version()
	log.Printf("[DB] Migraciones aplicadas — versión actual: %d (dirty: %v)", version, dirty)
	return nil
}
