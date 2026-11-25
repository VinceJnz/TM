package viewHelpers

import (
	"client1/v2/app/domEvents"
	"client1/v2/app/httpProcessor"
	"syscall/js"
)

// LoadRegistrationAgeGroups keeps the form available when the required lookup fails.
func LoadRegistrationAgeGroups(document, selectElement js.Value, client *httpProcessor.Client) (func(), func()) {
	status := document.Call("createElement", "p")
	status.Call("setAttribute", "role", "status")
	parent := selectElement.Get("parentElement")
	parent.Call("appendChild", status)
	var load func()
	retry := Button2(func() any { load(); return nil }, document, "Retry loading age groups", selectElement.Get("id").String()+"-retry")
	parent.Call("appendChild", retry)
	var scope httpProcessor.LoadScope

	load = func() {
		if selectElement.Get("options").Get("length").Int() > 1 {
			return
		}
		ticket := scope.Start()
		retry.Set("hidden", true)
		selectElement.Set("disabled", true)
		status.Set("textContent", "Loading age groups…")
		go func() {
			var groups []struct {
				ID   int    `json:"id"`
				Name string `json:"age_group"`
			}
			err := client.RequestContext(ticket.Context, "GET", "/userAgeGroups", &groups, nil)
			if !ticket.Finish(err) || !parent.Get("isConnected").Bool() {
				return
			}
			if err != nil || len(groups) == 0 {
				status.Set("textContent", "Age groups are unavailable. Try loading them again.")
				retry.Set("hidden", false)
				return
			}
			for _, group := range groups {
				option := document.Call("createElement", "option")
				option.Set("value", group.ID)
				option.Set("textContent", group.Name)
				selectElement.Call("appendChild", option)
			}
			selectElement.Set("disabled", false)
			status.Set("textContent", "")
		}()
	}
	cancelStart := domEvents.Timeout(func(js.Value, []js.Value) any {
		if parent.Get("isConnected").Bool() {
			load()
		}
		return nil
	}, 0)
	return load, func() {
		cancelStart()
		scope.Cancel()
	}
}
