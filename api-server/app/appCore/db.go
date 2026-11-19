package appCore

import (
	"context"
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
	return InitDBContext(context.Background(), dataSource)
}

func InitDBContext(ctx context.Context, dataSource string) (*sqlx.DB, error) {
	var err error
	var db *sqlx.DB
	connStr := strings.TrimSpace(dataSource)

	if connStr == "" {
		return nil, fmt.Errorf("missing database connection string in loaded settings (DataSource)")
	}

	// Retry up to 3 times
	for i := 0; i < 3; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		db, err = sqlx.Open("postgres", connStr)
		if err != nil {
			return nil, fmt.Errorf("database configuration invalid")
		}
		db.SetMaxOpenConns(20)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(30 * time.Minute)
		db.SetConnMaxIdleTime(5 * time.Minute)
		ping, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = db.PingContext(ping)
		cancel()
		if err != nil {
			db.Close()
			log.Printf("%sInitDB() connect attempt %d/3 failed", debugTag, i+1)
		} else {
			log.Printf("%sInitDB() connected", debugTag)
			return db, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return nil, fmt.Errorf("unable to connect to database after retries")
}
