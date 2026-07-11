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

	dbPath := filepath.Join(config.Get().StoragePath, ".config", "tagfs.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		dbLogger.Panicf("Error opening database: %v", err)
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
