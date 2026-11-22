package handlerAuth

import (
	authService "api-server/v2/app/services/authentication"
	authStore "api-server/v2/modelMethods/authentication"
	"api-server/v2/modelMethods/dbStandardTemplate"
	"api-server/v2/models"
	"log"
	"net/http"
)

// MenuUserGet is a public endpoint that returns menu user data if authenticated, or empty data if not
// This allows the client to check authentication status on page load without getting errors
func (h *Handler) MenuUserGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Try to find session cookie
	sc, err := r.Cookie("session")
	if err != nil {
		// No session cookie - return empty response
		writeJSONResponse(w, http.StatusOK, models.MenuUser{}, "MenuUserGet()")
		return
	}

	auth := authService.Service{Store: authStore.Store{DB: h.appConf.Db}}
	user, err := auth.Session(r.Context(), sc.Value)
	if err != nil {
		// User not found or not active
		writeJSONResponse(w, http.StatusOK, models.MenuUser{}, "MenuUserGet()")
		return
	}

	// User is authenticated and active - return menuUser info
	menuUser, err := auth.Store.MenuUser(r.Context(), user.ID)
	if err != nil {
		// Failed to get menu user.
		writeJSONResponse(w, http.StatusOK, models.MenuUser{}, "MenuUserGet()")
		return
	}

	// Return menu user info
	writeJSONResponse(w, http.StatusOK, menuUser, "MenuUserGet()")
}

// Get: retrieves and returns a single record identified by id
func (h *Handler) MenuListGet(w http.ResponseWriter, r *http.Request) {
	session := dbStandardTemplate.GetSession(w, r, h.appConf.SessionIDKey)
	if session == nil || session.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	items, err := (authStore.Store{DB: h.appConf.Db}).MenuCapabilities(r.Context(), session.UserID)
	if err != nil {
		log.Print("menu capability lookup failed")
		writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal_error", "message": "Internal server error"}, "MenuListGet()")
		return
	}
	writeJSONResponse(w, http.StatusOK, supportedMenuItems(items), "MenuListGet()")
}

func supportedMenuItems(items []models.MenuItem) []models.MenuItem {
	supported := make([]models.MenuItem, 0, len(items))
	for _, item := range items {
		if supportsScope(item.Name, item.AccessLevel, item.AccessScope) {
			supported = append(supported, item)
		}
	}
	return supported
}

// AuthStatus checks if the user is authenticated and returns their info, or returns not_authenticated status
// This endpoint does not require authentication and won't return 401
func (h *Handler) AuthStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Try to find session cookie
	sc, err := r.Cookie("session")
	if err != nil {
		// No session cookie - user not authenticated
		writeJSONResponse(w, http.StatusOK, map[string]any{
			"authenticated": false,
			"user":          nil,
		}, "AuthStatus()")
		return
	}

	auth := authService.Service{Store: authStore.Store{DB: h.appConf.Db}}
	user, err := auth.Session(r.Context(), sc.Value)
	if err != nil {
		// User not found or not active
		writeJSONResponse(w, http.StatusOK, map[string]any{
			"authenticated": false,
			"user":          nil,
		}, "AuthStatus()")
		return
	}

	// User is authenticated - return menuUser info
	menuUser, err := auth.Store.MenuUser(r.Context(), user.ID)
	if err != nil {
		// Failed to get menu user (maybe no group assigned)
		writeJSONResponse(w, http.StatusOK, map[string]any{
			"authenticated": true,
			"user": map[string]any{
				"user_id": user.ID,
				"name":    user.Name,
				"email":   user.Email.String,
				"group":   "",
			},
		}, "AuthStatus()")
		return
	}

	// Return full menu user info
	writeJSONResponse(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"user":          menuUser,
	}, "AuthStatus()")
}
