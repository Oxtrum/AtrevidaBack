package db

import (
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"atrevida-agenda-api/config"
)

// Connect abre la conexión usando el driver pgx en modo "simple protocol".
//
// El backend corre detrás de RDS Proxy (transaction pooling). Con el driver
// lib/pq, cada query parametrizada usa el protocolo extendido (Parse/Bind con
// prepared statements). El pooler enruta Parse y Bind a conexiones de backend
// distintas y los prepared statements se cruzan, produciendo:
//
//	pq: bind message supplies 2 parameters, but prepared statement "" requires 1 (08P01)
//
// QueryExecModeSimpleProtocol elimina los prepared statements (envía la query
// con los parámetros ya interpolados en un solo mensaje), por lo que es seguro
// bajo transaction pooling sin perder los beneficios de RDS Proxy.
func Connect(cfg *config.Config) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.User,
		cfg.DB.Password, cfg.DB.Name, cfg.DB.SSLMode,
	)

	connConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("error al parsear configuración de PostgreSQL: %w", err)
	}
	connConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	sqlDB := stdlib.OpenDB(*connConfig)
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("error al conectar con PostgreSQL: %w", err)
	}

	db := sqlx.NewDb(sqlDB, "pgx")

	db.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	db.SetMaxIdleConns(cfg.DB.MaxIdleConns)

	log.Println("[DB] Conexión establecida con PostgreSQL (pgx simple protocol)")
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
