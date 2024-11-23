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
	defer app.Close()

	r := mux.NewRouter()

	// Setup your API subrouter with CORS middleware
	subR := r.PathPrefix(os.Getenv("API_PATH_PREFIX")).Subrouter()

	//SRP authentication and registration process handlers
	auth := handlerAuth.New(app)
	subR.HandleFunc("/auth/register/", auth.AccountCreate).Methods("Post")
	subR.HandleFunc("/auth/{username}/salt/", auth.AuthGetSalt).Methods("Get", "Options")
	subR.HandleFunc("/auth/{username}/key/{A}", auth.AuthGetKeyB).Methods("Get")
	subR.HandleFunc("/auth/proof/", auth.AuthCheckClientProof).Methods("Post")
	subR.HandleFunc("/auth/validate/{token}", auth.AccountValidate).Methods("Get")
	subR.HandleFunc("/auth/reset/{username}/password/", auth.AuthReset).Methods("Get")
	subR.HandleFunc("/auth/reset/{token}/token/", auth.AuthUpdate).Methods("Post")
	subR.HandleFunc("/auth/logout/", auth.AuthLogout).Methods("Post")

	subR.Use(auth.RequireRestAuth) // Add some middleware, e.g. an auth handler

	// Seasons routes
	seasons := handlerSeasons.New(app)
	subR.HandleFunc("/seasons", seasons.GetAll).Methods("GET")
	subR.HandleFunc("/seasons/{id}", seasons.Get).Methods("GET")
	subR.HandleFunc("/seasons", seasons.Create).Methods("POST")
	subR.HandleFunc("/seasons/{id}", seasons.Update).Methods("PUT")
	subR.HandleFunc("/seasons/{id}", seasons.Delete).Methods("DELETE")

	// User routes
	user := handlerUser.New(app)
	subR.HandleFunc("/users", user.GetAll).Methods("GET")
	subR.HandleFunc("/users/{id}", user.Get).Methods("GET")
	subR.HandleFunc("/users", user.Create).Methods("POST")
	subR.HandleFunc("/users/{id}", user.Update).Methods("PUT")
	subR.HandleFunc("/users/{id}", user.Delete).Methods("DELETE")

	// UserCategory routes
	userAgeGroups := handlerUserAgeGroups.New(app)
	subR.HandleFunc("/userAgeGroups", userAgeGroups.GetAll).Methods("GET")
	subR.HandleFunc("/userAgeGroups/{id}", userAgeGroups.Get).Methods("GET")
	subR.HandleFunc("/userAgeGroups", userAgeGroups.Create).Methods("POST")
	subR.HandleFunc("/userAgeGroups/{id}", userAgeGroups.Update).Methods("PUT")
	subR.HandleFunc("/userAgeGroups/{id}", userAgeGroups.Delete).Methods("DELETE")

	// UserPayments routes
	userPayments := handlerUserPayments.New(app)
	subR.HandleFunc("/userPayments", userPayments.GetAll).Methods("GET")
	subR.HandleFunc("/userPayments/{id}", userPayments.Get).Methods("GET")
	subR.HandleFunc("/userPayments", userPayments.Create).Methods("POST")
	subR.HandleFunc("/userPayments/{id}", userPayments.Update).Methods("PUT")
	subR.HandleFunc("/userPayments/{id}", userPayments.Delete).Methods("DELETE")

	// UserStatus routes
	userStatus := handlerUserStatus.New(app)
	subR.HandleFunc("/userStatus", userStatus.GetAll).Methods("GET")
	subR.HandleFunc("/userStatus/{id}", userStatus.Get).Methods("GET")
	subR.HandleFunc("/userStatus", userStatus.Create).Methods("POST")
	subR.HandleFunc("/userStatus/{id}", userStatus.Update).Methods("PUT")
	subR.HandleFunc("/userStatus/{id}", userStatus.Delete).Methods("DELETE")

	// Booking routes
	booking := handlerBooking.New(app)
	subR.HandleFunc("/bookings", booking.GetAll).Methods("GET")
	subR.HandleFunc("/bookings/{id:[0-9]+}", booking.Get).Methods("GET")
	subR.HandleFunc("/bookings", booking.Create).Methods("POST")
	subR.HandleFunc("/bookings/{id:[0-9]+}", booking.Update).Methods("PUT")
	subR.HandleFunc("/bookings/{id:[0-9]+}", booking.Delete).Methods("DELETE")
	subR.HandleFunc("/trips/{id:[0-9]+}/bookings", booking.GetList).Methods("GET")

	// BookingUsers routes
	bookingPeople := handlerBookingPeople.New(app)
	subR.HandleFunc("/bookingPeople", bookingPeople.GetAll).Methods("GET")
	subR.HandleFunc("/bookingPeople/{id:[0-9]+}", bookingPeople.Get).Methods("GET")
	subR.HandleFunc("/bookingPeople", bookingPeople.Create).Methods("POST")
	subR.HandleFunc("/bookingPeople/{id:[0-9]+}", bookingPeople.Update).Methods("PUT")
	subR.HandleFunc("/bookingPeople/{id:[0-9]+}", bookingPeople.Delete).Methods("DELETE")
	subR.HandleFunc("/bookings/{id:[0-9]+}/bookingPeople", bookingPeople.GetList).Methods("GET")

	// GroupBookings routes
	groupBooking := handlerGroupBooking.New(app)
	subR.HandleFunc("/groupBooking", groupBooking.GetAll).Methods("GET")
	subR.HandleFunc("/groupBooking/{id:[0-9]+}", groupBooking.Get).Methods("GET")
	subR.HandleFunc("/groupBooking", groupBooking.Create).Methods("POST")
	subR.HandleFunc("/groupBooking/{id:[0-9]+}", groupBooking.Update).Methods("PUT")
	subR.HandleFunc("/groupBooking/{id:[0-9]+}", groupBooking.Delete).Methods("DELETE")

	// BookingStatus routes
	bookingStatus := handlerBookingStatus.New(app)
	subR.HandleFunc("/bookingStatus", bookingStatus.GetAll).Methods("GET")
	subR.HandleFunc("/bookingStatus/{id:[0-9]+}", bookingStatus.Get).Methods("GET")
	subR.HandleFunc("/bookingStatus", bookingStatus.Create).Methods("POST")
	subR.HandleFunc("/bookingStatus/{id:[0-9]+}", bookingStatus.Update).Methods("PUT")
	subR.HandleFunc("/bookingStatus/{id:[0-9]+}", bookingStatus.Delete).Methods("DELETE")

	// Trip routes
	trip := handlerTrip.New(app)
	subR.HandleFunc("/trips", trip.GetAll).Methods("GET")
	subR.HandleFunc("/trips/{id:[0-9]+}", trip.Get).Methods("GET")
	subR.HandleFunc("/trips", trip.Create).Methods("POST")
	subR.HandleFunc("/trips/{id:[0-9]+}", trip.Update).Methods("PUT")
	subR.HandleFunc("/trips/{id:[0-9]+}", trip.Delete).Methods("DELETE")
	subR.HandleFunc("/trips/participantStatus", trip.GetParticipantStatus).Methods("GET")

	// TripType routes
	tripType := handlerTripType.New(app)
	subR.HandleFunc("/tripType", tripType.GetAll).Methods("GET")
	subR.HandleFunc("/tripType/{id:[0-9]+}", tripType.Get).Methods("GET")
	subR.HandleFunc("/tripType", tripType.Create).Methods("POST")
	subR.HandleFunc("/tripType/{id:[0-9]+}", tripType.Update).Methods("PUT")
	subR.HandleFunc("/tripType/{id:[0-9]+}", tripType.Delete).Methods("DELETE")

	// TripStatus routes
	tripStatus := handlerTripStatus.New(app)
	subR.HandleFunc("/tripStatus", tripStatus.GetAll).Methods("GET")
	subR.HandleFunc("/tripStatus/{id:[0-9]+}", tripStatus.Get).Methods("GET")
	subR.HandleFunc("/tripStatus", tripStatus.Create).Methods("POST")
	subR.HandleFunc("/tripStatus/{id:[0-9]+}", tripStatus.Update).Methods("PUT")
	subR.HandleFunc("/tripStatus/{id:[0-9]+}", tripStatus.Delete).Methods("DELETE")

	// TripDifficulty routes
	tripDifficulty := handlerTripDifficulty.New(app)
	subR.HandleFunc("/tripDifficulty", tripDifficulty.GetAll).Methods("GET")
	subR.HandleFunc("/tripDifficulty/{id:[0-9]+}", tripDifficulty.Get).Methods("GET")
	subR.HandleFunc("/tripDifficulty", tripDifficulty.Create).Methods("POST")
	subR.HandleFunc("/tripDifficulty/{id:[0-9]+}", tripDifficulty.Update).Methods("PUT")
	subR.HandleFunc("/tripDifficulty/{id:[0-9]+}", tripDifficulty.Delete).Methods("DELETE")

	// TripCost routes
	tripCosts := handlerTripCost.New(app)
	subR.HandleFunc("/tripCosts", tripCosts.GetAll).Methods("GET")
	subR.HandleFunc("/tripCosts/{id:[0-9]+}", tripCosts.Get).Methods("GET")
	subR.HandleFunc("/tripCosts", tripCosts.Create).Methods("POST")
	subR.HandleFunc("/tripCosts/{id:[0-9]+}", tripCosts.Update).Methods("PUT")
	subR.HandleFunc("/tripCosts/{id:[0-9]+}", tripCosts.Delete).Methods("DELETE")

	// TripCostGroup routes
	tripCostGroups := handlerTripCostGroup.New(app)
	subR.HandleFunc("/tripCostGroups", tripCostGroups.GetAll).Methods("GET")
	subR.HandleFunc("/tripCostGroups/{id:[0-9]+}", tripCostGroups.Get).Methods("GET")
	subR.HandleFunc("/tripCostGroups", tripCostGroups.Create).Methods("POST")
	subR.HandleFunc("/tripCostGroups/{id:[0-9]+}", tripCostGroups.Update).Methods("PUT")
	subR.HandleFunc("/tripCostGroups/{id:[0-9]+}", tripCostGroups.Delete).Methods("DELETE")

	// Static handlers
	r.PathPrefix("/client/").Handler(http.StripPrefix("/client/", http.FileServer(http.Dir("./static"))))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(".")))

	// For debugging: Log all registered routes
	//r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
	//	path, _ := route.GetPathTemplate()
	//	methods, _ := route.GetMethods()
	//	log.Printf("Registered route: %s %v", path, methods)
	//	return nil
	//})

	// Define CORS options
	corsOpts := handlers.CORS(
		handlers.AllowedOrigins([]string{"http://localhost:8081"}),                                             // Allow requests from http://localhost:8080 //w.Header().Set("Access-Control-Allow-Origin", "http://localhost") // "http://localhost:8081" // or "*" if you want to test without restrictions
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),                           // Allowed HTTP methods
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization", "Access-Control-Allow-Credentials"}), // Allowed headers
		handlers.AllowCredentials(), // Headers([]string{"Content-Type"}) //w.Header().Set("Access-Control-Allow-Credentials", "true")
	)

	corsMuxHandler := helpers.LogRequest(corsOpts(r))

	// Paths to certificate and key files
	crtFile := "/etc/ssl/certs/localhost.crt" // "../certs/api-server/cert.pem"
	keyFile := "/etc/ssl/certs/localhost.key" // "../certs/api-server/key.pem"

	go func() {
		log.Println(debugTag + "HTTP server running on http://localhost:8085")
		if err := http.ListenAndServe(":8085", corsMuxHandler); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	go func() {
		log.Println(debugTag + "HTTPS server running on https://localhost:8086")
		if err := http.ListenAndServeTLS(":8086", crtFile, keyFile, corsMuxHandler); err != nil {
			log.Fatalf("HTTPS server error: %v", err)
		}
	}()

	// Block the main goroutine to keep the servers running
	select {}

	/*
		//******************************************************************
		// Config and Start HTTP
		//******************************************************************

		server := &http.Server{
			//Addr:         ":" + *portHttp,
			Addr: ":" + app.Settings.PortHttp,

			ReadTimeout:  5 * time.Minute, // 5 min to allow for delays when 'curl' on OSx prompts for username/password
			WriteTimeout: 10 * time.Second,
			//TLSConfig:    &tls.Config{ServerName: *host},
			//Handler: (main.LogRequest(app.Mux)),
			Handler: (corsMuxHandler),
		}

		go func() error {
			if err := server.ListenAndServe(); err != nil {
				//log.Fatal(err)
				log.Fatal("Web server (HTTP): ", err)
				return fmt.Errorf("Server failed to start: %w", err)
			}
			return nil
			//err_http := http.ListenAndServe(":8085", main.LogRequest(app.Mux))
			//if err_http != nil {
			//	log.Fatal("Web server (HTTP): ", err_http)
			//}
		}()

		//******************************************************************
		// Config and Start HTTPS
		//******************************************************************

		serverTLS := &http.Server{
			//Addr:         ":" + *portHttps,
			Addr:         ":" + app.Settings.PortHttps,
			ReadTimeout:  5 * time.Minute, // 5 min to allow for delays when 'curl' on OSx prompts for username/password
			WriteTimeout: 10 * time.Second,
			//TLSConfig: &tls.Config{
			//	ServerName: *host,
			//	ClientAuth: tls.ClientAuthType(*certOpt),
			//	MinVersion: tls.VersionTLS12, // TLS versions below 1.2 are considered insecure - see https://www.rfc-editor.org/rfc/rfc7525.txt for details
			//},
			//TLSConfig: getTLSConfig(*host, *clientCaCert, *serverCaCert, tls.ClientAuthType(*certOpt)),
			TLSConfig: &tls.Config{
				ServerName: host,
				ClientAuth: certOpt,
				ClientCAs:  caCertPool,
				MinVersion: tls.VersionTLS12, // TLS versions below 1.2 are considered insecure - see https://www.rfc-editor.org/rfc/rfc7525.txt for details
			},
			//getTLSConfig(app.Settings.Host,
			//	app.Settings.ClientCaCert,
			//	app.Settings.ServerCaCert,
			//	tls.ClientAuthType(app.Settings.CertOpt)),
			Handler: corsMuxHandler,
		}

		if err := serverTLS.ListenAndServeTLS(app.Settings.ServerCert, app.Settings.ServerKey); err != nil {
			log.Fatal(debugTag+"Web server (HTTPS): ", err)
		}
	*/
}

/*
func XXCorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8081")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// If it's an OPTIONS request, just return without passing to next handler
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
*/
