package helpers

import (
	"net/http"

	"github.com/gorilla/mux"
)

type genericHandler interface {
	GetAll(w http.ResponseWriter, r *http.Request)
	Get(w http.ResponseWriter, r *http.Request)
	Create(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	Delete(w http.ResponseWriter, r *http.Request)
}

func AddRouteGroup(r *mux.Router, resourcePath string, handler genericHandler) {
	r.HandleFunc(resourcePath, handler.GetAll).Methods("GET")
	r.HandleFunc(resourcePath+"/{id:[0-9]+}", handler.Get).Methods("GET")
	r.HandleFunc(resourcePath, handler.Create).Methods("POST")
	r.HandleFunc(resourcePath+"/{id:[0-9]+}", handler.Update).Methods("PUT")
	r.HandleFunc(resourcePath+"/{id:[0-9]+}", handler.Delete).Methods("DELETE")
	// Add some code to register the route resource for managing security access
}
