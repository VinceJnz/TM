package app

import (
	"fmt"
	"log"
	"os"

	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

/*
set DB_USER api_user
set DB_PASSWORD api_password
set DB_NAME mydatabase
*/

func InitDB() (*sqlx.DB, error) {
	var err error
	var db *sqlx.DB

	err = godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	db, err = sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, err
	}

	return db, nil
}
