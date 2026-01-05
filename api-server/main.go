package main

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/handlerAccessLevel"
	"api-server/v2/localHandlers/handlerAccessType"
	"api-server/v2/localHandlers/handlerAuth"
	"api-server/v2/localHandlers/handlerBooking"
	"api-server/v2/localHandlers/handlerBookingPeople"
	"api-server/v2/localHandlers/handlerBookingStatus"
	"api-server/v2/localHandlers/handlerGroupBooking"
	"api-server/v2/localHandlers/handlerMemberStatus"
	"api-server/v2/localHandlers/handlerMyBookings"
	"api-server/v2/localHandlers/handlerOAuth"
	"api-server/v2/localHandlers/handlerResource"
	"api-server/v2/localHandlers/handlerSeasons"
	"api-server/v2/localHandlers/handlerSecurityGroup"
	"api-server/v2/localHandlers/handlerSecurityGroupResource"
	"api-server/v2/localHandlers/handlerSecurityUserGroup"
	"api-server/v2/localHandlers/handlerTrip"
	"api-server/v2/localHandlers/handlerTripCost"
	"api-server/v2/localHandlers/handlerTripCostGroup"
	"api-server/v2/localHandlers/handlerTripDifficulty"
	"api-server/v2/localHandlers/handlerTripStatus"
	"api-server/v2/localHandlers/handlerTripType"
	"api-server/v2/localHandlers/handlerUser"
	"api-server/v2/localHandlers/handlerUserAccountStatus"
	"api-server/v2/localHandlers/handlerUserAgeGroups"
	"api-server/v2/localHandlers/handlerUserPayments"
	"api-server/v2/localHandlers/helpers"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

const debugTag = "main."

func main() {
	app := appCore.New(true)
	//app.Settings.LoadEnv()
	app.Run()
	defer app.Close()
	log.Printf("%smain() App settings: %+v, os Env: %+v\n", debugTag, app.Settings, os.Environ())

	r := mux.NewRouter()
	// Setup your API subrouter with CORS middleware
	subR1 := r.PathPrefix(os.Getenv("API_PATH_PREFIX")).Subrouter()
	// The following routes are unprotected, i.e. do not require authentication to use.

	//SRP authentication and registration process handlers
	//handlerSRPAuth.New(app).RegisterRoutes(subR1, "/srpAuth")
	//subR1.HandleFunc("/srpAuth/sessioncheck/", auth.SessionCheck).Methods("Get")

	// WebAuthn handlers
	//handlerWebAuthn.New(app).RegisterRoutes(subR1, "/webauthn")

	// OAuth handlers
	oauth := handlerOAuth.New(app)
	oauth.RegisterRoutes(subR1, "/auth/oauth") // OAuth handlers

	subR2 := r.PathPrefix(os.Getenv("API_PATH_PREFIX")).Subrouter()
	auth := handlerAuth.New(app)
	auth.RegisterRoutes(subR2, "/auth")
	//subR2.Use(SRPauth.RequireRestAuth) // Add some middleware, e.g. an auth handler
	//subR2.Use(auth.RequireRestAuth)           // Add some middleware, e.g. an auth handler
	subR2.Use(auth.RequireOAuthOrSessionAuth) // Add some middleware, e.g. an auth handler

	// The following routes are protected, i.e. require authentication to use.
	oauth.RegisterRoutesProtected(subR2, "/auth/oauth") // Protected OAuth handlers

	// Add route groups (protected)
	//addRouteGroup(subR2, "webauthn", handlerWebAuthnManagement.New(app))                 // WebAuthn routes
	addRouteGroup(subR2, "seasons", handlerSeasons.New(app))                             // Seasons routes
	handlerUser.New(app).RegisterRoutes(subR2, "/users")                                 // User routes
	addRouteGroup(subR2, "userAgeGroups", handlerUserAgeGroups.New(app))                 // UserAgeGroup routes
	addRouteGroup(subR2, "userPayments", handlerUserPayments.New(app))                   // UserPayments routes
	addRouteGroup(subR2, "userMemberStatus", handlerMemberStatus.New(app))               // UserMemberStatus routes
	addRouteGroup(subR2, "userAccountStatus", handlerUserAccountStatus.New(app))         // UserAccountStatus routes
	addRouteGroup(subR2, "groupBooking", handlerGroupBooking.New(app))                   // GroupBookings routes
	addRouteGroup(subR2, "bookingStatus", handlerBookingStatus.New(app))                 // BookingStatus routes
	addRouteGroup(subR2, "tripType", handlerTripType.New(app))                           // TripType routes
	addRouteGroup(subR2, "tripStatus", handlerTripStatus.New(app))                       // TripStatus routes
	addRouteGroup(subR2, "tripDifficulty", handlerTripDifficulty.New(app))               // TripDifficulty routes
	addRouteGroup(subR2, "tripCosts", handlerTripCost.New(app))                          // TripCost routes
	addRouteGroup(subR2, "tripCostGroups", handlerTripCostGroup.New(app))                // TripCostGroup routes
	addRouteGroup(subR2, "securityGroup", handlerSecurityGroup.New(app))                 // SecurityGroup routes
	addRouteGroup(subR2, "securityGroupResource", handlerSecurityGroupResource.New(app)) // SecurityGroupResource routes
	addRouteGroup(subR2, "securityUserGroup", handlerSecurityUserGroup.New(app))         // SecurityUserGroup routes
	addRouteGroup(subR2, "securityAccessLevel", handlerAccessLevel.New(app))             // AccessLevel routes
	addRouteGroup(subR2, "securityAccessType", handlerAccessType.New(app))               // AccessType routes
	addRouteGroup(subR2, "securityResource", handlerResource.New(app))                   // Resource routes
	addRouteGroup(subR2, "myBookings", handlerMyBookings.New(app))                       // Resource routes

	booking := handlerBooking.New(app)                                              // Booking routes
	addRouteGroup(subR2, "bookings", booking)                                       // Booking routes
	subR2.HandleFunc("/trips/{id:[0-9]+}/bookings", booking.GetList).Methods("GET") // Booking routes

	bookingPeople := handlerBookingPeople.New(app)                                                // BookingPeople routes
	addRouteGroup(subR2, "bookingPeople", bookingPeople)                                          // BookingPeople routes
	subR2.HandleFunc("/bookings/{id:[0-9]+}/bookingPeople", bookingPeople.GetList).Methods("GET") // BookingPeople routes

	handlerTrip.New(app).RegisterRoutes(subR2, "/trips") // Trip routes

	// Static handlers
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("./static")))) // Serve static files from the "/static" directory under the url "/"

	// For debugging: Log all registered routes
	subR1.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		log.Printf("Registered routes for subR1: %s %v", path, methods)
		return nil
	})
	subR2.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		log.Printf("Registered routes for subR2: %s %v", path, methods)
		return nil
	})

	// Define CORS options
	corsOpts := handlers.CORS(
		handlers.AllowedOrigins([]string{"http://localhost:8081", "https://localhost:8081", "http://localhost:8085", "https://localhost:8086"}), // Allow requests from http://localhost:8080 //w.Header().Set("Access-Control-Allow-Origin", "http://localhost") // "http://localhost:8081" // or "*" if you want to test without restrictions
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),                                                            // Allowed HTTP methods
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization", "Access-Control-Allow-Credentials"}),                                  // Allowed headers
		handlers.AllowCredentials(), // Headers([]string{"Content-Type"}) //w.Header().Set("Access-Control-Allow-Credentials", "true")
	)

	corsMuxHandler := corsOpts(r)
	loggedHandler := helpers.LogRequest(corsMuxHandler, app.SessionIDKey) // Wrap the router with the logging middleware

	// Paths to certificate and key files
	crtFile := app.Settings.ServerCert
	keyFile := app.Settings.ServerKey

	go func() {
		log.Println(debugTag + "HTTP server running on http://localhost:8085")
		if err := http.ListenAndServe(":8085", loggedHandler); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	go func() {
		log.Println(debugTag + "HTTPS server running on https://localhost:8086")
		if err := http.ListenAndServeTLS(":8086", crtFile, keyFile, loggedHandler); err != nil {
			log.Fatalf("HTTPS server error: %v", err)
		}
	}()

	// Block the main goroutine to keep the servers running
	select {}
}

type genericHandler interface {
	GetAll(w http.ResponseWriter, r *http.Request)
	Get(w http.ResponseWriter, r *http.Request)
	Create(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	Delete(w http.ResponseWriter, r *http.Request)
}

func addRouteGroup(r *mux.Router, resourcePath string, handler genericHandler) {
	r.HandleFunc("/"+resourcePath, handler.GetAll).Methods("GET")
	r.HandleFunc("/"+resourcePath+"/{id:[0-9]+}", handler.Get).Methods("GET")
	r.HandleFunc("/"+resourcePath, handler.Create).Methods("POST")
	r.HandleFunc("/"+resourcePath+"/{id:[0-9]+}", handler.Update).Methods("PUT")
	r.HandleFunc("/"+resourcePath+"/{id:[0-9]+}", handler.Delete).Methods("DELETE")
	// Add some code to register the route resource for managing security access
}
