package accessTypeView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/views/accessScopeView"
	"syscall/js"
)

// Deprecated: accessTypeView is a legacy compatibility shim. Prefer accessScopeView.
const ApiURL = accessScopeView.ApiURL

type ItemEditor = accessScopeView.ItemEditor

func New(document js.Value, eventProcessor *eventProcessor.EventProcessor, appCore *appCore.AppCore, idList ...int) *ItemEditor {
	return accessScopeView.New(document, eventProcessor, appCore, idList...)
}
