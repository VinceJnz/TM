package handlerOperations

import (
	"api-server/v2/app/appCore"
	service "api-server/v2/app/services/operations"
	"api-server/v2/localHandlers/helpers"
	store "api-server/v2/modelMethods/operations"
	"context"
	"net/http"
	"strings"
	"time"
)

func Read(app *appCore.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := helpers.RequiredSession(w, r, app.SessionIDKey)
		if session == nil {
			return
		}
		if !strings.EqualFold(session.AccessScope, "any") {
			helpers.WriteError(w, 403, "forbidden", "Operations requires any scope")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		status, err := (service.Service{Store: store.Store{DB: app.Db}}).Status(ctx)
		if err != nil {
			helpers.WriteDataError(w, err)
			return
		}
		status.NotificationDeliveryAvailable = app.EmailService() != nil
		helpers.WriteJSON(w, 200, status)
	}
}
