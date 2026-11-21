// Personal booking routes share the booking service and resource store.
package handlerMyBookings

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/handlerBooking"
	"api-server/v2/localHandlers/helpers"
	"github.com/gorilla/mux"
)

type Handler struct{ *handlerBooking.Handler }

func New(app *appCore.Config) *Handler { return &Handler{handlerBooking.New(app, true)} }
func (h *Handler) RegisterRoutes(r *mux.Router, base string) {
	helpers.AddRouteGroup(r, base, h, "myBookings")
}
