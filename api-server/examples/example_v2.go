// Example from CoPilot 20241020
// Dealing with group bookings
// Each booking contains 1 user. This means we don't need a BookingPeople table
// Group bookings can be managed via the UI
// This offers more flexibility and solves some UI problems.

package main

/*
import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const (
	dbUser     = "your_db_user"
	dbPassword = "your_db_password"
	dbName     = "trip_manager"
)

var db *sqlx.DB

func initDB() {
	var err error
	dbInfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", dbUser, dbPassword, dbName)
	db, err = sqlx.Open("postgres", dbInfo)
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	var users []struct {
		UserID   int    `db:"user_id" json:"user_id"`
		UserName string `db:"user_name" json:"user_name"`
		Email    string `db:"email" json:"email"`
		Status   string `db:"status" json:"status"`
	}

	err := db.Select(&users, "SELECT user_id, user_name, email, status FROM users")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func createGroupBooking(w http.ResponseWriter, r *http.Request) {
	var group struct {
		GroupName string `json:"group_name"`
		Bookings  []struct {
			TripID   int    `json:"trip_id"`
			UserID   int    `json:"user_id"`
			FromDate string `json:"from_date"`
			ToDate   string `json:"to_date"`
			BookDate string `json:"booking_date"`
		} `json:"bookings"`
	}

	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tx := db.MustBegin()
	var groupID int
	tx.QueryRowx("INSERT INTO group_bookings (group_name) VALUES ($1) RETURNING group_booking_id", group.GroupName).Scan(&groupID)

	for _, booking := range group.Bookings {
		tx.MustExec("INSERT INTO bookings (trip_id, user_id, from_date, to_date, booking_date, group_booking_id) VALUES ($1, $2, $3, $4, $5, $6)", booking.TripID, booking.UserID, booking.FromDate, booking.ToDate, booking.BookDate, groupID)
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"group_booking_id": groupID})
}

func main() {
	initDB()
	defer db.Close()

	r := mux.NewRouter()
	r.HandleFunc("/users", getUsers).Methods("GET")
	r.HandleFunc("/group_bookings", createGroupBooking).Methods("POST")

	// Define other routes (e.g., for trips, bookings, costs, etc.)

	log.Fatal(http.ListenAndServe(":8080", r))
}

*/
