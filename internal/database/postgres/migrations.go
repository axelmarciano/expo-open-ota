package postgres

import (
	"context"
	"database/sql"
	"embed"
	_ "expo-open-ota/internal/database/postgres/migrations"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*
var embedMigrations embed.FS

// Arbitrary app-wide id for the Postgres advisory lock that serializes migrators.
const migrationAdvisoryLockID = 823672941

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

	// Several migrators can run at once: parallel test packages sharing one
	// TEST_DATABASE_URL, or multiple server replicas booting simultaneously.
	// Without a lock they race inside goose.Up (duplicate CREATE TABLE hits
	// pg_type_typname_nsp_index) and the loser dies on Fatalf. An advisory
	// lock serializes them: the first applies, the rest wait then no-op.
	// Advisory locks are session-scoped, so hold a dedicated connection.
	lockCtx, cancel := context.WithTimeout(context.Background(), migrationLockTimeout)
	defer cancel()
	conn, err := db.Conn(lockCtx)
	if err != nil {
		log.Fatalf("❌ [DATABASE] Failed to acquire connection for migration lock: %v", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(lockCtx, "SELECT pg_advisory_lock($1)", migrationAdvisoryLockID); err != nil {
		log.Fatalf("❌ [DATABASE] Failed to acquire migration advisory lock: %v", err)
	}
	defer conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", migrationAdvisoryLockID)

	// WithAllowMissing applies migrations whose version is lower than the one already
	// recorded in the database. Parallel PRs get merged out of timestamp order, so a
	// deployment can pick up a migration that predates one it already ran. Migrations
	// here are independent of each other, so applying them out of order is safe.
	if err := goose.Up(db, "migrations", goose.WithAllowMissing()); err != nil {
		log.Fatalf("🚨 [DATABASE] PostgreSQL migration execution failed: %v", err)
	}

	log.Println("🎉 [DATABASE] PostgreSQL schema up to date!")
}
