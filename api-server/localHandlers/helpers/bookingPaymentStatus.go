package helpers

import (
	"database/sql"

	"time"
)

func TimeOrNil(validTime sql.NullTime) any {
	if validTime.Valid {
		return validTime.Time
	}
	return nil
}

func MaxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
