package appCore

/*

import (
	"api-server/v2/localHandlers/handlerAuth"
	"api-server/v2/localHandlers/handlerSeasons"
	"api-server/v2/localHandlers/handlerUser"
	"api-server/v2/localHandlers/handlerUserAgeGroups"
)

type Config struct {
	auth          handlerAuth.Handler
	seasons       handlerSeasons.Handler
	user          handlerUser.Handler
	userAgeGroups handlerUserAgeGroups.Handler
}


func MuxNew(appConf *Config) Config {
	//r := mux.NewRouter()
	m := mux.NewRouter()
	r := m.PathPrefix("/api/v1").Subrouter()
	auth := handlerAuth.New(appConf)
	r.Use(auth.RequireRestAuthTst) // Add some middleware, e.g. an auth handler

	//SRP authentication and registration process handlers
	r.HandleFunc("/api/v1/auth/register/", auth.AccountCreate).Methods("Post")
	r.HandleFunc("/api/v1/auth/{username}/salt/", auth.AuthGetSalt).Methods("Get", "Options")
	r.HandleFunc("/api/v1/auth/{username}/key/{A}", auth.AuthGetKeyB).Methods("Get")
	r.HandleFunc("/api/v1/auth/proof/", auth.AuthCheckClientProof).Methods("Post")
	r.HandleFunc("/api/v1/auth/validate/{token}", auth.AccountValidate).Methods("Get")
	r.HandleFunc("/api/v1/auth/reset/{username}/password/", auth.AuthReset).Methods("Get")
	r.HandleFunc("/api/v1/auth/reset/{token}/token/", auth.AuthUpdate).Methods("Post")

	// Seasons routes
	seasons := handlerSeasons.New(appConf)
	r.HandleFunc("/seasons", seasons.GetAll).Methods("GET")
	r.HandleFunc("/seasons/{id}", seasons.Get).Methods("GET")
	r.HandleFunc("/seasons", seasons.Create).Methods("POST")
	r.HandleFunc("/seasons/{id}", seasons.Update).Methods("PUT")
	r.HandleFunc("/seasons/{id}", seasons.Delete).Methods("DELETE")

	// User routes
	user := handlerUser.New(appConf)
	r.HandleFunc("/users", user.GetAll).Methods("GET")
	r.HandleFunc("/users/{id}", user.Get).Methods("GET")
	r.HandleFunc("/users", user.Create).Methods("POST")
	r.HandleFunc("/users/{id}", user.Update).Methods("PUT")
	r.HandleFunc("/users/{id}", user.Delete).Methods("DELETE")

	// UserCategory routes
	userAgeGroups := handlerUserAgeGroups.New(appConf)
	r.HandleFunc("/userAgeGroups", userAgeGroups.GetAll).Methods("GET")
	r.HandleFunc("/userAgeGroups/{id}", userAgeGroups.Get).Methods("GET")
	r.HandleFunc("/userAgeGroups", userAgeGroups.Create).Methods("POST")
	r.HandleFunc("/userAgeGroups/{id}", userAgeGroups.Update).Methods("PUT")
	r.HandleFunc("/userAgeGroups/{id}", userAgeGroups.Delete).Methods("DELETE")

	// UserPayments routes
	userPayments := handlerUserPayments.New(appConf)
	r.HandleFunc("/userPayments", userPayments.GetAll).Methods("GET")
	r.HandleFunc("/userPayments/{id}", userPayments.Get).Methods("GET")
	r.HandleFunc("/userPayments", userPayments.Create).Methods("POST")
	r.HandleFunc("/userPayments/{id}", userPayments.Update).Methods("PUT")
	r.HandleFunc("/userPayments/{id}", userPayments.Delete).Methods("DELETE")

	// UserStatus routes
	userStatus := handlerUserStatus.New(appConf)
	r.HandleFunc("/userStatus", userStatus.GetAll).Methods("GET")
	r.HandleFunc("/userStatus/{id}", userStatus.Get).Methods("GET")
	r.HandleFunc("/userStatus", userStatus.Create).Methods("POST")
	r.HandleFunc("/userStatus/{id}", userStatus.Update).Methods("PUT")
	r.HandleFunc("/userStatus/{id}", userStatus.Delete).Methods("DELETE")

	// Booking routes
	booking := handlerBooking.New(appConf)
	r.HandleFunc("/bookings", booking.GetAll).Methods("GET")
	r.HandleFunc("/bookings/{id:[0-9]+}", booking.Get).Methods("GET")
	r.HandleFunc("/bookings", booking.Create).Methods("POST")
	r.HandleFunc("/bookings/{id:[0-9]+}", booking.Update).Methods("PUT")
	r.HandleFunc("/bookings/{id:[0-9]+}", booking.Delete).Methods("DELETE")
	r.HandleFunc("/trips/{id:[0-9]+}/bookings", booking.GetList).Methods("GET")

	// BookingUsers routes
	bookingPeople := handlerBookingPeople.New(appConf)
	r.HandleFunc("/bookingPeople", bookingPeople.GetAll).Methods("GET")
	r.HandleFunc("/bookingPeople/{id:[0-9]+}", bookingPeople.Get).Methods("GET")
	r.HandleFunc("/bookingPeople", bookingPeople.Create).Methods("POST")
	r.HandleFunc("/bookingPeople/{id:[0-9]+}", bookingPeople.Update).Methods("PUT")
	r.HandleFunc("/bookingPeople/{id:[0-9]+}", bookingPeople.Delete).Methods("DELETE")
	r.HandleFunc("/bookings/{id:[0-9]+}/bookingPeople", bookingPeople.GetList).Methods("GET")

	// GroupBookings routes
	groupBooking := handlerGroupBooking.New(appConf)
	r.HandleFunc("/groupBooking", groupBooking.GetAll).Methods("GET")
	r.HandleFunc("/groupBooking/{id:[0-9]+}", groupBooking.Get).Methods("GET")
	r.HandleFunc("/groupBooking", groupBooking.Create).Methods("POST")
	r.HandleFunc("/groupBooking/{id:[0-9]+}", groupBooking.Update).Methods("PUT")
	r.HandleFunc("/groupBooking/{id:[0-9]+}", groupBooking.Delete).Methods("DELETE")

	// BookingStatus routes
	bookingStatus := handlerBookingStatus.New(appConf)
	r.HandleFunc("/bookingStatus", bookingStatus.GetAll).Methods("GET")
	r.HandleFunc("/bookingStatus/{id:[0-9]+}", bookingStatus.Get).Methods("GET")
	r.HandleFunc("/bookingStatus", bookingStatus.Create).Methods("POST")
	r.HandleFunc("/bookingStatus/{id:[0-9]+}", bookingStatus.Update).Methods("PUT")
	r.HandleFunc("/bookingStatus/{id:[0-9]+}", bookingStatus.Delete).Methods("DELETE")

	// Trip routes
	trip := handlerTrip.New(appConf)
	r.HandleFunc("/trips", trip.GetAll).Methods("GET")
	r.HandleFunc("/trips/{id:[0-9]+}", trip.Get).Methods("GET")
	r.HandleFunc("/trips", trip.Create).Methods("POST")
	r.HandleFunc("/trips/{id:[0-9]+}", trip.Update).Methods("PUT")
	r.HandleFunc("/trips/{id:[0-9]+}", trip.Delete).Methods("DELETE")
	r.HandleFunc("/trips/participantStatus", trip.GetParticipantStatus).Methods("GET")

	// TripType routes
	tripType := handlerTripType.New(appConf)
	r.HandleFunc("/tripType", tripType.GetAll).Methods("GET")
	r.HandleFunc("/tripType/{id:[0-9]+}", tripType.Get).Methods("GET")
	r.HandleFunc("/tripType", tripType.Create).Methods("POST")
	r.HandleFunc("/tripType/{id:[0-9]+}", tripType.Update).Methods("PUT")
	r.HandleFunc("/tripType/{id:[0-9]+}", tripType.Delete).Methods("DELETE")

	// TripStatus routes
	tripStatus := handlerTripStatus.New(appConf)
	r.HandleFunc("/tripStatus", tripStatus.GetAll).Methods("GET")
	r.HandleFunc("/tripStatus/{id:[0-9]+}", tripStatus.Get).Methods("GET")
	r.HandleFunc("/tripStatus", tripStatus.Create).Methods("POST")
	r.HandleFunc("/tripStatus/{id:[0-9]+}", tripStatus.Update).Methods("PUT")
	r.HandleFunc("/tripStatus/{id:[0-9]+}", tripStatus.Delete).Methods("DELETE")

	// TripDifficulty routes
	tripDifficulty := handlerTripDifficulty.New(appConf)
	r.HandleFunc("/tripDifficulty", tripDifficulty.GetAll).Methods("GET")
	r.HandleFunc("/tripDifficulty/{id:[0-9]+}", tripDifficulty.Get).Methods("GET")
	r.HandleFunc("/tripDifficulty", tripDifficulty.Create).Methods("POST")
	r.HandleFunc("/tripDifficulty/{id:[0-9]+}", tripDifficulty.Update).Methods("PUT")
	r.HandleFunc("/tripDifficulty/{id:[0-9]+}", tripDifficulty.Delete).Methods("DELETE")

	// TripCost routes
	tripCosts := handlerTripCost.New(appConf)
	r.HandleFunc("/tripCosts", tripCosts.GetAll).Methods("GET")
	r.HandleFunc("/tripCosts/{id:[0-9]+}", tripCosts.Get).Methods("GET")
	r.HandleFunc("/tripCosts", tripCosts.Create).Methods("POST")
	r.HandleFunc("/tripCosts/{id:[0-9]+}", tripCosts.Update).Methods("PUT")
	r.HandleFunc("/tripCosts/{id:[0-9]+}", tripCosts.Delete).Methods("DELETE")

	// TripCostGroup routes
	tripCostGroups := handlerTripCostGroup.New(appConf)
	r.HandleFunc("/tripCostGroups", tripCostGroups.GetAll).Methods("GET")
	r.HandleFunc("/tripCostGroups/{id:[0-9]+}", tripCostGroups.Get).Methods("GET")
	r.HandleFunc("/tripCostGroups", tripCostGroups.Create).Methods("POST")
	r.HandleFunc("/tripCostGroups/{id:[0-9]+}", tripCostGroups.Update).Methods("PUT")
	r.HandleFunc("/tripCostGroups/{id:[0-9]+}", tripCostGroups.Delete).Methods("DELETE")

	return Config{}
}
*/
