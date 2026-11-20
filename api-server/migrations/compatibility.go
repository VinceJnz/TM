package migrations

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"github.com/jmoiron/sqlx"
)

//go:embed baseline_catalog.json catalog.query
var catalogFiles embed.FS

type CatalogEntry struct {
	Kind       string `json:"kind" db:"kind"`
	Name       string `json:"name" db:"name"`
	Definition string `json:"definition" db:"definition"`
}
type Difference struct {
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
}
type catalogReader interface {
	SelectContext(context.Context, any, string, ...any) error
}

// Compatibility is strictly read-only. Extra application objects are allowed;
// missing or altered baseline invariants are reported for explicit reconciliation.
func Compatibility(ctx context.Context, db *sqlx.DB) ([]Difference, error) {
	tx, err := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	return compareCatalog(ctx, tx)
}
func compareCatalog(ctx context.Context, reader catalogReader) ([]Difference, error) {
	manifest, _ := catalogFiles.ReadFile("baseline_catalog.json")
	query, _ := catalogFiles.ReadFile("catalog.query")
	var expected, actual []CatalogEntry
	if err := json.Unmarshal(manifest, &expected); err != nil {
		return nil, err
	}
	if err := reader.SelectContext(ctx, &actual, string(query)); err != nil {
		return nil, err
	}
	found := map[string]string{}
	for _, entry := range actual {
		found[entry.Kind+":"+entry.Name] = entry.Definition
	}
	differences := []Difference{}
	for _, entry := range expected {
		value := found[entry.Kind+":"+entry.Name]
		// Migration 0003 deliberately widens the token column. Both the original
		// baseline and that known upgrade are compatible with baseline adoption.
		if entry.Kind == "column" && entry.Name == "st_token.token" && value == "text" {
			continue
		}
		if value != entry.Definition {
			differences = append(differences, Difference{entry.Kind, entry.Name, entry.Definition, value})
		}
	}
	return differences, nil
}
