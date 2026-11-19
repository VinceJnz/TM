package handlerAuth

import (
	"api-server/v2/localHandlers/helpers"
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCapabilityComesFromRouteDeclaration(t *testing.T) {
	for _, path := range []string{"/custom/v9/trips/12/bookings", "/custom/v9/bookings/checkout/create/37", "/anything/with/many/segments"} {
		router := mux.NewRouter()
		helpers.CapabilityRoute(router, path, "bookings", func(w http.ResponseWriter, r *http.Request) {
			resource, err := (&Handler{}).setRestResource(r)
			if err != nil || resource.ResourceName != "bookings" || resource.AccessMethod != "GET" {
				t.Fatalf("wrong capability: %+v %v", resource, err)
			}
		}).Methods("GET")
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", path, nil))
	}
}
func TestUndeclaredCapabilityFailsClosed(t *testing.T) {
	if _, err := (&Handler{}).setRestResource(httptest.NewRequest("GET", "/api/v1/bookings", nil)); err == nil {
		t.Fatal("inferred capability from URL")
	}
}
