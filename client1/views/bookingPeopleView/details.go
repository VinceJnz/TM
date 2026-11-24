package bookingPeopleView

import (
	"client1/v2/app/api"
	"encoding/json"
	"strings"
	"syscall/js"
)

func (e *ItemEditor) participantFields(form js.Value, questionnaire api.Questionnaire, details api.ParticipantDetails) func() *api.ParticipantDetails {
	identity := e.fieldGroup(form, "Participant details", "")
	emergency := e.fieldGroup(form, "Emergency contact", "Choose someone who is not on the trip.")
	type field struct {
		label, kind string
		target      *string
		max         int
		group       js.Value
	}
	fields := []field{
		{"Full name", "text", &details.Name, 255, identity},
		{"Mobile phone", "tel", &details.Mobile, 50, identity},
		{"Home address", "textarea", &details.Address, 500, identity},
		{"Emergency contact name", "text", &details.EmergencyName, 255, emergency},
		{"Emergency contact phone", "tel", &details.EmergencyPhone, 50, emergency},
		{"Dietary requirements (enter None if none)", "textarea", &details.Dietary, 2000, identity},
	}
	inputs := make([]js.Value, len(fields))
	for index, item := range fields {
		tag := "input"
		if item.kind == "textarea" {
			tag = "textarea"
		}
		input := e.labelledField(item.group, item.label, tag)
		if tag == "input" {
			input.Set("type", item.kind)
		}
		input.Set("maxLength", item.max)
		input.Set("value", *item.target)
		input.Set("required", true)
		inputs[index] = input
	}
	membership := e.labelledField(identity, "Membership", "select")
	identity.Call("insertBefore", membership.Get("parentElement"), inputs[2].Get("parentElement"))
	membership.Call("appendChild", e.option("", "Select membership"))
	membership.Call("appendChild", e.option("member", "TTC member"))
	membership.Call("appendChild", e.option("non_member", "Non-member"))
	membership.Call("appendChild", e.option("affiliated_member", "Affiliated member"))
	membership.Set("value", details.Membership)
	membership.Set("required", true)

	readers := make(map[string]func() any, len(questionnaire.Questions))
	if len(questionnaire.Questions) > 0 {
		questions := e.fieldGroup(form, "Trip questions", "")
		for _, question := range questionnaire.Questions {
			readers[question.Key] = e.questionField(questions, question, details.Answers[question.Key])
		}
	}
	return func() *api.ParticipantDetails {
		for index, item := range fields {
			*item.target = strings.TrimSpace(inputs[index].Get("value").String())
		}
		details.Membership = membership.Get("value").String()
		details.QuestionnaireRevision = questionnaire.Revision
		details.Answers = make(map[string]json.RawMessage, len(readers))
		for key, read := range readers {
			if answer := read(); answer != nil {
				raw, _ := json.Marshal(answer)
				details.Answers[key] = raw
			}
		}
		return &details
	}
}

func (e *ItemEditor) fieldGroup(parent js.Value, title, hint string) js.Value {
	group := e.document.Call("createElement", "fieldset")
	group.Set("className", "tm-participant-group")
	legend := e.document.Call("createElement", "legend")
	legend.Set("textContent", title)
	group.Call("appendChild", legend)
	if hint != "" {
		description := e.document.Call("createElement", "p")
		description.Set("className", "tm-trip-meta")
		description.Set("textContent", hint)
		group.Call("appendChild", description)
	}
	grid := e.document.Call("createElement", "div")
	grid.Set("className", "tm-booking-fields")
	group.Call("appendChild", grid)
	parent.Call("appendChild", group)
	return grid
}

func (e *ItemEditor) labelledField(parent js.Value, label, tag string) js.Value {
	wrapper := e.document.Call("createElement", "label")
	wrapper.Set("className", "tm-participant-field")
	input := e.document.Call("createElement", tag)
	wrapper.Call("append", label, input)
	parent.Call("appendChild", wrapper)
	return input
}

func (e *ItemEditor) option(value, label string) js.Value {
	option := e.document.Call("createElement", "option")
	option.Set("value", value)
	option.Set("textContent", label)
	return option
}

func (e *ItemEditor) questionField(form js.Value, question api.Question, previous json.RawMessage) func() any {
	if question.Type == "multiple_choice" {
		label := question.Label
		if question.Required {
			label += " (select at least one)"
		}
		field := e.labelledField(form, label, "select")
		field.Set("multiple", true)
		field.Set("required", question.Required)
		var values []string
		_ = json.Unmarshal(previous, &values)
		for _, value := range question.Options {
			option := e.option(value, value)
			field.Call("appendChild", option)
			for _, selected := range values {
				if selected == value {
					option.Set("selected", true)
				}
			}
		}
		return func() any {
			selected := field.Get("selectedOptions")
			if selected.Length() == 0 {
				return nil
			}
			values := make([]string, selected.Length())
			for index := range values {
				values[index] = selected.Index(index).Get("value").String()
			}
			return values
		}
	}
	if question.Type == "text" {
		field := e.labelledField(form, question.Label, "textarea")
		field.Set("required", question.Required)
		field.Set("maxLength", 4000)
		var value string
		_ = json.Unmarshal(previous, &value)
		field.Set("value", value)
		return func() any {
			if text := strings.TrimSpace(field.Get("value").String()); text != "" {
				return text
			}
			return nil
		}
	}
	field := e.labelledField(form, question.Label, "select")
	field.Set("required", question.Required)
	field.Call("appendChild", e.option("", "Select an answer"))
	if question.Type == "yes_no" {
		field.Call("appendChild", e.option("yes", "Yes"))
		field.Call("appendChild", e.option("no", "No"))
		if string(previous) == "true" {
			field.Set("value", "yes")
		} else if string(previous) == "false" {
			field.Set("value", "no")
		}
		return func() any {
			if value := field.Get("value").String(); value != "" {
				return value == "yes"
			}
			return nil
		}
	}
	var previousValue string
	_ = json.Unmarshal(previous, &previousValue)
	for _, value := range question.Options {
		field.Call("appendChild", e.option(value, value))
	}
	field.Set("value", previousValue)
	return func() any {
		if selected := field.Get("value").String(); selected != "" {
			return selected
		}
		return nil
	}
}
