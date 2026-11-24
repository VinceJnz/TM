package tripView

import (
	"client1/v2/app/api"
	"client1/v2/app/domEvents"
	"client1/v2/views/utils/viewHelpers"
	"net/http"
	"strconv"
	"strings"
	"syscall/js"
)

func (e *ItemEditor) openQuestionnaire(trip TableData) {
	e.editLoad.Cancel()
	ticket := e.questionLoad.Start()
	e.EditDiv.Set("innerHTML", "")
	e.EditDiv.Get("style").Set("display", "block")
	path := ApiURL + "/" + strconv.Itoa(trip.ID) + "/questionnaire"
	go func() {
		var questionnaire api.Questionnaire
		err := e.client.RequestContext(ticket.Context, http.MethodGet, path, &questionnaire, nil)
		if !ticket.Finish(err) {
			return
		}
		if err != nil {
			viewHelpers.RenderRetry(e.EditDiv, func() { e.openQuestionnaire(trip) })
			return
		}
		e.renderQuestionnaire(trip, path, questionnaire)
	}()
}

func (e *ItemEditor) renderQuestionnaire(trip TableData, path string, questionnaire api.Questionnaire) {
	form := e.document.Call("createElement", "form")
	form.Set("className", "tm-card")
	heading := e.document.Call("createElement", "h2")
	heading.Set("textContent", "Booking questions for "+trip.Name)
	form.Call("appendChild", heading)
	rows := e.document.Call("createElement", "div")
	form.Call("appendChild", rows)
	addQuestion := func(question api.Question) {
		row := e.document.Call("createElement", "fieldset")
		field := func(label, tag, value string) js.Value {
			wrapper := e.document.Call("createElement", "label")
			input := e.document.Call("createElement", tag)
			input.Set("value", value)
			wrapper.Call("append", label, input)
			row.Call("appendChild", wrapper)
			return input
		}
		key := field("Question key", "input", question.Key)
		key.Set("name", "key")
		key.Set("required", true)
		key.Set("readOnly", question.Key != "")
		key.Set("pattern", "[a-z][a-z0-9_]{0,63}")
		label := field("Question", "input", question.Label)
		label.Set("name", "label")
		label.Set("required", true)
		label.Set("maxLength", 255)
		kind := field("Answer format", "select", "")
		kind.Set("name", "kind")
		for _, format := range []struct{ value, label string }{
			{"text", "Text"}, {"yes_no", "Yes or no"}, {"single_choice", "Single choice"}, {"multiple_choice", "Multiple choices"},
		} {
			option := e.document.Call("createElement", "option")
			option.Set("value", format.value)
			option.Set("textContent", format.label)
			kind.Call("appendChild", option)
		}
		if question.Type != "" {
			kind.Set("value", question.Type)
		}
		required := field("Required", "input", "")
		required.Set("name", "required")
		required.Set("type", "checkbox")
		required.Set("checked", question.Required)
		options := field("Choices, one per line", "textarea", strings.Join(question.Options, "\n"))
		options.Set("name", "options")
		setOptions := func() {
			isChoice := strings.HasSuffix(kind.Get("value").String(), "_choice")
			options.Get("parentElement").Set("hidden", !isChoice)
			options.Set("required", isChoice)
		}
		domEvents.Listen(kind, "change", func(js.Value, []js.Value) any {
			setOptions()
			return nil
		})
		setOptions()
		remove := viewHelpers.Button(func(js.Value, []js.Value) any {
			row.Call("remove")
			return nil
		}, e.document, "Remove question", "")
		up := viewHelpers.Button(func(js.Value, []js.Value) any {
			if previous := row.Get("previousElementSibling"); previous.Truthy() {
				rows.Call("insertBefore", row, previous)
			}
			return nil
		}, e.document, "Move earlier", "")
		row.Call("append", up, remove)
		rows.Call("appendChild", row)
	}
	for _, question := range questionnaire.Questions {
		addQuestion(question)
	}
	add := viewHelpers.Button(func(js.Value, []js.Value) any {
		addQuestion(api.Question{})
		return nil
	}, e.document, "Add question", "")
	cancel := viewHelpers.Button(func(js.Value, []js.Value) any {
		e.questionLoad.Cancel()
		e.EditDiv.Set("innerHTML", "")
		return nil
	}, e.document, "Cancel", "")
	publish := viewHelpers.SubmitButton(e.document, "Publish questions", "")
	viewHelpers.StyleButtonPrimary(publish)
	form.Call("append", add, publish, cancel)
	domEvents.Listen(form, "submit", func(_ js.Value, args []js.Value) any {
		args[0].Call("preventDefault")
		input := api.Questionnaire{Revision: questionnaire.Revision, Questions: []api.Question{}}
		children := rows.Get("children")
		for index := 0; index < children.Length(); index++ {
			row := children.Index(index)
			field := func(name string) js.Value { return row.Call("querySelector", "[name="+name+"]") }
			question := api.Question{
				Key: field("key").Get("value").String(), Label: field("label").Get("value").String(),
				Type: field("kind").Get("value").String(), Required: field("required").Get("checked").Bool(),
			}
			if strings.HasSuffix(question.Type, "_choice") {
				for _, option := range strings.Split(field("options").Get("value").String(), "\n") {
					if option = strings.TrimSpace(option); option != "" {
						question.Options = append(question.Options, option)
					}
				}
			}
			input.Questions = append(input.Questions, question)
		}
		ticket := e.questionLoad.Start()
		viewHelpers.Pending(form, true)
		go func() {
			var published api.Questionnaire
			err := e.client.RequestContext(ticket.Context, http.MethodPut, path, &published, &input)
			if !ticket.Finish(err) {
				return
			}
			viewHelpers.Pending(form, false)
			if err != nil {
				e.onCompletionMsg("Questions could not be published. Check the fields or reopen the editor to load the current revision.")
				return
			}
			e.EditDiv.Set("innerHTML", "")
			e.onCompletionMsg("Booking questions published.")
		}()
		return nil
	})
	e.EditDiv.Call("appendChild", form)
}
