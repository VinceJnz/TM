package appCore

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
)

type ContextKey string

type Config struct {
	Db        *sqlx.DB
	UserIDKey ContextKey
	Settings  settings
	Mux       *mux.Router
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
		Mux:       mux.NewRouter(),
	}
}

func (c *Config) Close() {
	c.Db.Close()
}
