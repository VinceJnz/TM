package userView

import (
	"client1/v2/app/domEvents"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"syscall/js"
	"unicode/utf8"
)

type PersonMatch struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Username   string `json:"username"`
	MemberCode string `json:"member_code"`
	Membership string `json:"membership"`
}

func (person PersonMatch) label() string {
	if person.Username != "" {
		return person.Name + " (" + person.Username + ")"
	}
	return person.Name
}

// NewPersonAutocomplete returns the field, its selected ID, and a function
// that closes the suggestions and cancels pending work.
func (editor *ItemEditor) NewPersonAutocomplete(id int, name, label, htmlID, lookupURL string, onSelection func(PersonMatch)) (js.Value, js.Value, func()) {
	doc := editor.document
	field := doc.Call("createElement", "fieldset")
	field.Set("className", "input-group")
	input := doc.Call("createElement", "input")
	input.Set("type", "text")
	input.Set("id", htmlID)
	input.Set("value", name)
	input.Set("required", true)
	input.Set("autocomplete", "off")
	input.Call("setAttribute", "role", "combobox")
	input.Call("setAttribute", "aria-autocomplete", "list")
	list := doc.Call("createElement", "div")
	list.Set("id", htmlID+"-matches")
	list.Set("className", "person-autocomplete-matches")
	list.Call("setAttribute", "role", "listbox")
	list.Call("setAttribute", "aria-label", label+" matches")
	input.Call("setAttribute", "aria-controls", list.Get("id"))
	selectedID := doc.Call("createElement", "input")
	selectedID.Set("type", "hidden")
	status := doc.Call("createElement", "div")
	status.Call("setAttribute", "role", "status")
	field.Call("append", viewHelpers.Label(doc, label, htmlID), input, list, selectedID, status)

	const selectMessage = "Select a person from the suggestions."
	var loads httpProcessor.LoadScope
	var cancelDelay func()
	var matches []PersonMatch
	active := -1
	closeList := func() {
		if cancelDelay != nil {
			cancelDelay()
			cancelDelay = nil
		}
		loads.Cancel()
		matches = nil
		active = -1
		list.Set("textContent", "")
		list.Set("hidden", true)
		input.Call("setAttribute", "aria-expanded", "false")
		input.Call("removeAttribute", "aria-activedescendant")
		status.Set("textContent", "")
	}
	choose := func(person PersonMatch) {
		input.Set("value", person.label())
		selectedID.Set("value", strconv.Itoa(person.ID))
		input.Call("setCustomValidity", "")
		closeList()
		onSelection(person)
	}
	load := func(query string, prefill bool) {
		ticket := loads.Start()
		go func() {
			var people []PersonMatch
			err := editor.client.RequestContext(ticket.Context, http.MethodGet, lookupURL+query, &people, nil)
			if !ticket.Finish(err) || !field.Get("isConnected").Bool() {
				return
			}
			switch {
			case err != nil:
				status.Set("textContent", "Person lookup failed. Type to search again.")
			case len(people) == 0:
				status.Set("textContent", "No matching people.")
			case prefill:
				choose(people[0])
			default:
				matches = people
				for index, person := range people {
					option := doc.Call("createElement", "div")
					option.Set("id", htmlID+"-match-"+strconv.Itoa(index))
					option.Set("textContent", person.label())
					option.Call("setAttribute", "role", "option")
					option.Call("setAttribute", "aria-selected", "false")
					list.Call("appendChild", option)
				}
				list.Set("hidden", false)
				input.Call("setAttribute", "aria-expanded", "true")
				status.Set("textContent", strconv.Itoa(len(people))+" matches")
			}
		}()
	}
	closeList()
	if id > 0 && name != "" {
		selectedID.Set("value", strconv.Itoa(id))
	} else {
		input.Call("setCustomValidity", selectMessage)
	}
	domEvents.Listen(input, "input", func(js.Value, []js.Value) any {
		closeList()
		selectedID.Set("value", "")
		onSelection(PersonMatch{})
		input.Call("setCustomValidity", selectMessage)
		query := strings.TrimSpace(input.Get("value").String())
		if utf8.RuneCountInString(query) < 2 {
			status.Set("textContent", "Enter at least 2 characters.")
			return nil
		}
		cancelDelay = domEvents.Timeout(func(js.Value, []js.Value) any {
			if field.Get("isConnected").Bool() {
				load("?q="+url.QueryEscape(query), false)
			}
			return nil
		}, 250)
		return nil
	})
	domEvents.Listen(input, "keydown", func(_ js.Value, args []js.Value) any {
		event := args[0]
		key := event.Get("key").String()
		switch key {
		case "Escape":
			event.Call("preventDefault")
			closeList()
		case "ArrowDown", "ArrowUp":
			if len(matches) == 0 {
				return nil
			}
			event.Call("preventDefault")
			options := list.Get("children")
			if active >= 0 {
				options.Index(active).Call("setAttribute", "aria-selected", "false")
			}
			if key == "ArrowDown" {
				active = (active + 1) % len(matches)
			} else if active <= 0 {
				active = len(matches) - 1
			} else {
				active--
			}
			option := options.Index(active)
			option.Call("setAttribute", "aria-selected", "true")
			input.Call("setAttribute", "aria-activedescendant", option.Get("id"))
			option.Call("scrollIntoView", js.ValueOf(map[string]any{"block": "nearest"}))
		case "Enter":
			if active >= 0 {
				event.Call("preventDefault")
				choose(matches[active])
			}
		}
		return nil
	})
	domEvents.Listen(list, "mousedown", func(_ js.Value, args []js.Value) any {
		args[0].Call("preventDefault")
		return nil
	})
	domEvents.Listen(list, "click", func(_ js.Value, args []js.Value) any {
		target := args[0].Get("target")
		options := list.Get("children")
		for index, person := range matches {
			if target.Equal(options.Index(index)) {
				choose(person)
				input.Call("focus")
				break
			}
		}
		return nil
	})
	domEvents.Listen(input, "blur", func(js.Value, []js.Value) any {
		closeList()
		return nil
	})
	if id > 0 {
		cancelDelay = domEvents.Timeout(func(js.Value, []js.Value) any {
			if field.Get("isConnected").Bool() {
				load("?id="+strconv.Itoa(id), true)
			}
			return nil
		}, 0)
	}
	return field, selectedID, closeList
}
