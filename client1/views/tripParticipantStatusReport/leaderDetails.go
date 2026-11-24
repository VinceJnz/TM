package tripParticipantStatusReport

import (
	"client1/v2/app/api"
	"client1/v2/app/domEvents"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"syscall/js"
)

func (e *ItemEditor) addLeaderDetails(parent js.Value, participant int) {
	panel := e.document.Call("createElement", "details")
	summary := e.document.Call("createElement", "summary")
	summary.Set("textContent", "Participant details")
	content := e.document.Call("createElement", "div")
	panel.Call("append", summary, content)
	parent.Call("appendChild", panel)
	scope := &httpProcessor.LoadScope{}
	e.detailLoads = append(e.detailLoads, scope)
	load := func() {
		content.Set("textContent", "Loading participant details…")
		ticket := scope.Start()
		go func() {
			var details api.ParticipantDetails
			path := ApiURL + "/participants/" + strconv.Itoa(participant) + "/details"
			err := e.client.RequestContext(ticket.Context, http.MethodGet, path, &details, nil)
			if !ticket.Finish(err) || !panel.Get("isConnected").Bool() {
				return
			}
			if err != nil {
				content.Set("textContent", "Participant details could not be loaded. Close and reopen to retry.")
				return
			}
			e.renderLeaderDetails(content, details)
		}()
	}
	domEvents.Listen(panel, "toggle", func(js.Value, []js.Value) any {
		if panel.Get("open").Bool() {
			load()
		} else {
			scope.Cancel()
			content.Set("textContent", "")
		}
		return nil
	})
}

func (e *ItemEditor) renderLeaderDetails(content js.Value, details api.ParticipantDetails) {
	content.Set("textContent", "")
	if details.NeedsAnswers {
		viewHelpers.Feedback(content, "Required trip answers are missing.")
	}
	if details.Name == "" {
		viewHelpers.Feedback(content, "Participant details have not been supplied.")
		return
	}
	list := e.document.Call("createElement", "dl")
	add := func(label, value string) {
		if value == "" {
			return
		}
		name := e.document.Call("createElement", "dt")
		name.Set("textContent", label)
		description := e.document.Call("createElement", "dd")
		description.Set("textContent", value)
		list.Call("append", name, description)
	}
	add("Name", details.Name)
	add("Mobile phone", details.Mobile)
	add("Emergency contact", details.EmergencyName+" · "+details.EmergencyPhone)
	add("Dietary requirements", details.Dietary)
	add("Other information or messages", details.Notes)
	for _, question := range details.Questions {
		add(question.Label, displayAnswer(details.Answers[question.Key]))
	}
	content.Call("appendChild", list)
}

func displayAnswer(raw json.RawMessage) string {
	switch string(raw) {
	case "true":
		return "Yes"
	case "false":
		return "No"
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	var choices []string
	if json.Unmarshal(raw, &choices) == nil {
		return strings.Join(choices, ", ")
	}
	return ""
}
