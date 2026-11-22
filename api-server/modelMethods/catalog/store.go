// Package catalog owns SQL for simple administrative lookup resources.
package catalog

import (
	"api-server/v2/modelMethods/resourcequery"
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
	"reflect"
	"strings"
)

func explicit[T any](query string) string {
	t := reflect.TypeOf((*T)(nil)).Elem()
	columns := []string{}
	for i := 0; i < t.NumField(); i++ {
		name := t.Field(i).Tag.Get("db")
		if name != "" && name != "-" {
			columns = append(columns, name)
		}
	}
	return strings.Replace(query, "SELECT *", "SELECT "+strings.Join(columns, ","), 1)
}

type Queries struct{ List, Get, Create, Update, Delete string }
type Definition[T any] struct {
	Queries    Queries
	CreateArgs func(*T) []any
	UpdateArgs func(int, *T) []any
	Limits     []int
}
type Store[T any] struct {
	DB         *sqlx.DB
	Definition Definition[T]
}

func (s Store[T]) List(ctx context.Context, p resourcequery.Page) ([]T, resourcequery.PageInfo, error) {
	return resourcequery.SelectPage[T](ctx, s.DB, p, s.Definition.Queries.List)
}
func (s Store[T]) Get(ctx context.Context, id int) (T, error) {
	var v T
	err := s.DB.GetContext(ctx, &v, s.Definition.Queries.Get, id)
	return v, err
}
func (s Store[T]) Save(ctx context.Context, id int, v T) (T, error) {
	var result T
	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	if id == 0 {
		err = tx.GetContext(ctx, &id, s.Definition.Queries.Create, s.Definition.CreateArgs(&v)...)
	} else {
		var r sql.Result
		r, err = tx.ExecContext(ctx, s.Definition.Queries.Update, s.Definition.UpdateArgs(id, &v)...)
		if err == nil {
			n, e := r.RowsAffected()
			err = e
			if err == nil && n == 0 {
				err = sql.ErrNoRows
			}
		}
	}
	if err != nil {
		return result, err
	}
	if err = tx.GetContext(ctx, &result, s.Definition.Queries.Get, id); err != nil {
		return result, err
	}
	return result, tx.Commit()
}
func (s Store[T]) Delete(ctx context.Context, id int) error {
	r, err := s.DB.ExecContext(ctx, s.Definition.Queries.Delete, id)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return err
}
func ID[T any](v T) int { return int(reflect.ValueOf(v).FieldByName("ID").Int()) }
