package models

import (
	"encoding/json"
	"strings"
)

type ParticipantWrite struct {
	BookingID int                 `json:"booking_id"`
	PersonID  int                 `json:"person_id"`
	Notes     string              `json:"notes"`
	Details   *ParticipantDetails `json:"details,omitempty"`
}

type ParticipantDetails struct {
	Name                  string                     `json:"submitted_name" db:"submitted_name"`
	Mobile                string                     `json:"mobile_phone" db:"mobile_phone"`
	Address               string                     `json:"home_address" db:"home_address"`
	Membership            string                     `json:"membership_category" db:"membership_category"`
	EmergencyName         string                     `json:"emergency_contact_name" db:"emergency_contact_name"`
	EmergencyPhone        string                     `json:"emergency_contact_phone" db:"emergency_contact_phone"`
	Dietary               string                     `json:"dietary_requirements" db:"dietary_requirements"`
	Notes                 string                     `json:"notes,omitempty" db:"notes"`
	QuestionnaireRevision int                        `json:"questionnaire_revision" db:"questionnaire_revision"`
	Answers               map[string]json.RawMessage `json:"trip_answers" db:"-"`
	Questions             []Question                 `json:"answered_questions,omitempty" db:"-"`
	NeedsAnswers          bool                       `json:"needs_answers,omitempty" db:"-"`
}

func (d ParticipantDetails) Valid() bool {
	for _, value := range []string{d.Name, d.Mobile, d.Address, d.EmergencyName, d.EmergencyPhone, d.Dietary} {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	return len(d.Name) <= 255 && len(d.Mobile) <= 50 && len(d.Address) <= 500 &&
		len(d.EmergencyName) <= 255 && len(d.EmergencyPhone) <= 50 && len(d.Dietary) <= 2000 &&
		(d.Membership == "member" || d.Membership == "non_member" || d.Membership == "affiliated_member")
}
