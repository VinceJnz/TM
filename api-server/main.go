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
	debugFlag := true
	app := appCore.New(debugFlag)
	app.Run()
	defer app.Close()
	log.Printf("%smain(), debug: %v, App settings: %+v, os Env: %+v\n", debugTag, debugFlag, app.Settings, os.Environ())

	r := mux.NewRouter()
	// *****************************************************
	// Setup your API subrouter with CORS middleware. These routes are unprotected, i.e. do not require authentication to use. This is where you would register any routes that should be accessible without authentication, such as login or registration endpoints.
	public := r.PathPrefix(os.Getenv("API_PATH_PREFIX")).Subrouter()
	public.Use(func(next http.Handler) http.Handler {
		return helpers.LogRequest(next, app.SessionIDKey, "public") // First logging
	})

	// OAuth handlers
	oauth := handlerOAuth.New(app)
	oauth.RegisterRoutes(public, "/auth/oauth") // OAuth handlers

	// Auth handlers - Public routes (no authentication required)
	auth := handlerAuth.New(app)
	auth.RegisterRoutesPublic(public, "/auth") // Public auth endpoints (register, verify, login, etc.)

	// *****************************************************
	// Protected routes (require authentication) - These are registered on a subrouter so that the auth middleware is only applied to these routes and not the public routes above. This allows us to have a mix of protected and unprotected routes.
	protected := r.PathPrefix(os.Getenv("API_PATH_PREFIX")).Subrouter()
	protected.Use(auth.RequireSessionAuth) // This needs to be here to protect the routes below. It checks for a valid session cookie and ensures the user has access to the requested resource. If the session is valid and access is granted, it allows the request to proceed to the appropriate handler. If not, it returns an unauthorized error.
	protected.Use(func(next http.Handler) http.Handler {
		return helpers.LogRequest(next, app.SessionIDKey, "protected") // Then logging
	})
	adminProtected := protected.PathPrefix("").Subrouter()
	adminProtected.Use(auth.RequireRole("admin"))
	sysadminProtected := protected.PathPrefix("").Subrouter()
	sysadminProtected.Use(auth.RequireRole("sysadmin"))

	// Auth handlers - Protected routes (requires authentication)
	auth.RegisterRoutesProtected(protected, "/auth") // Protected auth endpoints (menuUser, logout, etc.)

	// The following routes are protected, i.e. require authentication to use.
	oauth.RegisterRoutesProtected(protected, "/auth/oauth") // Protected OAuth handlers

	// Add route groups (protected - user and above)
	handlerMyBookings.New(app).RegisterRoutes(protected, "/myBookings")       // MyBookings routes
	handlerBooking.New(app).RegisterRoutes(protected, "/bookings", "/trips")  // Booking routes
	handlerBookingPeople.New(app).RegisterRoutes(protected, "/bookingPeople") // BookingPeople routes
	handlerTrip.New(app).RegisterRoutes(protected, "/trips")                  // Trip routes

	// Add route groups (admin and above)
	handlerSeasons.New(app).RegisterRoutes(adminProtected, "/seasons")                     // Seasons routes
	handlerUser.New(app).RegisterRoutes(adminProtected, "/users")                          // User routes
	handlerUserAgeGroups.New(app).RegisterRoutes(adminProtected, "/userAgeGroups")         // UserAgeGroup routes
	handlerUserPayments.New(app).RegisterRoutes(adminProtected, "/userPayments")           // UserPayments routes
	handlerMemberStatus.New(app).RegisterRoutes(adminProtected, "/userMemberStatus")       // UserMemberStatus routes
	handlerUserAccountStatus.New(app).RegisterRoutes(adminProtected, "/userAccountStatus") // UserAccountStatus routes
	handlerGroupBooking.New(app).RegisterRoutes(adminProtected, "/groupBooking")           // GroupBookings routes
	handlerBookingStatus.New(app).RegisterRoutes(adminProtected, "/bookingStatus")         // BookingStatus routes
	handlerTripType.New(app).RegisterRoutes(adminProtected, "/tripType")                   // TripType routes
	handlerTripStatus.New(app).RegisterRoutes(adminProtected, "/tripStatus")               // TripStatus routes
	handlerTripDifficulty.New(app).RegisterRoutes(adminProtected, "/tripDifficulty")       // TripDifficulty routes
	handlerTripCost.New(app).RegisterRoutes(adminProtected, "/tripCosts")                  // TripCost routes
	handlerTripCostGroup.New(app).RegisterRoutes(adminProtected, "/tripCostGroups")        // TripCostGroup routes
	handlerResource.New(app).RegisterRoutes(adminProtected, "/securityResource")           // Resource routes

	// Add route groups (sysadmin only)
	handlerSecurityGroup.New(app).RegisterRoutes(sysadminProtected, "/securityGroup")                 // SecurityGroup routes
	handlerSecurityGroupResource.New(app).RegisterRoutes(sysadminProtected, "/securityGroupResource") // SecurityGroupResource routes
	handlerSecurityUserGroup.New(app).RegisterRoutes(sysadminProtected, "/securityUserGroup")         // SecurityUserGroup routes
	handlerAccessLevel.New(app).RegisterRoutes(sysadminProtected, "/securityAccessLevel")             // AccessLevel routes
	handlerAccessType.New(app).RegisterRoutes(sysadminProtected, "/securityAccessType")               // AccessType routes

	// Static handlers
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("./static")))) // Serve static files from the "/static" directory under the url "/"

	/*
		//if debugFlag {
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
		//}
	*/

	// Define CORS options
	corsOpts := handlers.CORS(
		handlers.AllowedOrigins([]string{"http://localhost:8081", "https://localhost:8081", "http://localhost:8085", "https://localhost:8086"}), // Allow requests from http://localhost:8080 //w.Header().Set("Access-Control-Allow-Origin", "http://localhost") // "http://localhost:8081" // or "*" if you want to test without restrictions
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),                                                            // Allowed HTTP methods
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization", "Access-Control-Allow-Credentials"}),                                  // Allowed headers
		handlers.AllowedOrigins([]string{"http://localhost:8081", "https://localhost:8081", "https://checkout.stripe.com"}),                     // Add Stripe
		handlers.AllowCredentials(), // Headers([]string{"Content-Type"}) //w.Header().Set("Access-Control-Allow-Credentials", "true")
	)

	corsMuxHandler := corsOpts(r)
	loggedHandler := corsMuxHandler

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
