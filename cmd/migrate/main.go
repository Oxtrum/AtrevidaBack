package main

import (
	"log"

	"atrevida-agenda-api/config"
	"atrevida-agenda-api/db"
)

func main() {
	config.Load()

	pgDB, err := db.Connect(config.App)
	if err != nil {
		log.Fatalf("Error al conectar: %v", err)
	}

	if err := db.RunMigrations(pgDB, "file://migrations"); err != nil {
		log.Fatalf("Error en migraciones: %v", err)
	}

	log.Println("Migraciones completadas exitosamente")
}
