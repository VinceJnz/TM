package viewHelpers

import (
	"client1/v2/app/domEvents"
	"net/url"
	"strconv"
	"strings"
	"syscall/js"
	"time"
)

type ScheduleFilterOption struct {
	Kind  string `json:"kind"`
	ID    int    `json:"id"`
	Label string `json:"label"`
}

type ScheduleFilters struct {
	month, period, tripType, grade js.Value
	query                          url.Values
	Loaded                         bool
}

// Mount adds the shared schedule controls. Prefix gives each view its own URL settings.
func (f *ScheduleFilters) Mount(doc, parent js.Value, headingText, prefix, defaultPeriod string, onApply func()) js.Value {
	form := doc.Call("createElement", "form")
	form.Set("className", "tm-schedule-filters")
	if headingText != "" {
		heading := doc.Call("createElement", "h1")
		heading.Set("textContent", headingText)
		parent.Call("appendChild", heading)
	}
	parent.Call("appendChild", form)
	control := func(label, tag string) js.Value {
		field := doc.Call("createElement", "label")
		input := doc.Call("createElement", tag)
		field.Call("append", label, input)
		form.Call("appendChild", field)
		return input
	}
	f.query = url.Values{}
	query := scheduleLocationQuery()
	keys := []string{"month", "type", "grade"}
	for _, key := range keys {
		if value := query.Get(prefix + key); value != "" {
			f.query.Set(key, value)
		}
	}
	f.period = control("Show", "select")
	scheduleOption(f.period, "month", "By month")
	scheduleOption(f.period, "upcoming", "All upcoming")
	scheduleOption(f.period, "past", "Past trips")
	f.month = control("Month", "input")
	f.month.Set("type", "month")
	f.month.Set("pattern", "[0-9]{4}-[0-9]{2}")
	f.month.Set("placeholder", "YYYY-MM")
	f.month.Set("required", true)
	month := f.query.Get("month")
	period := defaultPeriod
	if month == "past" || month == "upcoming" {
		period = month
	}
	if _, err := time.Parse("2006-01", month); err == nil {
		period = "month"
	} else {
		now := js.Global().Get("Date").New()
		month = time.Date(now.Call("getFullYear").Int(), time.Month(now.Call("getMonth").Int()+1), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
	}
	f.month.Set("value", month)
	f.period.Set("value", period)
	f.tripType = control("Trip type", "select")
	f.grade = control("Grade", "select")
	scheduleOption(f.tripType, "", "All types")
	scheduleOption(f.grade, "", "All grades")
	updateMonth := func() { f.month.Set("disabled", f.period.Get("value").String() != "month") }
	updateMonth()
	domEvents.Listen(f.period, "change", func(js.Value, []js.Value) any { updateMonth(); return nil })
	selectedPeriod := func() string {
		if period := f.period.Get("value").String(); period != "month" {
			return period
		}
		return f.month.Get("value").String()
	}
	f.query.Set("month", selectedPeriod())
	apply := func() {
		f.query.Set("month", selectedPeriod())
		for key, field := range map[string]js.Value{"type": f.tripType, "grade": f.grade} {
			if f.Loaded {
				f.query.Del(key)
				if value := field.Get("value").String(); value != "" {
					f.query.Set(key, value)
				}
			}
		}
		location := js.Global().Get("location")
		query := scheduleLocationQuery()
		for _, key := range keys {
			query.Del(prefix + key)
			if value := f.query.Get(key); value != "" {
				query.Set(prefix+key, value)
			}
		}
		js.Global().Get("history").Call("replaceState", js.Null(), "", location.Get("pathname").String()+"?"+query.Encode()+location.Get("hash").String())
		onApply()
	}
	for _, step := range []struct {
		label string
		delta int
	}{{"Previous month", -1}, {"Next month", 1}} {
		form.Call("appendChild", Button(func(js.Value, []js.Value) any {
			date, err := time.Parse("2006-01", f.month.Get("value").String())
			if err != nil {
				f.month.Call("reportValidity")
				return nil
			}
			f.period.Set("value", "month")
			updateMonth()
			f.month.Set("value", date.AddDate(0, step.delta, 0).Format("2006-01"))
			apply()
			return nil
		}, doc, step.label, ""))
	}
	submit := doc.Call("createElement", "button")
	submit.Set("type", "submit")
	submit.Set("className", "btn btn-primary")
	submit.Set("textContent", "Apply filters")
	form.Call("appendChild", submit)
	form.Call("appendChild", Button(func(js.Value, []js.Value) any {
		f.tripType.Set("value", "")
		f.grade.Set("value", "")
		f.query.Del("type")
		f.query.Del("grade")
		apply()
		return nil
	}, doc, "Clear type and grade", ""))
	domEvents.Listen(form, "submit", func(_ js.Value, args []js.Value) any {
		args[0].Call("preventDefault")
		if f.month.Get("disabled").Bool() || f.month.Call("checkValidity").Bool() {
			apply()
		}
		return nil
	})
	return form
}

func scheduleLocationQuery() url.Values {
	query, _ := url.ParseQuery(strings.TrimPrefix(js.Global().Get("location").Get("search").String(), "?"))
	return query
}

func scheduleOption(selectBox js.Value, value, label string) {
	option := selectBox.Get("ownerDocument").Call("createElement", "option")
	option.Set("value", value)
	option.Set("textContent", label)
	selectBox.Call("appendChild", option)
}

func (f *ScheduleFilters) SetOptions(options []ScheduleFilterOption) {
	for _, option := range options {
		target := f.tripType
		if option.Kind == "grade" {
			target = f.grade
		}
		scheduleOption(target, strconv.Itoa(option.ID), option.Label)
	}
	f.tripType.Set("value", f.query.Get("type"))
	f.grade.Set("value", f.query.Get("grade"))
	f.Loaded = true
}

func (f *ScheduleFilters) Path(path string) string {
	return path + "?view=schedule&" + f.query.Encode()
}
