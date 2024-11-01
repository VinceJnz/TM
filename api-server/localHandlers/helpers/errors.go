package helpers

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/go-sql-driver/mysql"
)

//	_ "github.com/jinzhu/gorm/dialects/postgres"
//	"github.com/jmoiron/sqlx"

func SqlErr(err error) (int, error) {
	switch err := err.(type) {
	case *mysql.MySQLError:
		switch err.Number {
		case 1451:
			return http.StatusConflict, errors.New("error: remove child records before deleteing this record")
		case 1452:
			return http.StatusConflict, errors.New("error: input data incomplete or incorrect")
		default:
			log.Printf("%v %v %+v", debugTag+"SqlErr()1", "err =", err)
			return http.StatusInternalServerError, errors.New(fmt.Errorf("mysql: %s", err.Error()).Error())
		}
	//case pg errors: // Add a type in here to deal with pq errors
	default:
		log.Printf("%v %v %T %+v", debugTag+"SqlErr()2", "err =", err, err)
		return http.StatusNotFound, errors.New("error: record not found")
	}
}
