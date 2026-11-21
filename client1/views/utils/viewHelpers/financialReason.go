package viewHelpers

import (
	"strings"
	"syscall/js"
)

func FinancialChangeReason() (string, bool) {
	value := js.Global().Call("prompt", "Reason for this financial correction (required):", "")
	if value.IsNull() || value.IsUndefined() {
		return "", false
	}
	reason := strings.TrimSpace(value.String())
	return reason, len(reason) > 0 && len(reason) <= 500
}
