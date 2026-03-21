package main

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/handlerAccessLevel"
	"api-server/v2/localHandlers/handlerAccessScope"
	"api-server/v2/localHandlers/handlerAuth"
	"api-server/v2/localHandlers/handlerBooking"
	"api-server/v2/localHandlers/handlerBookingPeople"
	"api-server/v2/localHandlers/handlerBookingStatus"
	"api-server/v2/localHandlers/handlerBookingVoucher"
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
	"api-server/v2/modelMethods/dbAuthTemplate"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
)

const debugTag = "main."

func parseOrigins(csv string) []string {
	origins := make([]string, 0)
	for _, entry := range strings.Split(csv, ",") {
		origin := strings.TrimSpace(entry)
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 {
		return []string{"http://localhost:8086"}
	}
	return origins
}

// startTokenCleanupJob runs token cleanup every 15 minutes in the background.
// This removes expired tokens from the database to prevent storage bloat and maintain security.
// Token cleanup was moved from the hot read path (FindSessionToken/FindToken) to this
// background job to improve performance and reduce database write pressure on authenticated requests.
func startTokenCleanupJob(db *sqlx.DB) {
	ticker := time.NewTicker(15 * time.Minute)
	go func() {
		// Run cleanup immediately on startup
		if err := dbAuthTemplate.TokenCleanExpired("background-cleanup-startup", db); err != nil {
			log.Printf("%sToken cleanup job (startup) failed: %v", debugTag, err)
		} else {
			log.Printf("%sToken cleanup job (startup) completed successfully", debugTag)
		}

		// Then run on schedule
		for range ticker.C {
			if err := dbAuthTemplate.TokenCleanExpired("background-cleanup", db); err != nil {
				log.Printf("%sToken cleanup job failed: %v", debugTag, err)
			} else {
				log.Printf("%sToken cleanup job completed successfully", debugTag)
			}
		}
	}()
	log.Printf("%sToken cleanup background job started (runs every 15 minutes)", debugTag)
}

func main() {
	debugFlag := true
	app := appCore.New(debugFlag)
	app.Run()
	defer app.Close()

	// Start background jobs
	startTokenCleanupJob(app.Db)

	r := mux.NewRouter()
	// *****************************************************
	// Setup your API subrouter with CORS middleware. These routes are unprotected, i.e. do not require authentication to use. This is where you would register any routes that should be accessible without authentication, such as login or registration endpoints.
	public := r.PathPrefix(app.Settings.APIprefix).Subrouter()
	public.Use(func(next http.Handler) http.Handler {
		return helpers.LogRequest(next, app.SessionIDKey, "public") // First logging
	})

	// OAuth handlers
	oauth := handlerOAuth.New(app)
	oauth.RegisterRoutes(public, "/auth/oauth") // OAuth handlers

	// Auth handlers - Public routes (no authentication required)
	auth := handlerAuth.New(app)
	auth.RegisterRoutesPublic(public, "/auth") // Public auth endpoints (register, verify, login, etc.)
	handlerBooking.New(app).RegisterRoutesStripeWebhook(public, "/bookings")

	// *****************************************************
	// Protected routes (require authentication) - These are registered on a subrouter so that the auth middleware is only applied to these routes and not the public routes above. This allows us to have a mix of protected and unprotected routes.
	protected := r.PathPrefix(app.Settings.APIprefix).Subrouter()
	protected.Use(auth.RequireSessionAuth) // This needs to be here to protect the routes below. It checks for a valid session cookie and ensures the user has access to the requested resource. If the session is valid and access is granted, it allows the request to proceed to the appropriate handler. If not, it returns an unauthorized error.
	protected.Use(func(next http.Handler) http.Handler {
		return helpers.LogRequest(next, app.SessionIDKey, "protected") // Then logging
	})

	// Auth handlers - Protected routes (requires authentication)
	auth.RegisterRoutesProtected(protected, "/auth") // Protected auth endpoints (menuUser, logout, etc.)

	// The following routes are protected, i.e. require authentication to use.
	oauth.RegisterRoutesProtected(protected, "/auth/oauth") // Protected OAuth handlers

	// Add route groups (protected - user and above)
	handlerMyBookings.New(app).RegisterRoutes(protected, "/myBookings")       // MyBookings routes
	handlerBooking.New(app).RegisterRoutes(protected, "/bookings", "/trips")  // Booking routes
	handlerBookingPeople.New(app).RegisterRoutes(protected, "/bookingPeople") // BookingPeople routes
	handlerBookingVoucher.New(app).RegisterRoutesProtected(protected, "/bookingVouchers")
	handlerTrip.New(app).RegisterRoutes(protected, "/trips") // Trip routes

	// Add route groups previously role-gated; capability checks are enforced by RequireSessionAuth.
	handlerSeasons.New(app).RegisterRoutes(protected, "/seasons")                     // Seasons routes
	handlerUser.New(app).RegisterRoutes(protected, "/users")                          // User routes
	handlerUserAgeGroups.New(app).RegisterRoutes(protected, "/userAgeGroups")         // UserAgeGroup routes
	handlerUserPayments.New(app).RegisterRoutes(protected, "/userPayments")           // UserPayments routes
	handlerMemberStatus.New(app).RegisterRoutes(protected, "/userMemberStatus")       // UserMemberStatus routes
	handlerUserAccountStatus.New(app).RegisterRoutes(protected, "/userAccountStatus") // UserAccountStatus routes
	handlerGroupBooking.New(app).RegisterRoutes(protected, "/groupBooking")           // GroupBookings routes
	handlerBookingStatus.New(app).RegisterRoutes(protected, "/bookingStatus")         // BookingStatus routes
	handlerBookingVoucher.New(app).RegisterRoutesAdmin(protected, "/bookingVouchers")
	handlerTripType.New(app).RegisterRoutes(protected, "/tripType")             // TripType routes
	handlerTripStatus.New(app).RegisterRoutes(protected, "/tripStatus")         // TripStatus routes
	handlerTripDifficulty.New(app).RegisterRoutes(protected, "/tripDifficulty") // TripDifficulty routes
	handlerTripCost.New(app).RegisterRoutes(protected, "/tripCosts")            // TripCost routes
	handlerTripCostGroup.New(app).RegisterRoutes(protected, "/tripCostGroups")  // TripCostGroup routes
	handlerResource.New(app).RegisterRoutes(protected, "/securityResource")     // Resource routes

	handlerSecurityGroup.New(app).RegisterRoutes(protected, "/securityGroup")                 // SecurityGroup routes
	handlerSecurityGroupResource.New(app).RegisterRoutes(protected, "/securityGroupResource") // SecurityGroupResource routes
	handlerSecurityUserGroup.New(app).RegisterRoutes(protected, "/securityUserGroup")         // SecurityUserGroup routes
	handlerAccessLevel.New(app).RegisterRoutes(protected, "/securityAccessLevel")             // AccessLevel routes
	handlerAccessScope.New(app).RegisterRoutes(protected, "/securityAccessScope")             // AccessScope routes
	handlerAccessScope.New(app).RegisterRoutes(protected, "/securityAccessType")              // Legacy alias for AccessScope routes

	// Static handlers
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("./static")))) // Serve static files from the "/static" directory under the url "/"

	// Define CORS options
	allowedOrigins := parseOrigins(app.Settings.CoreOrigins)
	log.Printf("%sCORS allowed origins: %v", debugTag, allowedOrigins)
	corsOpts := handlers.CORS(
		handlers.AllowedOrigins(allowedOrigins),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization", "Access-Control-Allow-Credentials"}),
		handlers.AllowCredentials(), // Headers([]string{"Content-Type"}) //w.Header().Set("Access-Control-Allow-Credentials", "true")
	)

	corsMuxHandler := corsOpts(r)
	loggedHandler := corsMuxHandler

	// Paths to certificate and key files
	crtFile := app.Settings.ServerCert
	keyFile := app.Settings.ServerKey
	httpAddr := ":" + app.Settings.PortHttp
	httpsAddr := ":" + app.Settings.PortHttps
	serverErrCh := make(chan error, 2)

	go func() {
		log.Printf("%sHTTP server running on http://%s:%s", debugTag, app.Settings.Host, app.Settings.PortHttp)
		if err := http.ListenAndServe(httpAddr, loggedHandler); err != nil {
			serverErrCh <- fmt.Errorf("http server error: %w", err)
		}
	}()

	go func() {
		log.Printf("%sHTTPS server running on https://%s:%s", debugTag, app.Settings.Host, app.Settings.PortHttps)
		if err := http.ListenAndServeTLS(httpsAddr, crtFile, keyFile, loggedHandler); err != nil {
			serverErrCh <- fmt.Errorf("https server error: %w", err)
		}
	}()

	log.Fatalf("%v", <-serverErrCh)
}
