package appCore

import (
	"log"

	"github.com/jmoiron/sqlx"
)

const debugTag = "appCore."

type ContextKey string

func GenerateSessionIDContextKey() ContextKey { // User for generating the context key for passing values via the context (ctx)
	return ContextKey("sessionID")
}

type Config struct {
	Db           *sqlx.DB
	SessionIDKey ContextKey // User for passing the user id value via the context (ctx)
	Settings     settings
	TestMode     bool
}

func New(testMode bool) *Config {
	db, err := InitDB()
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	sessionIDKey := GenerateSessionIDContextKey()
	return &Config{
		Db:           db,
		SessionIDKey: sessionIDKey,
		TestMode:     testMode,
	}
}

func (c *Config) Close() {
	c.Db.Close()
}

func (c *Config) Access() {

}
