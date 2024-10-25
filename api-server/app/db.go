package app

import (
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

const debugTag = "app"

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

	// Retry up to 3 times
	for i := 0; i < 3; i++ {
		db, err = sqlx.Connect("postgres", connStr)
		if err != nil {
			log.Println(debugTag+"InitDB()1 ", err)
		} else {
			log.Println(debugTag+"InitDB()2 connected", err)
			return db, nil
		}
		time.Sleep(1 * time.Second)
	}
	log.Println(debugTag+"InitDB()3 ", err)
	return nil, err
}
