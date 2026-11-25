package viewHelpers

import (
	"client1/v2/app/domEvents"
	"client1/v2/app/httpProcessor"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"syscall/js"
	"time"
)

// BookingReturn keeps the list action available while the booking step contents change.
func BookingReturn(doc, editor js.Value, saving func() bool, back func()) js.Value {
	button := Button(func(js.Value, []js.Value) any {
		if saving() || !editor.Call("querySelector", "[data-booking-request]").IsNull() {
			return nil
		}
		if !editor.Call("querySelector", "form[data-booking-dirty]").IsNull() &&
			!js.Global().Call("confirm", "Discard unsaved changes and return to My Bookings?").Bool() {
			return nil
		}
		back()
		return nil
	}, doc, "Back to My Bookings", "")
	StyleButtonSecondary(button)
	button.Set("hidden", true)
	markDirty := func(_ js.Value, args []js.Value) any {
		form := args[0].Get("target").Call("closest", "form")
		if !form.IsNull() {
			form.Call("setAttribute", "data-booking-dirty", "")
		}
		return nil
	}
	domEvents.Listen(editor, "input", markDirty)
	domEvents.Listen(editor, "change", markDirty)
	return button
}

func BookingReview(doc js.Value, client *httpProcessor.Client, id int, onCheckout func(string), onConfirmed func(), backToPeople ...func()) (js.Value, js.Value) {
	panel := doc.Call("createElement", "div")
	panel.Call("setAttribute", "aria-live", "polite")
	path := "/" + strconv.Itoa(id)
	const checkout = "/bookings/checkout/"
	show := func(message string) {
		panel.Set("textContent", message)
	}
	reviewButton := doc.Call("createElement", "button")
	reviewButton.Set("type", "button")
	reviewButton.Set("className", "btn btn-secondary")
	reviewButton.Set("textContent", "Review booking")
	guided := len(backToPeople) > 0
	backButton := func(label string, back func()) js.Value {
		return Button(func(js.Value, []js.Value) any {
			back()
			return nil
		}, doc, label, "booking-step-back")
	}
	heading := func(title string) {
		element := doc.Call("createElement", "h3")
		element.Set("textContent", title)
		panel.Call("appendChild", element)
	}
	domEvents.Listen(reviewButton, "click", func(js.Value, []js.Value) any {
		reviewButton.Set("disabled", true)
		show("Loading booking review…")
		go func() {
			var review struct {
				AmountMinor  int64     `json:"amount_minor"`
				VoucherCode  string    `json:"voucher_code"`
				TripName     string    `json:"trip_name"`
				Notes        string    `json:"notes"`
				FromDate     time.Time `json:"from_date"`
				ToDate       time.Time `json:"to_date"`
				Participants []string  `json:"participants"`
			}
			err := client.Request(http.MethodGet, checkout+"review"+path, &review, nil)
			if !panel.Get("isConnected").Bool() {
				return
			}
			reviewButton.Set("disabled", false)
			if err != nil {
				show(err.Error())
				if guided {
					panel.Call("appendChild", backButton("Back to people", backToPeople[0]))
				}
				return
			}
			show("")
			if guided {
				heading("Step 3 of 4 · Review booking")
			}
			addDetail := func(value string) {
				detail := doc.Call("createElement", "p")
				detail.Set("textContent", value)
				panel.Call("appendChild", detail)
			}
			addDetail(review.TripName)
			addDetail(review.FromDate.Format(Layout) + " to " + review.ToDate.Format(Layout))
			if review.Notes != "" {
				addDetail(review.Notes)
			}
			addDetail("Participants: " + strings.Join(review.Participants, ", "))
			if review.VoucherCode != "" {
				addDetail("Voucher: " + review.VoucherCode)
			}
			addDetail(fmt.Sprintf("Total: NZ$%d.%02d", review.AmountMinor/100, review.AmountMinor%100))
			label := "Pay for booking"
			if review.AmountMinor == 0 {
				label = "Confirm booking"
			}
			var action js.Value
			var actionBack js.Value
			action = Button(func(js.Value, []js.Value) any {
				action.Set("disabled", true)
				reviewButton.Set("disabled", true)
				panel.Call("setAttribute", "data-booking-request", "")
				if guided {
					actionBack.Set("disabled", true)
				}
				go func() {
					defer panel.Call("removeAttribute", "data-booking-request")
					var err error
					if review.AmountMinor == 0 {
						err = client.Request(http.MethodPost, checkout+"confirm"+path, nil, nil)
						if err == nil {
							if panel.Get("isConnected").Bool() {
								show("Booking confirmed.")
								reviewButton.Set("hidden", true)
								onConfirmed()
							}
							return
						}
					} else {
						var response struct {
							URL string `json:"checkout_url"`
						}
						err = client.Request(http.MethodPost, checkout+"create"+path, &response, map[string]int64{"amount_minor": review.AmountMinor})
						if err == nil && response.URL != "" {
							if panel.Get("isConnected").Bool() {
								reviewButton.Set("disabled", false)
								action.Set("disabled", false)
								if guided {
									actionBack.Set("disabled", false)
								}
								onCheckout(response.URL)
							}
							return
						}
					}
					if panel.Get("isConnected").Bool() {
						reviewButton.Set("disabled", false)
						if err != nil {
							show(err.Error())
						} else {
							show("Checkout could not be opened. Review the booking and try again.")
						}
						if guided {
							panel.Call("appendChild", backButton("Back to review", func() { reviewButton.Call("click") }))
						}
					}
				}()
				return nil
			}, doc, label, "booking-review-action-"+strconv.Itoa(id))
			StyleButtonPrimary(action)
			if !guided {
				panel.Call("appendChild", action)
				return
			}
			if review.AmountMinor == 0 {
				actionBack = backButton("Back to people", backToPeople[0])
				panel.Call("appendChild", FormButtonRow(doc, actionBack, action))
				return
			}
			next := Button(func(js.Value, []js.Value) any {
				show("")
				heading("Step 4 of 4 · Pay for booking")
				addDetail(fmt.Sprintf("Total: NZ$%d.%02d", review.AmountMinor/100, review.AmountMinor%100))
				actionBack = backButton("Back to review", func() { reviewButton.Call("click") })
				panel.Call("appendChild", FormButtonRow(doc, actionBack, action))
				return nil
			}, doc, "Continue to payment", "booking-continue-to-payment")
			StyleButtonPrimary(next)
			panel.Call("appendChild", FormButtonRow(doc, backButton("Back to people", backToPeople[0]), next))
		}()
		return nil
	})
	return reviewButton, panel
}
