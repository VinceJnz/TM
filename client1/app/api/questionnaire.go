package api

import "encoding/json"

type Question struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
}

type Questionnaire struct {
	Revision  int        `json:"revision"`
	Questions []Question `json:"questions"`
}

type ParticipantDetails struct {
	Name                  string                     `json:"submitted_name"`
	Mobile                string                     `json:"mobile_phone"`
	Address               string                     `json:"home_address"`
	Membership            string                     `json:"membership_category"`
	EmergencyName         string                     `json:"emergency_contact_name"`
	EmergencyPhone        string                     `json:"emergency_contact_phone"`
	Dietary               string                     `json:"dietary_requirements"`
	Notes                 string                     `json:"notes,omitempty"`
	QuestionnaireRevision int                        `json:"questionnaire_revision"`
	Answers               map[string]json.RawMessage `json:"trip_answers"`
	Questions             []Question                 `json:"answered_questions,omitempty"`
	NeedsAnswers          bool                       `json:"needs_answers,omitempty"`
}
