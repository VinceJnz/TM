// Personal booking routes share the booking service and resource store.
package handlerMyBookings

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/handlerBooking"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/trips"
	"net/http"

	"github.com/gorilla/mux"
)

type Handler struct {
	*handlerBooking.Handler
	trips trips.Store
}

func New(app *appCore.Config) *Handler {
	return &Handler{Handler: handlerBooking.New(app, true), trips: trips.Store{DB: app.Db}}
}
func (h *Handler) RegisterRoutes(r *mux.Router, base string) {
	helpers.AddRouteGroup(r, base, h, "myBookings")
	helpers.CapabilityRoute(r, base+"/trips", "myBookings", h.bookingTrips).Methods("GET")
}

func (h *Handler) bookingTrips(w http.ResponseWriter, r *http.Request) {
	page, ok := helpers.ReadResourcePage(w, r)
	if !ok {
		return
	}
	items, info, err := h.trips.BookingChoices(r.Context(), page)
	if err != nil {
		helpers.WriteDataError(w, err)
		return
	}
	helpers.WriteResourcePage(w, items, info)
}
