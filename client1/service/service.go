package service

import (
	"time"
)

type DataState[T any] struct {
	Data        T
	LastFetched time.Time
	Loading     bool
	Err         error
}
