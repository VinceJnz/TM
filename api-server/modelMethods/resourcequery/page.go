package resourcequery

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"reflect"
	"strings"
)

// Page is only for resources with a unique integer ID. Reports own their ordering.
type Page struct {
	Limit, Offset int
	Cursor        bool
	After         int64
}
type PageInfo struct {
	Limit, NextOffset int
	Cursor            bool
	NextCursor        int64
}

func SelectPage[T any](ctx context.Context, db *sqlx.DB, page Page, query string, args ...any) ([]T, PageInfo, error) {
	info := PageInfo{Limit: page.Limit, NextOffset: -1, Cursor: page.Cursor}
	if page.Limit < 1 || page.Limit > 500 || page.Offset < 0 || page.Offset > 1000000 || page.After < 0 {
		return nil, info, fmt.Errorf("invalid page")
	}
	query = strings.TrimSuffix(strings.TrimSpace(query), ";")
	sql := fmt.Sprintf("SELECT * FROM (%s) resource_page ORDER BY resource_page.id LIMIT $%d OFFSET $%d", query, len(args)+1, len(args)+2)
	values := append(append([]any{}, args...), page.Limit+1, page.Offset)
	if page.Cursor {
		sql = fmt.Sprintf("SELECT * FROM (%s) resource_page WHERE resource_page.id>$%d ORDER BY resource_page.id LIMIT $%d", query, len(args)+1, len(args)+2)
		values = append(append([]any{}, args...), page.After, page.Limit+1)
	}
	items := []T{}
	if err := db.SelectContext(ctx, &items, sql, values...); err != nil {
		return nil, info, err
	}
	if len(items) > page.Limit {
		items = items[:page.Limit]
		info.NextOffset = page.Offset + page.Limit
		if page.Cursor {
			last := reflect.ValueOf(items[len(items)-1])
			if last.Kind() != reflect.Struct {
				return nil, info, fmt.Errorf("cursor requires resource struct")
			}
			id := last.FieldByName("ID")
			if !id.IsValid() || !id.CanInt() {
				return nil, info, fmt.Errorf("cursor requires integer ID")
			}
			info.NextCursor = id.Int()
		}
	}
	return items, info, nil
}
