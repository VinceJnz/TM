package tripParticipantStatusReport

import (
	"client1/v2/app/domEvents"
	"client1/v2/views/utils/viewHelpers"
	"net/url"
	"strconv"
	"strings"
	"syscall/js"
	"time"
)

type filterOption struct {
	Kind  string `json:"kind"`
	ID    int    `json:"id"`
	Label string `json:"label"`
}

type scheduleFilters struct {
	month, period, tripType, grade js.Value
	query                          url.Values
	loaded                         bool
}

func (e *ItemEditor) addScheduleFilters() {
	form := e.document.Call("createElement", "form")
	form.Set("className", "tm-schedule-filters")
	heading := e.document.Call("createElement", "h1")
	heading.Set("textContent", "Trip schedule")
	e.Div.Call("append", heading, form)
	control := func(label, tag string) js.Value {
		field := e.document.Call("createElement", "label")
		input := e.document.Call("createElement", tag)
		field.Call("append", label, input)
		form.Call("appendChild", field)
		return input
	}
	f := &e.filters
	f.query, _ = url.ParseQuery(strings.TrimPrefix(js.Global().Get("location").Get("search").String(), "?"))
	f.period = control("Show", "select")
	e.addScheduleOption(f.period, "month", "By month")
	e.addScheduleOption(f.period, "upcoming", "All upcoming")
	e.addScheduleOption(f.period, "past", "Past trips")
	f.month = control("Month", "input")
	f.month.Set("type", "month")
	f.month.Set("pattern", "[0-9]{4}-[0-9]{2}")
	f.month.Set("placeholder", "YYYY-MM")
	f.month.Set("required", true)
	month := f.query.Get("month")
	f.period.Set("value", "month")
	if month == "upcoming" || month == "past" {
		f.period.Set("value", month)
	}
	if _, err := time.Parse("2006-01", month); err != nil {
		now := js.Global().Get("Date").New()
		month = time.Date(now.Call("getFullYear").Int(), time.Month(now.Call("getMonth").Int()+1), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
	}
	f.month.Set("value", month)
	f.tripType = control("Trip type", "select")
	f.grade = control("Grade", "select")
	e.addScheduleOption(f.tripType, "", "All types")
	e.addScheduleOption(f.grade, "", "All grades")
	updateMonthState := func() {
		f.month.Set("disabled", f.period.Get("value").String() != "month")
	}
	updateMonthState()
	domEvents.Listen(f.period, "change", func(js.Value, []js.Value) any {
		updateMonthState()
		return nil
	})
	for _, step := range []struct {
		label string
		delta int
	}{{"Previous month", -1}, {"Next month", 1}} {
		button := viewHelpers.Button(func(js.Value, []js.Value) any {
			date, err := time.Parse("2006-01", f.month.Get("value").String())
			if err != nil {
				f.month.Call("reportValidity")
				return nil
			}
			f.period.Set("value", "month")
			updateMonthState()
			f.month.Set("value", date.AddDate(0, step.delta, 0).Format("2006-01"))
			e.applyScheduleFilters()
			return nil
		}, e.document, step.label, "")
		button.Set("type", "button")
		button.Set("className", "btn btn-secondary")
		form.Call("appendChild", button)
	}
	submit := e.document.Call("createElement", "button")
	submit.Set("type", "submit")
	submit.Set("className", "btn btn-primary")
	submit.Set("textContent", "Apply filters")
	form.Call("appendChild", submit)
	clear := viewHelpers.Button(func(js.Value, []js.Value) any {
		f.tripType.Set("value", "")
		f.grade.Set("value", "")
		e.applyScheduleFilters()
		return nil
	}, e.document, "Clear type and grade", "")
	clear.Set("type", "button")
	clear.Set("className", "btn btn-secondary")
	form.Call("appendChild", clear)
	domEvents.Listen(form, "submit", func(_ js.Value, args []js.Value) any {
		args[0].Call("preventDefault")
		if f.month.Get("disabled").Bool() || f.month.Call("checkValidity").Bool() {
			e.applyScheduleFilters()
		}
		return nil
	})
	f.query.Set("month", e.selectedPeriod())
}

func (e *ItemEditor) addScheduleOption(selectBox js.Value, value, label string) {
	option := e.document.Call("createElement", "option")
	option.Set("value", value)
	option.Set("textContent", label)
	selectBox.Call("appendChild", option)
}

func (e *ItemEditor) setFilterOptions(options []filterOption) {
	f := &e.filters
	for _, option := range options {
		target := f.tripType
		if option.Kind == "grade" {
			target = f.grade
		}
		e.addScheduleOption(target, strconv.Itoa(option.ID), option.Label)
	}
	if value := f.query.Get("type"); value != "" {
		f.tripType.Set("value", value)
	}
	if value := f.query.Get("grade"); value != "" {
		f.grade.Set("value", value)
	}
	f.loaded = true
}

func (e *ItemEditor) selectedPeriod() string {
	if period := e.filters.period.Get("value").String(); period != "month" {
		return period
	}
	return e.filters.month.Get("value").String()
}

func (e *ItemEditor) applyScheduleFilters() {
	f := &e.filters
	f.query.Set("month", e.selectedPeriod())
	for key, field := range map[string]js.Value{"type": f.tripType, "grade": f.grade} {
		if value := field.Get("value").String(); value != "" {
			f.query.Set(key, value)
		} else {
			f.query.Del(key)
		}
	}
	location := js.Global().Get("location")
	js.Global().Get("history").Call("replaceState", js.Null(), "", location.Get("pathname").String()+"?"+f.query.Encode()+location.Get("hash").String())
	e.cancelLoads()
	e.RecordState = RecordStateReloadRequired
	e.FetchItems()
}

func (e *ItemEditor) schedulePath() string {
	query := url.Values{"view": {"schedule"}}
	for _, key := range []string{"month", "type", "grade"} {
		if value := e.filters.query.Get(key); value != "" {
			query.Set(key, value)
		}
	}
	return ApiURL + "?" + query.Encode()
}

func tripDates(trip TableData) string {
	if trip.TripFrom.Equal(trip.TripTo) {
		return trip.TripFrom.Format("Mon 2 Jan 2006")
	}
	return trip.TripFrom.Format("Mon 2 Jan 2006") + " – " + trip.TripTo.Format("Mon 2 Jan 2006")
}

func tripMetadata(trip TableData) string {
	var labels []string
	for _, value := range []string{trip.Location, trip.Type, trip.Grade, trip.Status} {
		if value != "" {
			labels = append(labels, value)
		}
	}
	return strings.Join(labels, " · ")
}
