package postgres

import (
	"database/sql"
	"embed"
	_ "expo-open-ota/internal/database/postgres/migrations"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*
var embedMigrations embed.FS

func RunDBMigrations(dbURL string) {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("❌ [DATABASE] Failed to open SQL connection for schema migrations: %v", err)
	}
	defer db.Close()

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("❌ [DATABASE] Failed to set goose dialect: %v", err)
	}

	log.Println("🔧 [DATABASE] Checking and running PostgreSQL schema migrations...")

	if err := goose.Up(db, "migrations"); err != nil {
		log.Fatalf("🚨 [DATABASE] PostgreSQL migration execution failed: %v", err)
	}

	log.Println("🎉 [DATABASE] PostgreSQL schema up to date!")
}
