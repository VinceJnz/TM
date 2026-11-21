package resourcequery

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"reflect"
)

// One scans one database row into a record, then adapts legacy array responses.
// SQL code has no HTTP dependency and missing rows remain sql.ErrNoRows.
func One(ctx context.Context, db *sqlx.DB, dest any, query string, args ...any) error {
	value := reflect.ValueOf(dest)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return fmt.Errorf("record destination must be a pointer")
	}
	if value.Elem().Kind() != reflect.Slice {
		return db.GetContext(ctx, dest, query, args...)
	}
	slice := value.Elem()
	record := reflect.New(slice.Type().Elem())
	if err := db.GetContext(ctx, record.Interface(), query, args...); err != nil {
		return err
	}
	slice.Set(reflect.Append(reflect.MakeSlice(slice.Type(), 0, 1), record.Elem()))
	return nil
}
