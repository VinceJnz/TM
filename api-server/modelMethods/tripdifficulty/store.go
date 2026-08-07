package tripdifficulty

import (
	"api-server/v2/models"
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("trip difficulty not found")

const (
	selectColumns = `id, level, level_short, description, created, modified`
	listQuery     = `SELECT ` + selectColumns + ` FROM et_trip_difficulty ORDER BY id`
	getQuery      = `SELECT ` + selectColumns + ` FROM et_trip_difficulty WHERE id = $1`
	createQuery   = `INSERT INTO et_trip_difficulty (level, level_short, description)
		VALUES ($1, $2, $3) RETURNING ` + selectColumns
	updateQuery = `UPDATE et_trip_difficulty
		SET level = $1, level_short = $2, description = $3
		WHERE id = $4 RETURNING ` + selectColumns
	deleteQuery = `DELETE FROM et_trip_difficulty WHERE id = $1`
)

// Store owns database access for trip difficulties. It deliberately has no HTTP dependencies.
type Store struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Store {
	return &Store{db: db}
}

func (s *Store) List(ctx context.Context) ([]models.TripDifficulty, error) {
	items := []models.TripDifficulty{}
	if err := s.db.SelectContext(ctx, &items, listQuery); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) Get(ctx context.Context, id int) (models.TripDifficulty, error) {
	var item models.TripDifficulty
	if err := s.db.GetContext(ctx, &item, getQuery, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.TripDifficulty{}, ErrNotFound
		}
		return models.TripDifficulty{}, err
	}
	return item, nil
}

func (s *Store) Create(ctx context.Context, input models.TripDifficulty) (models.TripDifficulty, error) {
	var item models.TripDifficulty
	if err := s.db.GetContext(ctx, &item, createQuery, input.Level, input.LevelShort, input.Description); err != nil {
		return models.TripDifficulty{}, err
	}
	return item, nil
}

func (s *Store) Update(ctx context.Context, id int, input models.TripDifficulty) (models.TripDifficulty, error) {
	var item models.TripDifficulty
	if err := s.db.GetContext(ctx, &item, updateQuery, input.Level, input.LevelShort, input.Description, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.TripDifficulty{}, ErrNotFound
		}
		return models.TripDifficulty{}, err
	}
	return item, nil
}

func (s *Store) Delete(ctx context.Context, id int) error {
	result, err := s.db.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
