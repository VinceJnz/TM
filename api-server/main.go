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
	public := r.PathPrefix(os.Getenv("API_PATH_PREFIX")).Subrouter()
	// The following routes are unprotected, i.e. do not require authentication to use.

	//SRP authentication and registration process handlers
	//handlerSRPAuth.New(app).RegisterRoutes(public, "/srpAuth")
	//public.HandleFunc("/srpAuth/sessioncheck/", auth.SessionCheck).Methods("Get")

	// WebAuthn handlers
	//handlerWebAuthn.New(app).RegisterRoutes(public, "/webauthn")

	// OAuth handlers
	oauth := handlerOAuth.New(app)
	oauth.RegisterRoutes(public, "/auth/oauth") // OAuth handlers

	protected := r.PathPrefix(os.Getenv("API_PATH_PREFIX")).Subrouter()
	auth := handlerAuth.New(app)
	auth.RegisterRoutes(protected, "/auth")
	//protected.Use(SRPauth.RequireRestAuth) // Add some middleware, e.g. an auth handler
	//protected.Use(auth.RequireRestAuth)           // Add some middleware, e.g. an auth handler
	protected.Use(auth.RequireOAuthOrSessionAuth) // Add some middleware, e.g. an auth handler

	// The following routes are protected, i.e. require authentication to use.
	oauth.RegisterRoutesProtected(protected, "/auth/oauth") // Protected OAuth handlers

	// Add route groups (protected)
	//addRouteGroup(protected, "webauthn", handlerWebAuthnManagement.New(app))                 // WebAuthn routes
	handlerSeasons.New(app).RegisterRoutes(protected, "/seasons")                             // Seasons routes
	handlerUser.New(app).RegisterRoutes(protected, "/users")                                  // User routes
	handlerUserAgeGroups.New(app).RegisterRoutes(protected, "/userAgeGroups")                 // UserAgeGroup routes
	handlerUserPayments.New(app).RegisterRoutes(protected, "/userPayments")                   // UserPayments routes
	handlerMemberStatus.New(app).RegisterRoutes(protected, "/userMemberStatus")               // UserMemberStatus routes
	handlerUserAccountStatus.New(app).RegisterRoutes(protected, "/userAccountStatus")         // UserAccountStatus routes
	handlerGroupBooking.New(app).RegisterRoutes(protected, "/groupBooking")                   // GroupBookings routes
	handlerBookingStatus.New(app).RegisterRoutes(protected, "/bookingStatus")                 // BookingStatus routes
	handlerTripType.New(app).RegisterRoutes(protected, "/tripType")                           // TripType routes
	handlerTripStatus.New(app).RegisterRoutes(protected, "/tripStatus")                       // TripStatus routes
	handlerTripDifficulty.New(app).RegisterRoutes(protected, "/tripDifficulty")               // TripDifficulty routes
	handlerTripCost.New(app).RegisterRoutes(protected, "/tripCosts")                          // TripCost routes
	handlerTripCostGroup.New(app).RegisterRoutes(protected, "/tripCostGroups")                // TripCostGroup routes
	handlerSecurityGroup.New(app).RegisterRoutes(protected, "/securityGroup")                 // SecurityGroup routes
	handlerSecurityGroupResource.New(app).RegisterRoutes(protected, "/securityGroupResource") // SecurityGroupResource routes
	handlerSecurityUserGroup.New(app).RegisterRoutes(protected, "/securityUserGroup")         // SecurityUserGroup routes
	handlerAccessLevel.New(app).RegisterRoutes(protected, "/securityAccessLevel")             // AccessLevel routes
	handlerAccessType.New(app).RegisterRoutes(protected, "/securityAccessType")               // AccessType routes
	handlerResource.New(app).RegisterRoutes(protected, "/securityResource")                   // Resource routes
	handlerMyBookings.New(app).RegisterRoutes(protected, "/myBookings")                       // Resource routes
	handlerBooking.New(app).RegisterRoutes(protected, "/bookings")                            // Booking routes
	handlerBookingPeople.New(app).RegisterRoutes(protected, "/bookingPeople")                 // BookingPeople routes
	handlerTrip.New(app).RegisterRoutes(protected, "/trips")                                  // Trip routes
	//subR2.HandleFunc("/trips/{id:[0-9]+}/bookings", booking.GetList).Methods("GET") // Booking routes

	// Static handlers
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("./static")))) // Serve static files from the "/static" directory under the url "/"

	// For debugging: Log all registered routes
	public.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		log.Printf("Registered routes for public: %s %v", path, methods)
		return nil
	})
	protected.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		log.Printf("Registered routes for protected: %s %v", path, methods)
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
