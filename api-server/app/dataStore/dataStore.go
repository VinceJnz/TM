package dataStore

import (
	"database/sql"
	"log"
	"time"

	//_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const debugTag = "dataStore."

//go slq drivers list - https://go.dev/wiki/SQLDrivers
//mysql driver - https://github.com/go-sql-driver/mysql/
// some problems with connecting to mariadb/mysql in docker
//https://github.com/docker-library/mysql/issues/124
//https://github.com/go-sql-driver/mysql/issues/674

// DB ??
type DB struct {
	*sql.DB
}

// NewConn ?? New DB connection //used in ... see appInit.InitDB
func NewConn(dataSourceName string) *sql.DB {
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		log.Panic(err)
		return nil // ?????
	}
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(3 * time.Second)
	if err = db.Ping(); err != nil {
		log.Panicln(debugTag+"NewConn()1", "err =", err)
		return nil
	}
	//DBConn = d
	return db
}

// NewConn ?? New DB connection //used in ... see appInit.InitDB
func NewPostgres(dataSourceName string) (*DB, error) {
	//"postgres://postgres:123@localhost:5432/project_mgnt?sslmode=disable"
	d, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		log.Panic(err)
		return nil, err
	}
	db := &DB{d}
	if err = db.Ping(); err != nil {
		log.Panicln(debugTag+"NewConn()1", "err =", err)
		return nil, err
	}
	//defer db.Close() //???????
	return db, nil
}

// NewConn ?? New DB connection //used in ... see appInit.InitDB
func NewMysql(dataSourceName string) (*DB, error) {
	d, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		log.Panic(err)
		return nil, err
	}
	db := &DB{d}
	if err = db.Ping(); err != nil {
		log.Panicln(debugTag+"NewConn()1", "err =", err)
		return nil, err
	}
	//defer db.Close() //???????
	return db, nil
}

// InitDB ??
func InitDB(dataSourceName string) *sql.DB {
	// Create the database handle, confirm driver is present
	// and wait for the DB connection to be up
	var err error
	var db *sql.DB
	var dbStatus string
	dbStatus = "Db Closed"
	for dbStatus != "" {
		log.Println(debugTag+"DB0 ", err, dbStatus)
		//db, err = sql.Open("mysql", "[username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]")
		//a.db, err = sql.Open("mysql", "project_mgnt_app:123@tcp(localhost:3306)/project_mgnt?parseTime=true")
		db, err := sql.Open("mysql", dataSourceName)
		log.Println(debugTag+"DB1 ", err, dbStatus)
		switch {
		case err == sql.ErrNoRows:
			dbStatus = "DB not available 1"
			log.Println(debugTag+"DB2 ", err, dbStatus)
			db.Close()
		case err != nil:
			dbStatus = "DB not available 2"
			log.Println(debugTag+"DB3 ", err, dbStatus)
			db.Close()
		}
		log.Println(debugTag+"DB4 ", err, dbStatus)
		err = db.Ping()
		log.Println(debugTag+"DB5 ", err, dbStatus)
		switch {
		case err != nil:
			dbStatus = "DB not available 3"
			log.Println(debugTag+"DB6 ", err, dbStatus)
			db.Close()
		default:
			dbStatus = ""
			log.Println(debugTag+"DB7 ", err, dbStatus)
		}
		time.Sleep(1 * time.Second)
	}
	return db
}
