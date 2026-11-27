package appearance

import (
	"context"
	"encoding/json"

	"github.com/jmoiron/sqlx"
)

type Store struct{ DB *sqlx.DB }

type Revision struct {
	Version  int64           `db:"version"`
	Settings json.RawMessage `db:"settings"`
}

func (s Store) Read(ctx context.Context, previous bool) (Revision, error) {
	var r Revision
	offset := 0
	if previous {
		offset = 1
	}
	err := s.DB.GetContext(ctx, &r, `SELECT version,settings FROM st_appearance_revision ORDER BY version DESC LIMIT 1 OFFSET $1`, offset)
	return r, err
}

func (s Store) Publish(ctx context.Context, expected int64, settings []byte, actor int) (int64, error) {
	var version int64
	err := s.DB.GetContext(ctx, &version, `
 INSERT INTO st_appearance_revision(version,settings,published_by)
 SELECT current.version + 1,$2::jsonb,$3 FROM (
   SELECT version FROM st_appearance_revision ORDER BY version DESC LIMIT 1
 ) current WHERE current.version=$1
 ON CONFLICT(version) DO NOTHING RETURNING version`, expected, string(settings), actor)
	return version, err
}
