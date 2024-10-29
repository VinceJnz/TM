package models

import (
	"time"

	"github.com/guregu/null/v5/zero"
)

type Season struct {
	ID       int       `json:"id" db:"id"`
	Season   string    `json:"season" db:"season"`
	StartDay zero.Int  `json:"start_day" db:"start_day"`
	Length   zero.Int  `json:"length" db:"length"`
	Created  time.Time `json:"created" db:"created"`
	Modified time.Time `json:"modified" db:"modified"`
}
