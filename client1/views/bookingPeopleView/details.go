package bookingPeopleView

import (
	"client1/v2/app/api"
	"encoding/json"
	"strings"
	"syscall/js"
)

func (e *ItemEditor) participantFields(form js.Value, questionnaire api.Questionnaire, details api.ParticipantDetails) func() *api.ParticipantDetails {
	type field struct {
		label, kind string
		target      *string
		max         int
	}
	fields := []field{
		{"Full name", "text", &details.Name, 255},
		{"Mobile phone", "tel", &details.Mobile, 50},
		{"Home address", "text", &details.Address, 500},
		{"Emergency contact name (someone outside the trip)", "text", &details.EmergencyName, 255},
		{"Emergency contact phone", "tel", &details.EmergencyPhone, 50},
		{"Dietary requirements (enter None if none)", "text", &details.Dietary, 2000},
	}
	inputs := make([]js.Value, len(fields))
	for index, item := range fields {
		input := e.labelledField(form, item.label, "input")
		input.Set("type", item.kind)
		input.Set("maxLength", item.max)
		input.Set("value", *item.target)
		input.Set("required", true)
		inputs[index] = input
	}
	membership := e.labelledField(form, "Membership", "select")
	membership.Call("appendChild", e.option("", "Select membership"))
	membership.Call("appendChild", e.option("member", "TTC member"))
	membership.Call("appendChild", e.option("non_member", "Non-member"))
	membership.Call("appendChild", e.option("affiliated_member", "Affiliated member"))
	membership.Set("value", details.Membership)
	membership.Set("required", true)

	readers := make(map[string]func() any, len(questionnaire.Questions))
	for _, question := range questionnaire.Questions {
		readers[question.Key] = e.questionField(form, question, details.Answers[question.Key])
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
