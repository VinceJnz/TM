package main

import (
	"api-server/v2/app"
	"api-server/v2/localHandlers/handlerBooking"
	"api-server/v2/localHandlers/handlerBookingPeople"
	"api-server/v2/localHandlers/handlerBookingStatus"
	"api-server/v2/localHandlers/handlerTrip"
	"api-server/v2/localHandlers/handlerTripBooking"
	"api-server/v2/localHandlers/handlerUser"
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	db, err := app.InitDB()
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer db.Close()

	r := mux.NewRouter()

	// User routes
	user := handlerUser.New(db)
	r.HandleFunc("/users", user.GetAll).Methods("GET")
	r.HandleFunc("/users/{id}", user.Get).Methods("GET")
	r.HandleFunc("/users", user.Create).Methods("POST")
	r.HandleFunc("/users/{id}", user.Update).Methods("PUT")
	r.HandleFunc("/users/{id}", user.Delete).Methods("DELETE")

	// Booking routes
	booking := handlerBooking.New(db)
	r.HandleFunc("/bookings", booking.GetAll).Methods("GET")
	r.HandleFunc("/bookings/{id:[0-9]+}", booking.Get).Methods("GET")
	r.HandleFunc("/bookings", booking.Create).Methods("POST")
	r.HandleFunc("/bookings/{id:[0-9]+}", booking.Update).Methods("PUT")
	r.HandleFunc("/bookings/{id:[0-9]+}", booking.Delete).Methods("DELETE")

	// BookingUsers routes
	bookingPeople := handlerBookingPeople.New(db)
	r.HandleFunc("/bookingPeople", bookingPeople.GetAll).Methods("GET")
	r.HandleFunc("/bookingPeople/{id:[0-9]+}", bookingPeople.Get).Methods("GET")
	r.HandleFunc("/bookingPeople", bookingPeople.Create).Methods("POST")
	r.HandleFunc("/bookingPeople/{id:[0-9]+}", bookingPeople.Update).Methods("PUT")
	r.HandleFunc("/bookingPeople/{id:[0-9]+}", bookingPeople.Delete).Methods("DELETE")
	r.HandleFunc("/bookings/{id:[0-9]+}/people", bookingPeople.GetList).Methods("GET")

	// BookingStatus routes
	bookingStatus := handlerBookingStatus.New(db)
	r.HandleFunc("/bookingStatus", bookingStatus.GetAll).Methods("GET")
	r.HandleFunc("/bookingStatus/{id:[0-9]+}", bookingStatus.Get).Methods("GET")
	r.HandleFunc("/bookingStatus", bookingStatus.Create).Methods("POST")
	r.HandleFunc("/bookingStatus/{id:[0-9]+}", bookingStatus.Update).Methods("PUT")
	r.HandleFunc("/bookingStatus/{id:[0-9]+}", bookingStatus.Delete).Methods("DELETE")

	// Trip routes
	trip := handlerTrip.New(db)
	r.HandleFunc("/trips", trip.GetAll).Methods("GET")
	r.HandleFunc("/trips/{id:[0-9]+}", trip.Get).Methods("GET")
	r.HandleFunc("/trips", trip.Create).Methods("POST")
	r.HandleFunc("/trips/{id:[0-9]+}", trip.Update).Methods("PUT")
	r.HandleFunc("/trips/{id:[0-9]+}", trip.Delete).Methods("DELETE")

	// TripBooking routes
	tripBooking := handlerTripBooking.New(db)
	r.HandleFunc("/tripBooking", tripBooking.GetAll).Methods("GET")
	r.HandleFunc("/tripBooking/{id:[0-9]+}", tripBooking.Get).Methods("GET")
	r.HandleFunc("/tripBooking", tripBooking.Create).Methods("POST")
	r.HandleFunc("/tripBooking/{id:[0-9]+}", tripBooking.Update).Methods("PUT")
	r.HandleFunc("/tripBooking/{id:[0-9]+}", tripBooking.Delete).Methods("DELETE")
	r.HandleFunc("/trips/{id:[0-9]+}/bookings", tripBooking.GetList).Methods("GET")

	// Define CORS options
	corsOpts := handlers.CORS(
		handlers.AllowedOrigins([]string{"http://localhost:8081"}),        // Allow requests from http://localhost:8080
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE"}), // Allowed HTTP methods
		handlers.AllowedHeaders([]string{"Content-Type"}),                 // Allowed headers
	)

	log.Println("Server running on port 8085")
	log.Fatal(http.ListenAndServe(":8085", corsOpts(r))) // Apply CORS middleware

	//log.Println("Server running on port 8085")
	//log.Fatal(http.ListenAndServe(":8085", r))
}
