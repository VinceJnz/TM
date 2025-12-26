package handlerSRPAuth

import (
	"api-server/v2/app/appCore"
	"api-server/v2/app/pools/srpPool"
	"net/http"
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
