package appCore

import (
	"log"

	"github.com/jmoiron/sqlx"
)

type ContextKey string

type Config struct {
	Db        *sqlx.DB
	UserIDKey ContextKey
}

func IdKey() ContextKey {
	return ContextKey("userID")
}

func New() *Config {
	db, err := InitDB()
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	userIDKey := IdKey()
	return &Config{
		Db:        db,
		UserIDKey: userIDKey,
	}
}
