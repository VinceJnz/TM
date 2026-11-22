package main

import (
	"api-server/v2/app/appCore"
	"api-server/v2/app/httpserver"
	"api-server/v2/localHandlers/handlerAccessLevel"
	"api-server/v2/localHandlers/handlerAccessScope"
	"api-server/v2/localHandlers/handlerAuth"
	"api-server/v2/localHandlers/handlerBooking"
	"api-server/v2/localHandlers/handlerBookingPeople"
	"api-server/v2/localHandlers/handlerBookingStatus"
	"api-server/v2/localHandlers/handlerBookingVoucher"
	"api-server/v2/localHandlers/handlerGmailCert"
	"api-server/v2/localHandlers/handlerGroupBooking"
	"api-server/v2/localHandlers/handlerMemberStatus"
	"api-server/v2/localHandlers/handlerMyBookings"
	"api-server/v2/localHandlers/handlerMyProfile"
	"api-server/v2/localHandlers/handlerOAuth"
	"api-server/v2/localHandlers/handlerOperations"
	"api-server/v2/localHandlers/handlerPaymentRefunds"
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
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
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

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	app := appCore.New(false)
	defer app.Close()
	if err := app.Run(ctx); err != nil {
		return err
	}
	r := buildRouter(app)
	// Define CORS options
	allowedOrigins := parseOrigins(app.Settings.CoreOrigins)
	log.Printf("%sCORS allowed origins: %v", debugTag, allowedOrigins)
	corsOpts := handlers.CORS(
		handlers.AllowedOrigins(allowedOrigins),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization", "Access-Control-Allow-Credentials", "X-Change-Reason"}),
		handlers.ExposedHeaders([]string{"X-Next-Offset", "X-Page-Limit", "X-Pagination-Mode", "X-Next-Cursor"}),
		handlers.AllowCredentials(), // Headers([]string{"Content-Type"}) //w.Header().Set("Access-Control-Allow-Credentials", "true")
	)

	proxy, err := helpers.TrustedProxy(os.Getenv("TRUSTED_PROXY_CIDRS"))
	if err != nil {
		return err
	}
	corsMuxHandler := proxy(corsOpts(r))
	loggedHandler := corsMuxHandler

	origin := "https://" + net.JoinHostPort(app.Settings.Host, app.Settings.PortHttps)
	servers := httpserver.New(loggedHandler, ":"+app.Settings.PortHttp, ":"+app.Settings.PortHttps, origin)
	return servers.Serve(ctx, app.Settings.ServerCert, app.Settings.ServerKey)
}
func buildRouter(app *appCore.Config) *mux.Router {
	r := mux.NewRouter()
	r.Use(helpers.RequestTrace, helpers.RequestDeadline(30*time.Second))
	r.HandleFunc("/livez", app.Liveness).Methods("GET")
	r.HandleFunc("/readyz", app.Readiness).Methods("GET")
	r.HandleFunc("/client-config.js", httpserver.ClientConfig(app.Settings.APIprefix)).Methods("GET")
	// *****************************************************
	// Setup your API subrouter with CORS middleware. These routes are unprotected, i.e. do not require authentication to use. This is where you would register any routes that should be accessible without authentication, such as login or registration endpoints.
	public := r.PathPrefix(app.Settings.APIprefix).Name("public-api").Subrouter()
	public.Use(helpers.AuthRateLimit(app.Db))
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

	// UserAgeGroups public read-only routes (for registration form dropdown)
	handlerUserAgeGroups.New(app).RegisterRoutesPublic(public, "/userAgeGroups")

	sessionRoutes := r.PathPrefix(app.Settings.APIprefix).Name("session-api").Subrouter()
	sessionRoutes.Use(auth.RequireSessionAuth)
	auth.RegisterRoutesSession(sessionRoutes, "/auth")

	// *****************************************************
	// Protected routes (require authentication) - These are registered on a subrouter so that the auth middleware is only applied to these routes and not the public routes above. This allows us to have a mix of protected and unprotected routes.
	protected := r.PathPrefix(app.Settings.APIprefix).Name("protected-api").Subrouter()
	protected.Use(auth.RequireSessionAuth, auth.RequireCapability) // This needs to be here to protect the routes below. It checks for a valid session cookie and ensures the user has access to the requested resource. If the session is valid and access is granted, it allows the request to proceed to the appropriate handler. If not, it returns an unauthorized error.
	protected.Use(func(next http.Handler) http.Handler {
		return helpers.LogRequest(next, app.SessionIDKey, "protected") // Then logging
	})
	helpers.CapabilityRoute(protected, "/operations", "operations", handlerOperations.Read(app)).Methods("GET")

	// Auth handlers - Protected routes (requires authentication)
	auth.RegisterRoutesProtected(protected, "/auth") // Protected auth endpoints (menuUser, logout, etc.)

	// The following routes are protected, i.e. require authentication to use.
	oauth.RegisterRoutesProtected(protected, "/auth/oauth") // Protected OAuth handlers

	// Add route groups (protected - user and above)
	handlerMyBookings.New(app).RegisterRoutes(protected, "/myBookings")       // MyBookings routes
	handlerMyProfile.New(app).RegisterRoutes(protected, "/myProfile")         // MyProfile routes
	handlerBooking.New(app).RegisterRoutes(protected, "/bookings", "/trips")  // Booking routes
	handlerBookingPeople.New(app).RegisterRoutes(protected, "/bookingPeople") // BookingPeople routes
	handlerBookingVoucher.New(app).RegisterRoutesProtected(protected, "/bookingVouchers")
	handlerTrip.New(app).RegisterRoutes(protected, "/trips") // Trip routes

	// Add route groups previously role-gated; capability checks are enforced by RequireSessionAuth.
	handlerSeasons.New(app).RegisterRoutes(protected, "/seasons") // Seasons routes
	users := handlerUser.New(app)
	users.RegisterRoutes(protected, "/users")
	users.RegisterBookingPeopleLookup(protected)
	handlerUserAgeGroups.New(app).RegisterRoutes(protected, "/userAgeGroups")         // UserAgeGroup routes
	handlerUserPayments.New(app).RegisterRoutes(protected, "/payments")               // Payments routes
	handlerPaymentRefunds.New(app).RegisterRoutes(protected, "/refunds")              // Refunds routes
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
	handlerGmailCert.New(app).RegisterRoutes(protected, "/gmailcert")                         // Gmail cert management (Sys Admin only)

	// Static handlers
	r.PathPrefix("/").Handler(httpserver.Static("./static"))

	return r
}
