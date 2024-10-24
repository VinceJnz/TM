package models

import "time"

type Season struct {
	ID       int       `json:"id" db:"id"`
	Season   string    `json:"season" db:"season"`
	Created  time.Time `json:"created" db:"created"`
	Modified time.Time `json:"modified" db:"modified"`
}
