package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/noa-santo/tagfs/internal/db/gen"
	"github.com/noa-santo/tagfs/internal/logic"
)

func (db *DB) GetFilesForDir(ctx context.Context, selectedTags []string) ([]gen.File, error) {
	if len(selectedTags) == 0 {
		return nil, nil
	}
	implicitTags := logic.GetImplicitTags(selectedTags)
	allTagsMap := make(map[string]struct{})
	for _, t := range selectedTags {
		allTagsMap[t] = struct{}{}
	}
	for _, t := range implicitTags {
		allTagsMap[t] = struct{}{}
	}

	var tagArgs []interface{}
	for t := range allTagsMap {
		tagArgs = append(tagArgs, t)
	}
	tagCount := len(tagArgs)

	placeholders := make([]string, tagCount)
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(`
		SELECT f.id, f.orig_name, f.mode
		FROM files f
		JOIN file_tags ft ON f.id = ft.file_id
		WHERE ft.tag_name IN (%s)
		GROUP BY f.id
		HAVING COUNT(DISTINCT ft.tag_name) = ?
	`, strings.Join(placeholders, ","))

	args := append(tagArgs, tagCount)

	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying files for tags: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			dbLogger.Fatalf("Error while closing rows: %v", err)
		}
	}(rows)

	var files []gen.File
	for rows.Next() {
		var f gen.File
		err := rows.Scan(&f.ID, &f.OrigName, &f.Mode)
		if err != nil {
			return nil, fmt.Errorf("scanning file row: %w", err)
		}
		files = append(files, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating file rows: %w", err)
	}

	return files, nil
}

func (db *DB) UpdateTags(id string, tags []string) error {
	tx, err := db.db.BeginTx(db.Ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	committed := false
	defer func(tx *sql.Tx, committed *bool) {
		if *committed {
			return
		}
		err := tx.Rollback()
		if err != nil {
			dbLogger.Fatalf("Error while rolling back transaction: %v", err)
		}
	}(tx, &committed)
	qtx := db.Queries.WithTx(tx)

	err = qtx.ClearFileTags(db.Ctx, id)
	if err != nil {
		return fmt.Errorf("clearing existing tags: %w", err)
	}

	seen := make(map[string]struct{})
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		err = qtx.InsertFileTag(db.Ctx, gen.InsertFileTagParams{
			FileID:  id,
			TagName: tag,
		})
		if err != nil {
			return fmt.Errorf("inserting tag %q: %w", tag, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	committed = true

	return nil
}
