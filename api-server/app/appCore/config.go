package appCore

import (
	"log"

	"github.com/jmoiron/sqlx"
)

type ContextKey string

func GenerateUserIDContextKey() ContextKey { // User for generating the context key for passing values via the context (ctx)
	return ContextKey("userID")
}

type Config struct {
	Db        *sqlx.DB
	UserIDKey ContextKey // User for passing the user id value via the context (ctx)
	Settings  settings
	TestMode  bool
}

func New(testMode bool) *Config {
	db, err := InitDB()
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	userIDKey := GenerateUserIDContextKey()
	return &Config{
		Db:        db,
		UserIDKey: userIDKey,
		TestMode:  testMode,
	}
}

func (c *Config) Close() {
	c.Db.Close()
}
