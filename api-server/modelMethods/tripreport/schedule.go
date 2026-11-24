package tripreport

import (
	"context"
	"time"
)

type ScheduleFilter struct {
	Month           *time.Time
	TypeID, GradeID int
	Upcoming, Past  bool
}

type Summary struct {
	TripID          int       `db:"trip_id" json:"trip_id"`
	TripName        string    `db:"trip_name" json:"trip_name"`
	TripFrom        time.Time `db:"from_date" json:"from_date"`
	TripTo          time.Time `db:"to_date" json:"to_date"`
	MaxParticipants int       `db:"max_participants" json:"max_participants"`
	Type            string    `db:"type" json:"type"`
	Grade           string    `db:"grade" json:"grade"`
	Status          string    `db:"status" json:"status"`
	Location        string    `db:"location" json:"location"`
	Description     string    `db:"description" json:"description"`
}

type FilterOption struct {
	Kind  string `db:"kind" json:"kind"`
	ID    int    `db:"id" json:"id"`
	Label string `db:"label" json:"label"`
}

func (s Store) Filters(ctx context.Context) ([]FilterOption, error) {
	options := []FilterOption{}
	err := s.DB.SelectContext(ctx, &options, `
		SELECT 'type' AS kind, id, type AS label FROM et_trip_type
		UNION ALL SELECT 'grade', id, level FROM et_trip_difficulty
		ORDER BY kind, label, id`)
	return options, err
}

func (s Store) Schedule(ctx context.Context, filter ScheduleFilter, limit, offset int) ([]Summary, bool, error) {
	rows := []Summary{}
	var start, end *time.Time
	if filter.Month != nil {
		start = filter.Month
		next := start.AddDate(0, 1, 0)
		end = &next
	}
	err := s.DB.SelectContext(ctx, &rows, `
		SELECT t.id AS trip_id, t.trip_name, t.from_date, t.to_date, t.max_participants,
			COALESCE(ty.type, '') AS type, COALESCE(d.level, '') AS grade,
			COALESCE(st.status, '') AS status, COALESCE(t.location, '') AS location,
			COALESCE(t.description, '') AS description
		FROM at_trips t
		LEFT JOIN et_trip_type ty ON ty.id=t.trip_type_id
		LEFT JOIN et_trip_difficulty d ON d.id=t.difficulty_level_id
		LEFT JOIN et_trip_status st ON st.id=t.trip_status_id
		WHERE t.from_date IS NOT NULL AND t.to_date IS NOT NULL
			AND ($1::date IS NULL OR (t.to_date >= $1::date AND t.from_date < $2::date))
			AND ($3::integer = 0 OR t.trip_type_id = $3)
			AND ($4::integer = 0 OR t.difficulty_level_id = $4)
			AND (NOT $5::boolean OR t.to_date >= (now() AT TIME ZONE 'Pacific/Auckland')::date)
			AND (NOT $6::boolean OR t.to_date < (now() AT TIME ZONE 'Pacific/Auckland')::date)
		ORDER BY CASE WHEN $6 THEN t.from_date END DESC, t.from_date, t.id
		LIMIT $7 OFFSET $8`, start, end, filter.TypeID, filter.GradeID, filter.Upcoming, filter.Past, limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	more := len(rows) > limit
	if more {
		rows = rows[:limit]
	}
	return rows, more, nil
}
