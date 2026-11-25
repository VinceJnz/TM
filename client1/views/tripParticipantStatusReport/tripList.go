package tripParticipantStatusReport

import (
	"client1/v2/app/domEvents"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/bookingView"
	"client1/v2/views/utils/viewHelpers"
	"net/http"
	"strconv"
	"syscall/js"
)

func (e *ItemEditor) cancelLoads() {
	e.load.Cancel()
	for _, scope := range e.detailLoads {
		scope.Cancel()
	}
	for _, editor := range e.bookingEditors {
		editor.Hide()
	}
}
func (e *ItemEditor) FetchItems() {
	if e.RecordState != RecordStateReloadRequired {
		return
	}
	e.RecordState = RecordStateCurrent
	ticket := e.load.Start()
	page := e.Page.Snapshot()
	viewHelpers.Pending(e.Div, true)
	go func() {
		var rows []TableData
		var options []viewHelpers.ScheduleFilterOption
		var err error
		if !e.filters.Loaded {
			err = e.client.RequestContext(ticket.Context, http.MethodGet, ApiURL+"?view=filters", &options, nil)
		}
		if err == nil {
			err = e.client.PageRequestContext(ticket.Context, &page, http.MethodGet, e.filters.Path(ApiURL), &rows, nil)
		}
		if !ticket.Finish(err) {
			return
		}
		if err != nil {
			viewHelpers.Pending(e.Div, false)
			e.RecordState = RecordStateReloadRequired
			viewHelpers.RenderRetry(e.ListDiv, e.FetchItems)
			return
		}
		e.Page = page
		if !e.filters.Loaded {
			e.filters.SetOptions(options)
		}
		e.Records = rows
		e.populateItemList()
		viewHelpers.Pending(e.Div, false)
	}()
}
func (e *ItemEditor) populateItemList() {
	for _, scope := range e.detailLoads {
		scope.Cancel()
	}
	for _, editor := range e.bookingEditors {
		editor.Hide()
	}
	e.detailLoads = nil
	e.bookingEditors = nil
	e.ListDiv.Set("innerHTML", "")
	if len(e.Records) == 0 {
		empty := e.document.Call("createElement", "p")
		empty.Set("className", "tm-empty")
		empty.Set("textContent", "No trips match these filters.")
		e.ListDiv.Call("appendChild", empty)
	}
	for _, item := range e.Records {
		record := item
		card := e.document.Call("createElement", "section")
		card.Set("className", "tm-card tm-trip")
		row := e.document.Call("createElement", "div")
		row.Set("className", "tm-trip-row")
		toggle := e.document.Call("createElement", "button")
		toggle.Set("type", "button")
		toggle.Set("className", "tm-disclosure")
		arrow := e.document.Call("createElement", "span")
		arrow.Set("textContent", "\u25b6")
		arrow.Call("setAttribute", "aria-hidden", "true")
		title := e.document.Call("createElement", "strong")
		title.Set("textContent", record.TripName)
		dates := e.document.Call("createElement", "span")
		dates.Set("className", "tm-trip-meta")
		dates.Set("textContent", tripDates(record))
		meta := e.document.Call("createElement", "span")
		meta.Set("className", "tm-trip-meta")
		meta.Set("textContent", tripMetadata(record))
		toggle.Call("append", dates, arrow, " ", title, meta)
		details := e.document.Call("createElement", "div")
		details.Set("id", "trip-details-"+strconv.Itoa(record.TripID))
		details.Set("hidden", true)
		toggle.Call("setAttribute", "aria-controls", details.Get("id"))
		toggle.Call("setAttribute", "aria-expanded", "false")
		if record.Description != "" {
			description := e.document.Call("createElement", "p")
			description.Set("className", "tm-trip-description")
			description.Set("textContent", record.Description)
			details.Call("appendChild", description)
		}
		roster := e.document.Call("createElement", "div")
		details.Call("appendChild", roster)
		scope := &httpProcessor.LoadScope{}
		e.detailLoads = append(e.detailLoads, scope)
		page := &httpProcessor.Page{Limit: 50}
		loaded := false
		book := bookingView.New(e.document, e.events, e.appCore, bookingView.ParentData{ID: record.TripID, Name: record.TripName, FromDate: record.TripFrom, ToDate: record.TripTo})
		book.Hide()
		e.bookingEditors = append(e.bookingEditors, book)
		details.Call("appendChild", book.Div)
		open := func(value bool) {
			details.Set("hidden", !value)
			toggle.Call("setAttribute", "aria-expanded", value)
			if value {
				arrow.Set("textContent", "\u25bc")
			} else {
				arrow.Set("textContent", "\u25b6")
				scope.Cancel()
				loaded = false
				book.Hide()
			}
		}
		domEvents.Listen(toggle, "click", func(js.Value, []js.Value) any {
			expanded := toggle.Call("getAttribute", "aria-expanded").String() != "true"
			open(expanded)
			if expanded && !loaded {
				loaded = true
				e.fetchRoster(record.TripID, record.CanReadDetails, roster, page, scope)
			}
			return nil
		})
		makeBooking := viewHelpers.Button(func(js.Value, []js.Value) any {
			open(true)
			if !loaded {
				loaded = true
				e.fetchRoster(record.TripID, record.CanReadDetails, roster, page, scope)
			}
			book.NewItemData(js.Null(), nil)
			book.Display()
			return nil
		}, e.document, "Book this trip", "trip-book-"+strconv.Itoa(record.TripID))
		makeBooking.Set("className", "btn btn-primary")
		viewHelpers.ActionFor(makeBooking, "bookings", "POST")
		makeBooking.Set("hidden", !e.appCore.HasAccess("bookings", []string{"POST"}, []string{"own", "any"}))
		row.Call("append", toggle, makeBooking)
		card.Call("append", row, details)
		e.ListDiv.Call("appendChild", card)
	}
	viewHelpers.PageControls(e.ListDiv, &e.Page, len(e.Records), func() { e.RecordState = RecordStateReloadRequired; e.FetchItems() })
}
func (e *ItemEditor) fetchRoster(id int, canReadDetails bool, root js.Value, page *httpProcessor.Page, scope *httpProcessor.LoadScope) {
	ticket := scope.Start()
	pending := page.Snapshot()
	viewHelpers.Pending(root, true)
	go func() {
		var rows []TableData
		err := e.client.PageRequestContext(ticket.Context, &pending, http.MethodGet, ApiURL+"?trip_id="+strconv.Itoa(id), &rows, nil)
		if !ticket.Finish(err) {
			return
		}
		viewHelpers.Pending(root, false)
		if err != nil {
			viewHelpers.RenderRetry(root, func() { e.fetchRoster(id, canReadDetails, root, page, scope) })
			return
		}
		*page = pending
		root.Set("innerHTML", "")
		count := 0
		var booking js.Value
		lastBooking := -1
		for _, r := range rows {
			if r.PersonName == "" {
				continue
			}
			count++
			if r.BookingID != lastBooking {
				lastBooking = r.BookingID
				booking = e.document.Call("createElement", "div")
				booking.Set("className", "tm-card")
				root.Call("appendChild", booking)
			}
			participant := e.document.Call("createElement", "p")
			participant.Set("textContent", r.PersonName)
			colors := participantStatusColors(r.BookingStatus)
			participant.Get("style").Set("background", colors.background)
			if r.BookingPosition > 0 {
				participant.Call("append", " | Status: "+participantStatusLabel(r.BookingStatus)+" | Position: "+strconv.Itoa(r.BookingPosition))
			}
			booking.Call("appendChild", participant)
			if canReadDetails && r.ParticipantID > 0 {
				e.addLeaderDetails(booking, r.ParticipantID)
			}
		}
		viewHelpers.EmptyState(root, count)
		viewHelpers.PageControls(root, page, len(rows), func() { e.fetchRoster(id, canReadDetails, root, page, scope) })
	}()
}
