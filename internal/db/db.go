package db

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/noa-santo/tagfs/internal/config"
	"github.com/noa-santo/tagfs/internal/db/gen"
	_ "modernc.org/sqlite"
)

//go:generate sqlc generate
//go:embed schema.sql
var ddl string

var dbLogger = log.New(os.Stdout, "DB: ", log.LstdFlags)
var globalDBInstance *DB

type DB struct {
	db      *sql.DB
	Queries *gen.Queries
	Ctx     context.Context
}

func Get() *DB {
	if globalDBInstance != nil {
		return globalDBInstance
	}
	initDB()
	return globalDBInstance
}

func initDB() {
	ctx := context.Background()

	// The database is part of the user's durable home.  Create its parent
	// before opening it and make SQLite fail loudly rather than silently
	// running with unsafe defaults.
	dbDir := filepath.Join(config.Get().StoragePath, ".config")
	if err := os.MkdirAll(dbDir, 0700); err != nil {
		dbLogger.Panicf("Error creating database directory: %v", err)
	}
	dbPath := filepath.Join(dbDir, "tagfs.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		dbLogger.Panicf("Error opening database: %v", err)
	}
	// PRAGMAs such as foreign_keys are connection-local. A single pooled
	// connection keeps those guarantees true for every query.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// One writer plus WAL gives readers a consistent view while a write is in
	// progress. FULL synchronous is intentional: losing tags on a power loss
	// is worse than the small cost on metadata writes.
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = FULL",
		"PRAGMA busy_timeout = 5000",
	}
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			dbLogger.Panicf("SQLite setup failed (%s): %v", pragma, err)
		}
	}
	if err := migrate(ctx, db); err != nil {
		dbLogger.Panicf("Migration failed: %v", err)
	}

	queries := gen.New(db)
	globalDBInstance = &DB{
		db:      db,
		Queries: queries,
		Ctx:     ctx,
	}
}

func migrate(ctx context.Context, db *sql.DB) error {
	var version int
	err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version)
	if err != nil {
		return err
	}

	const targetVersion = 1

	if version < targetVersion {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return err
		}

		_, err = db.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", targetVersion))
		return err
	} else if version > targetVersion {
		dbLogger.Panic("DB schema mismatch! Schema in db is newer than schema source.")
	}

	return nil
}
