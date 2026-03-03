package appCore

import (
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/jmoiron/sqlx"
)

/*
set DB_USER api_user
set DB_PASSWORD api_password
set DB_NAME mydatabase
*/

func InitDB(dataSource string) (*sqlx.DB, error) {
	var err error
	var db *sqlx.DB
	connStr := strings.TrimSpace(dataSource)

	if connStr == "" {
		return nil, fmt.Errorf("missing database connection string in loaded settings (DataSource)")
	}

	// Retry up to 3 times
	for i := 0; i < 3; i++ {
		db, err = sqlx.Connect("postgres", connStr)
		if err != nil {
			log.Printf("%sInitDB()1 connect attempt %d/3 failed: %v", debugTag, i+1, err)
		} else {
			log.Printf("%sInitDB()2 connected", debugTag)
			return db, nil
		}
		time.Sleep(1 * time.Second)
	}
	log.Printf("%sInitDB()3 unable to connect after retries: %v", debugTag, err)
	return nil, fmt.Errorf("unable to connect to database after retries: %w", err)
}
