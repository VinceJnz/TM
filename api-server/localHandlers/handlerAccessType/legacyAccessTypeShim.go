package handlerAccessType

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/handlerAccessScope"
)

type Handler = handlerAccessScope.Handler

func New(appConf *appCore.Config) *Handler {
	return handlerAccessScope.New(appConf)
}
