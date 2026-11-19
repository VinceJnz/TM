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

func AddRouteGroup(r *mux.Router, resourcePath string, handler genericHandler, resource string) {
	CapabilityRoute(r, resourcePath, resource, handler.GetAll).Methods("GET")
	CapabilityRoute(r, resourcePath+"/{id:[0-9]+}", resource, handler.Get).Methods("GET")
	CapabilityRoute(r, resourcePath, resource, handler.Create).Methods("POST")
	CapabilityRoute(r, resourcePath+"/{id:[0-9]+}", resource, handler.Update).Methods("PUT")
	CapabilityRoute(r, resourcePath+"/{id:[0-9]+}", resource, handler.Delete).Methods("DELETE")
	// Add some code to register the route resource for managing security access
}

// A route's capability is declared alongside its path and methods. Request URL
// layout and API version never determine authorization.
func CapabilityRoute(r *mux.Router, path, resource string, handler http.HandlerFunc) *mux.Route {
	return r.HandleFunc(path, handler).Name("capability:" + resource)
}
