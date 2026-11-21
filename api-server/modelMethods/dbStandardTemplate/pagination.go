package dbStandardTemplate

import (
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

var ErrPagination = errors.New("invalid pagination")

// selectPage preserves array responses and uses a next-offset header. The resource ID
// is selected explicitly by name, independently of projection column order.
func selectPage(w http.ResponseWriter, r *http.Request, db *sqlx.DB, dest any, query string, args ...any) error {
	limit, offset := 100, 0
	for name, target := range map[string]*int{"limit": &limit, "offset": &offset} {
		if raw := r.URL.Query().Get(name); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value < 0 {
				return ErrPagination
			}
			*target = value
		}
	}
	if limit < 1 || limit > 500 || offset > 1000000 {
		return ErrPagination
	}
	query = strings.TrimSuffix(strings.TrimSpace(query), ";")
	paged := fmt.Sprintf("SELECT * FROM (%s) AS resource_page ORDER BY resource_page.id LIMIT $%d OFFSET $%d", query, len(args)+1, len(args)+2)
	cursor := r.URL.Query().Get("pagination") == "cursor"
	if cursor {
		after := int64(0)
		if raw := r.URL.Query().Get("after"); raw != "" {
			var err error
			after, err = strconv.ParseInt(raw, 10, 64)
			if err != nil || after < 0 {
				return ErrPagination
			}
		}
		paged = fmt.Sprintf("SELECT * FROM (%s) AS resource_page WHERE resource_page.id>$%d ORDER BY resource_page.id LIMIT $%d", query, len(args)+1, len(args)+2)
		args = append(append([]any{}, args...), after, limit+1)
		w.Header().Set("X-Pagination-Mode", "cursor")
	} else {
		args = append(append([]any{}, args...), limit+1, offset)
	}
	if err := db.SelectContext(r.Context(), dest, paged, args...); err != nil {
		return err
	}
	value := reflect.ValueOf(dest).Elem()
	if value.Kind() != reflect.Slice {
		return fmt.Errorf("pagination requires slice destination")
	}
	if value.Len() > limit {
		value.Set(value.Slice(0, limit))
		w.Header().Set("X-Next-Offset", strconv.Itoa(offset+limit))
		if cursor {
			last := value.Index(limit - 1)
			if last.Kind() == reflect.Struct {
				last = last.FieldByName("ID")
			}
			if !last.IsValid() || !last.CanInt() {
				return fmt.Errorf("cursor requires integer resource ID")
			}
			w.Header().Set("X-Next-Cursor", strconv.FormatInt(last.Int(), 10))
		}
	}
	w.Header().Set("X-Page-Limit", strconv.Itoa(limit))
	return nil
}
