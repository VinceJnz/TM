package handlerSRPAuth

import (
	"api-server/v2/app/appCore"
	"api-server/v2/app/pools/srpPool"
	"net/http"

	"github.com/gorilla/mux"
)

const debugTag = "handlerSRPAuth."

type HandlerFunc func(http.ResponseWriter, *http.Request)

type Handler struct {
	appConf *appCore.Config
	Pool    *srpPool.Pool
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{
		appConf: appConf,
		Pool:    srpPool.New(),
	}
}

// RegisterRoutes registers handler routes on the provided router.
func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	r.HandleFunc(baseURL+"/register/", h.AccountCreate).Methods("Post")
	r.HandleFunc(baseURL+"/{username}/salt/", h.AuthGetSalt).Methods("Get", "Options")
	r.HandleFunc(baseURL+"/{username}/key/{A}", h.AuthGetKeyB).Methods("Get")
	r.HandleFunc(baseURL+"/proof/", h.AuthCheckClientProof).Methods("Post")
	r.HandleFunc(baseURL+"/validate/{token}", h.AccountValidate).Methods("Get")
	r.HandleFunc(baseURL+"/reset/{username}/password/", h.AuthReset).Methods("Get")
	r.HandleFunc(baseURL+"/reset/{token}/token/", h.AuthUpdate).Methods("Post")
	//r.HandleFunc("/srpAuth/sessioncheck/", auth.SessionCheck).Methods("Get")
}
