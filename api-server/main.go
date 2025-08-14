package main

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/handlerAccessLevel"
	"api-server/v2/localHandlers/handlerAccessType"
	"api-server/v2/localHandlers/handlerBooking"
	"api-server/v2/localHandlers/handlerBookingPeople"
	"api-server/v2/localHandlers/handlerBookingStatus"
	"api-server/v2/localHandlers/handlerGroupBooking"
	"api-server/v2/localHandlers/handlerMemberStatus"
	"api-server/v2/localHandlers/handlerMyBookings"
	"api-server/v2/localHandlers/handlerResource"
	"api-server/v2/localHandlers/handlerSRPAuth"
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
	"api-server/v2/localHandlers/handlerWebAuthn"
	"api-server/v2/localHandlers/handlerWebAuthnManagement"
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
	app.Settings.LoadEnv()
	defer app.Close()
	log.Printf("%smain() App settings: %+v, os Env: %+v\n", debugTag, app.Settings, os.Environ())
	r := mux.NewRouter()

	// Setup your API subrouter with CORS middleware
	subR1 := r.PathPrefix(os.Getenv("API_PATH_PREFIX")).Subrouter()

	//SRP authentication and registration process handlers
	SRPauth := handlerSRPAuth.New(app)
	subR1.HandleFunc("/auth/register/", SRPauth.AccountCreate).Methods("Post")
	subR1.HandleFunc("/auth/{username}/salt/", SRPauth.AuthGetSalt).Methods("Get", "Options")
	subR1.HandleFunc("/auth/{username}/key/{A}", SRPauth.AuthGetKeyB).Methods("Get")
	subR1.HandleFunc("/auth/proof/", SRPauth.AuthCheckClientProof).Methods("Post")
	subR1.HandleFunc("/auth/validate/{token}", SRPauth.AccountValidate).Methods("Get")
	subR1.HandleFunc("/auth/reset/{username}/password/", SRPauth.AuthReset).Methods("Get")
	subR1.HandleFunc("/auth/reset/{token}/token/", SRPauth.AuthUpdate).Methods("Post")
	//subR1.HandleFunc("/auth/sessioncheck/", auth.SessionCheck).Methods("Get")

	// WebAuthn handlers
	WebAuthn := handlerWebAuthn.New(app)
	subR1.HandleFunc("/webauthn/register/begin/", WebAuthn.BeginRegistration).Methods("POST")
	subR1.HandleFunc("/webauthn/register/finish/", WebAuthn.FinishRegistration).Methods("POST")
	subR1.HandleFunc("/webauthn/login/begin/{username}", WebAuthn.BeginLogin).Methods("POST")
	subR1.HandleFunc("/webauthn/login/finish/", WebAuthn.FinishLogin).Methods("POST")
	subR1.HandleFunc("/webauthn/emailReset/begin/", WebAuthn.BeginEmailResetHandler).Methods("POST")
	subR1.HandleFunc("/webauthn/emailReset/finish/{token}", WebAuthn.FinishEmailResetHandler).Methods("Get")

	subR2 := r.PathPrefix(os.Getenv("API_PATH_PREFIX")).Subrouter()
	subR2.Use(SRPauth.RequireRestAuth) // Add some middleware, e.g. an auth handler

	subR2.HandleFunc("/auth/logout/", SRPauth.AuthLogout).Methods("Get")
	subR2.HandleFunc("/auth/menuUser/", SRPauth.MenuUserGet).Methods("Get")
	subR2.HandleFunc("/auth/menuList/", SRPauth.MenuListGet).Methods("Get")

	// Add route groups
	addRouteGroup(subR2, "webauthn", handlerWebAuthnManagement.New(app))                 // WebAuthn routes
	addRouteGroup(subR2, "seasons", handlerSeasons.New(app))                             // Seasons routes
	addRouteGroup(subR2, "users", handlerUser.New(app))                                  // User routes
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

	trip := handlerTrip.New(app)                                               // Trip routes
	addRouteGroup(subR2, "trips", trip)                                        // Trip routes
	subR2.HandleFunc("/tripsReport", trip.GetParticipantStatus).Methods("GET") // Trip routes

	// For debugging: Log all registered routes
	//subR2.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
	//	path, _ := route.GetPathTemplate()
	//	methods, _ := route.GetMethods()
	//	log.Printf("Registered routes for subR2: %s %v", path, methods)
	//	return nil
	//})

	// Static handlers
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("./static")))) // Serve static files from the "/static" directory under the url "/"

	// Define CORS options
	corsOpts := handlers.CORS(
		handlers.AllowedOrigins([]string{"http://localhost:8081", "https://localhost:8081", "http://localhost:8085", "https://localhost:8086"}), // Allow requests from http://localhost:8080 //w.Header().Set("Access-Control-Allow-Origin", "http://localhost") // "http://localhost:8081" // or "*" if you want to test without restrictions
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),                                                            // Allowed HTTP methods
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization", "Access-Control-Allow-Credentials"}),                                  // Allowed headers
		handlers.AllowCredentials(), // Headers([]string{"Content-Type"}) //w.Header().Set("Access-Control-Allow-Credentials", "true")
	)

	corsMuxHandler := corsOpts(r)
	loggedHandler := helpers.LogRequest(corsMuxHandler)
	//corsMuxHandler := corsOpts(r)

	// Paths to certificate and key files
	crtFile := "/etc/ssl/certs/localhost.crt" // "../certs/api-server/cert.pem"
	keyFile := "/etc/ssl/certs/localhost.key" // "../certs/api-server/key.pem"

	// For debugging: Log all registered routes
	//r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
	//	path, _ := route.GetPathTemplate()
	//	methods, _ := route.GetMethods()
	//	log.Printf("Registered route: %s %v", path, methods)
	//	return nil
	//})

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

	/* Seup helpers to start the servers with basic and TLS configuration.
	appCore.StartServerHTTP(":8086", loggedHandler, debugTag)

	// ...set up caCertPool, etc...
	tlsConfig := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  caCertPool,
		MinVersion: tls.VersionTLS12,
	}
	appCore.StartServerHTTPS(":8085", crtFile, keyFile, loggedHandler, debugTag, tlsConfig)
	*/

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
