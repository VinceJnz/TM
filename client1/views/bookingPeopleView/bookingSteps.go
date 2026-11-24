package bookingPeopleView

import (
	"client1/v2/views/utils/viewHelpers"
	"syscall/js"
)

// BookingSteps resumes a saved booking by ID, independently of the list page.
func (editor *ItemEditor) BookingSteps(container js.Value, tripContext string, backToDetails func(), onCheckout func(string), onConfirmed func()) {
	container.Set("textContent", "")
	context := editor.document.Call("createElement", "p")
	context.Set("className", "tm-booking-context")
	context.Set("textContent", tripContext)
	host := editor.document.Call("createElement", "section")
	host.Set("className", "tm-card tm-booking-form")
	container.Call("append", context, host)
	var showPeople func()
	showReview := func() {
		host.Set("textContent", "")
		load, review := viewHelpers.BookingReview(editor.document, editor.client, editor.ParentID, onCheckout, onConfirmed, showPeople)
		load.Set("hidden", true)
		host.Call("appendChild", load)
		host.Call("appendChild", review)
		load.Call("click")
	}
	showPeople = func() {
		host.Set("textContent", "")
		heading := editor.document.Call("createElement", "h3")
		heading.Set("textContent", "Step 2 of 4 · Add people")
		host.Call("appendChild", heading)
		people := New(editor.document, editor.events, editor.appCore, editor.ParentID)
		host.Call("appendChild", people.Div)
		people.FetchItems()
		goBack := viewHelpers.Button(func(js.Value, []js.Value) any {
			if people.ItemState != ItemStateNone && people.ItemState != ItemStateFetching {
				people.onCompletionMsg("Save or cancel the participant form before going back.")
				return nil
			}
			backToDetails()
			return nil
		}, editor.document, "Back to booking details", "booking-back-to-details")
		next := viewHelpers.Button(func(js.Value, []js.Value) any {
			if people.ItemState != ItemStateNone && people.ItemState != ItemStateFetching {
				people.onCompletionMsg("Save or cancel the participant form before continuing.")
				return nil
			}
			showReview()
			return nil
		}, editor.document, "Continue to review", "booking-continue-to-review")
		viewHelpers.StyleButtonPrimary(next)
		host.Call("appendChild", viewHelpers.FormButtonRow(editor.document, goBack, next))
	}
	showPeople()
}
