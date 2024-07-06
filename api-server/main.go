package main

import (
	"api-server/v2/app"
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

	handler := handlerUser.New(db)
	r.HandleFunc("/users", handler.GetAll).Methods("GET")
	r.HandleFunc("/users/{id}", handler.Get).Methods("GET")
	r.HandleFunc("/users", handler.Create).Methods("POST")
	r.HandleFunc("/users/{id}", handler.Update).Methods("PUT")
	r.HandleFunc("/users/{id}", handler.Delete).Methods("DELETE")

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
