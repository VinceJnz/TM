package viewHelpers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"syscall/js"
	"time"
)

// FlexibleFloat handles numeric payloads that may arrive as numbers, strings, or decimal wrappers.
type FlexibleFloat struct {
	Value float64
	Valid bool
}

func (f *FlexibleFloat) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		f.Value = 0
		f.Valid = false
		return nil
	}

	var asFloat float64
	if err := json.Unmarshal(data, &asFloat); err == nil {
		f.Value = asFloat
		f.Valid = true
		return nil
	}

	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		parsed, parseErr := strconv.ParseFloat(strings.TrimSpace(asString), 64)
		if parseErr != nil {
			f.Value = 0
			f.Valid = false
			return nil
		}
		f.Value = parsed
		f.Valid = true
		return nil
	}

	var asObject map[string]any
	if err := json.Unmarshal(data, &asObject); err == nil {
		if validRaw, ok := asObject["Valid"]; ok {
			if valid, okBool := validRaw.(bool); okBool && !valid {
				f.Value = 0
				f.Valid = false
				return nil
			}
		}

		for _, key := range []string{"Decimal", "decimal", "Value", "value", "Float", "float"} {
			if raw, ok := asObject[key]; ok {
				switch value := raw.(type) {
				case float64:
					f.Value = value
					f.Valid = true
					return nil
				case string:
					parsed, parseErr := strconv.ParseFloat(strings.TrimSpace(value), 64)
					if parseErr == nil {
						f.Value = parsed
						f.Valid = true
						return nil
					}
				}
			}
		}
	}

	f.Value = 0
	f.Valid = false
	return nil
}

// BookingDropdownOption defines the fields used to render booking selectors consistently.
type BookingDropdownOption struct {
	ID           int           `json:"id"`
	OwnerName    string        `json:"owner_name"`
	TripName     string        `json:"trip_name"`
	BookingDate  time.Time     `json:"booking_date"`
	Notes        string        `json:"notes"`
	BookingCost  FlexibleFloat `json:"booking_cost"`
	BookingPrice FlexibleFloat `json:"booking_price"`
}

// BookingDropdown creates a labeled booking select control with a consistent option format.
func BookingDropdown(document js.Value, options []BookingDropdownOption, value int, labelText, htmlID string) (js.Value, js.Value) {
	fieldset := document.Call("createElement", "fieldset")
	fieldset.Set("className", "input-group")

	label := Label(document, labelText, htmlID)
	selectEl := document.Call("createElement", "select")
	selectEl.Set("id", htmlID)
	selectEl.Call("setAttribute", "required", "true")

	selected := false
	for _, item := range options {
		optionEl := document.Call("createElement", "option")
		optionEl.Set("value", strconv.Itoa(item.ID))

		parts := []string{fmt.Sprintf("Booking %d", item.ID)}
		if item.TripName != "" {
			parts = append(parts, "Trip: "+item.TripName)
		}
		if item.OwnerName != "" {
			parts = append(parts, "By: "+item.OwnerName)
		}
		if !item.BookingDate.IsZero() {
			parts = append(parts, "Booked: "+item.BookingDate.Format(Layout))
		}
		notes := strings.TrimSpace(item.Notes)
		if notes != "" {
			if len(notes) > 50 {
				notes = notes[:47] + "..."
			}
			parts = append(parts, "Notes: "+notes)
		}

		optionEl.Set("text", strings.Join(parts, " | "))
		if value == item.ID {
			optionEl.Set("selected", true)
			selected = true
		}
		selectEl.Call("appendChild", optionEl)
	}

	if value > 0 && !selected {
		optionEl := document.Call("createElement", "option")
		optionEl.Set("value", strconv.Itoa(value))
		optionEl.Set("text", fmt.Sprintf("Booking %d (current)", value))
		optionEl.Set("selected", true)
		selectEl.Call("appendChild", optionEl)
	}

	fieldset.Call("appendChild", label)
	fieldset.Call("appendChild", selectEl)
	return fieldset, selectEl
}
