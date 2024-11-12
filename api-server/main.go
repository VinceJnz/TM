package main

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/handlerAuth"
	"api-server/v2/localHandlers/handlerBooking"
	"api-server/v2/localHandlers/handlerBookingPeople"
	"api-server/v2/localHandlers/handlerBookingStatus"
	"api-server/v2/localHandlers/handlerGroupBooking"
	"api-server/v2/localHandlers/handlerSeasons"
	"api-server/v2/localHandlers/handlerTrip"
	"api-server/v2/localHandlers/handlerTripCost"
	"api-server/v2/localHandlers/handlerTripCostGroup"
	"api-server/v2/localHandlers/handlerTripDifficulty"
	"api-server/v2/localHandlers/handlerTripStatus"
	"api-server/v2/localHandlers/handlerTripType"
	"api-server/v2/localHandlers/handlerUser"
	"api-server/v2/localHandlers/handlerUserAgeGroups"
	"api-server/v2/localHandlers/handlerUserPayments"
	"api-server/v2/localHandlers/handlerUserStatus"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	app := appCore.New(true)
	defer app.Close()

	m := mux.NewRouter()
	r := m.PathPrefix(os.Getenv("API_PATH_PREFIX")).Subrouter()

	auth := handlerAuth.New(app)
	//SRP authentication and registration process handlers
	r.HandleFunc("/auth/register/", auth.AccountCreate).Methods("Post")
	r.HandleFunc("/auth/{username}/salt/", auth.AuthGetSalt).Methods("Get", "Options")
	r.HandleFunc("/auth/{username}/key/{A}", auth.AuthGetKeyB).Methods("Get")
	r.HandleFunc("/auth/proof/", auth.AuthCheckClientProof).Methods("Post")
	r.HandleFunc("/auth/validate/{token}", auth.AccountValidate).Methods("Get")
	r.HandleFunc("/auth/reset/{username}/password/", auth.AuthReset).Methods("Get")
	r.HandleFunc("/auth/reset/{token}/token/", auth.AuthUpdate).Methods("Post")
	r.HandleFunc("/auth/logout/", auth.AuthLogout).Methods("Post")

	//r.Use(auth.RequireRestAuth) // Add some middleware, e.g. an auth handler

	// Seasons routes
	seasons := handlerSeasons.New(app)
	r.HandleFunc("/seasons", seasons.GetAll).Methods("GET")
	r.HandleFunc("/seasons/{id}", seasons.Get).Methods("GET")
	r.HandleFunc("/seasons", seasons.Create).Methods("POST")
	r.HandleFunc("/seasons/{id}", seasons.Update).Methods("PUT")
	r.HandleFunc("/seasons/{id}", seasons.Delete).Methods("DELETE")

	// User routes
	user := handlerUser.New(app)
	r.HandleFunc("/users", user.GetAll).Methods("GET")
	r.HandleFunc("/users/{id}", user.Get).Methods("GET")
	r.HandleFunc("/users", user.Create).Methods("POST")
	r.HandleFunc("/users/{id}", user.Update).Methods("PUT")
	r.HandleFunc("/users/{id}", user.Delete).Methods("DELETE")

	// UserCategory routes
	userAgeGroups := handlerUserAgeGroups.New(app)
	r.HandleFunc("/userAgeGroups", userAgeGroups.GetAll).Methods("GET")
	r.HandleFunc("/userAgeGroups/{id}", userAgeGroups.Get).Methods("GET")
	r.HandleFunc("/userAgeGroups", userAgeGroups.Create).Methods("POST")
	r.HandleFunc("/userAgeGroups/{id}", userAgeGroups.Update).Methods("PUT")
	r.HandleFunc("/userAgeGroups/{id}", userAgeGroups.Delete).Methods("DELETE")

	// UserPayments routes
	userPayments := handlerUserPayments.New(app)
	r.HandleFunc("/userPayments", userPayments.GetAll).Methods("GET")
	r.HandleFunc("/userPayments/{id}", userPayments.Get).Methods("GET")
	r.HandleFunc("/userPayments", userPayments.Create).Methods("POST")
	r.HandleFunc("/userPayments/{id}", userPayments.Update).Methods("PUT")
	r.HandleFunc("/userPayments/{id}", userPayments.Delete).Methods("DELETE")

	// UserStatus routes
	userStatus := handlerUserStatus.New(app)
	r.HandleFunc("/userStatus", userStatus.GetAll).Methods("GET")
	r.HandleFunc("/userStatus/{id}", userStatus.Get).Methods("GET")
	r.HandleFunc("/userStatus", userStatus.Create).Methods("POST")
	r.HandleFunc("/userStatus/{id}", userStatus.Update).Methods("PUT")
	r.HandleFunc("/userStatus/{id}", userStatus.Delete).Methods("DELETE")

	// Booking routes
	booking := handlerBooking.New(app)
	r.HandleFunc("/bookings", booking.GetAll).Methods("GET")
	r.HandleFunc("/bookings/{id:[0-9]+}", booking.Get).Methods("GET")
	r.HandleFunc("/bookings", booking.Create).Methods("POST")
	r.HandleFunc("/bookings/{id:[0-9]+}", booking.Update).Methods("PUT")
	r.HandleFunc("/bookings/{id:[0-9]+}", booking.Delete).Methods("DELETE")
	r.HandleFunc("/trips/{id:[0-9]+}/bookings", booking.GetList).Methods("GET")

	// BookingUsers routes
	bookingPeople := handlerBookingPeople.New(app)
	r.HandleFunc("/bookingPeople", bookingPeople.GetAll).Methods("GET")
	r.HandleFunc("/bookingPeople/{id:[0-9]+}", bookingPeople.Get).Methods("GET")
	r.HandleFunc("/bookingPeople", bookingPeople.Create).Methods("POST")
	r.HandleFunc("/bookingPeople/{id:[0-9]+}", bookingPeople.Update).Methods("PUT")
	r.HandleFunc("/bookingPeople/{id:[0-9]+}", bookingPeople.Delete).Methods("DELETE")
	r.HandleFunc("/bookings/{id:[0-9]+}/bookingPeople", bookingPeople.GetList).Methods("GET")

	// GroupBookings routes
	groupBooking := handlerGroupBooking.New(app)
	r.HandleFunc("/groupBooking", groupBooking.GetAll).Methods("GET")
	r.HandleFunc("/groupBooking/{id:[0-9]+}", groupBooking.Get).Methods("GET")
	r.HandleFunc("/groupBooking", groupBooking.Create).Methods("POST")
	r.HandleFunc("/groupBooking/{id:[0-9]+}", groupBooking.Update).Methods("PUT")
	r.HandleFunc("/groupBooking/{id:[0-9]+}", groupBooking.Delete).Methods("DELETE")

	// BookingStatus routes
	bookingStatus := handlerBookingStatus.New(app)
	r.HandleFunc("/bookingStatus", bookingStatus.GetAll).Methods("GET")
	r.HandleFunc("/bookingStatus/{id:[0-9]+}", bookingStatus.Get).Methods("GET")
	r.HandleFunc("/bookingStatus", bookingStatus.Create).Methods("POST")
	r.HandleFunc("/bookingStatus/{id:[0-9]+}", bookingStatus.Update).Methods("PUT")
	r.HandleFunc("/bookingStatus/{id:[0-9]+}", bookingStatus.Delete).Methods("DELETE")

	// Trip routes
	trip := handlerTrip.New(app)
	r.HandleFunc("/trips", trip.GetAll).Methods("GET")
	r.HandleFunc("/trips/{id:[0-9]+}", trip.Get).Methods("GET")
	r.HandleFunc("/trips", trip.Create).Methods("POST")
	r.HandleFunc("/trips/{id:[0-9]+}", trip.Update).Methods("PUT")
	r.HandleFunc("/trips/{id:[0-9]+}", trip.Delete).Methods("DELETE")
	r.HandleFunc("/trips/participantStatus", trip.GetParticipantStatus).Methods("GET")

	// TripType routes
	tripType := handlerTripType.New(app)
	r.HandleFunc("/tripType", tripType.GetAll).Methods("GET")
	r.HandleFunc("/tripType/{id:[0-9]+}", tripType.Get).Methods("GET")
	r.HandleFunc("/tripType", tripType.Create).Methods("POST")
	r.HandleFunc("/tripType/{id:[0-9]+}", tripType.Update).Methods("PUT")
	r.HandleFunc("/tripType/{id:[0-9]+}", tripType.Delete).Methods("DELETE")

	// TripStatus routes
	tripStatus := handlerTripStatus.New(app)
	r.HandleFunc("/tripStatus", tripStatus.GetAll).Methods("GET")
	r.HandleFunc("/tripStatus/{id:[0-9]+}", tripStatus.Get).Methods("GET")
	r.HandleFunc("/tripStatus", tripStatus.Create).Methods("POST")
	r.HandleFunc("/tripStatus/{id:[0-9]+}", tripStatus.Update).Methods("PUT")
	r.HandleFunc("/tripStatus/{id:[0-9]+}", tripStatus.Delete).Methods("DELETE")

	// TripDifficulty routes
	tripDifficulty := handlerTripDifficulty.New(app)
	r.HandleFunc("/tripDifficulty", tripDifficulty.GetAll).Methods("GET")
	r.HandleFunc("/tripDifficulty/{id:[0-9]+}", tripDifficulty.Get).Methods("GET")
	r.HandleFunc("/tripDifficulty", tripDifficulty.Create).Methods("POST")
	r.HandleFunc("/tripDifficulty/{id:[0-9]+}", tripDifficulty.Update).Methods("PUT")
	r.HandleFunc("/tripDifficulty/{id:[0-9]+}", tripDifficulty.Delete).Methods("DELETE")

	// TripCost routes
	tripCosts := handlerTripCost.New(app)
	r.HandleFunc("/tripCosts", tripCosts.GetAll).Methods("GET")
	r.HandleFunc("/tripCosts/{id:[0-9]+}", tripCosts.Get).Methods("GET")
	r.HandleFunc("/tripCosts", tripCosts.Create).Methods("POST")
	r.HandleFunc("/tripCosts/{id:[0-9]+}", tripCosts.Update).Methods("PUT")
	r.HandleFunc("/tripCosts/{id:[0-9]+}", tripCosts.Delete).Methods("DELETE")

	// TripCostGroup routes
	tripCostGroups := handlerTripCostGroup.New(app)
	r.HandleFunc("/tripCostGroups", tripCostGroups.GetAll).Methods("GET")
	r.HandleFunc("/tripCostGroups/{id:[0-9]+}", tripCostGroups.Get).Methods("GET")
	r.HandleFunc("/tripCostGroups", tripCostGroups.Create).Methods("POST")
	r.HandleFunc("/tripCostGroups/{id:[0-9]+}", tripCostGroups.Update).Methods("PUT")
	r.HandleFunc("/tripCostGroups/{id:[0-9]+}", tripCostGroups.Delete).Methods("DELETE")

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
