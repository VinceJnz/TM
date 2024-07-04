package main

import (
	"api-server/v2/localHandlers"
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/users", localHandlers.GetUsers).Methods("GET")
	r.HandleFunc("/users/{id}", localHandlers.GetUser).Methods("GET")
	r.HandleFunc("/users", localHandlers.CreateUser).Methods("POST")
	r.HandleFunc("/users/{id}", localHandlers.UpdateUser).Methods("PUT")
	r.HandleFunc("/users/{id}", localHandlers.DeleteUser).Methods("DELETE")

	// Define CORS options
	corsOpts := handlers.CORS(
		handlers.AllowedOrigins([]string{"http://localhost:8080"}),        // Allow requests from http://localhost:8080
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE"}), // Allowed HTTP methods
		handlers.AllowedHeaders([]string{"Content-Type"}),                 // Allowed headers
	)

	log.Println("Server running on port 8085")
	log.Fatal(http.ListenAndServe(":8085", corsOpts(r))) // Apply CORS middleware

	//log.Println("Server running on port 8085")
	//log.Fatal(http.ListenAndServe(":8085", r))
}
