package models

import (
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"strings"
)

var ErrQuestionnaire = errors.New("invalid questionnaire")

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

var questionKey = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func (q Questionnaire) Validate() error {
	if q.Revision < 0 || q.Questions == nil || len(q.Questions) > 32 {
		return ErrQuestionnaire
	}
	keys := make(map[string]bool, len(q.Questions))
	for _, question := range q.Questions {
		if keys[question.Key] || !questionKey.MatchString(question.Key) ||
			strings.TrimSpace(question.Label) == "" || len(question.Label) > 255 {
			return ErrQuestionnaire
		}
		keys[question.Key] = true
		switch question.Type {
		case "yes_no", "text":
			if len(question.Options) != 0 {
				return ErrQuestionnaire
			}
		case "single_choice", "multiple_choice":
			if len(question.Options) == 0 || len(question.Options) > 20 {
				return ErrQuestionnaire
			}
			options := make(map[string]bool, len(question.Options))
			for _, value := range question.Options {
				if options[value] || strings.TrimSpace(value) == "" || len(value) > 120 {
					return ErrQuestionnaire
				}
				options[value] = true
			}
		default:
			return ErrQuestionnaire
		}
	}
	return nil
}

func (q Questionnaire) ValidateAnswers(answers map[string]json.RawMessage, complete bool) error {
	accepted := 0
	for _, question := range q.Questions {
		answer, found := answers[question.Key]
		if !found {
			if complete && question.Required {
				return ErrQuestionnaire
			}
			continue
		}
		if string(answer) == "null" || !question.validAnswer(answer) {
			return ErrQuestionnaire
		}
		accepted++
	}
	if accepted != len(answers) {
		return ErrQuestionnaire
	}
	return nil
}

func (q Question) validAnswer(answer json.RawMessage) bool {
	if string(answer) == "null" {
		return false
	}
	switch q.Type {
	case "yes_no":
		var value bool
		return json.Unmarshal(answer, &value) == nil
	case "text":
		var value string
		return json.Unmarshal(answer, &value) == nil && strings.TrimSpace(value) != "" && len(value) <= 4000
	case "single_choice":
		var value string
		return json.Unmarshal(answer, &value) == nil && slices.Contains(q.Options, value)
	case "multiple_choice":
		var values []string
		if json.Unmarshal(answer, &values) != nil || len(values) == 0 || len(values) > len(q.Options) {
			return false
		}
		seen := make(map[string]bool, len(values))
		for _, value := range values {
			if seen[value] || !slices.Contains(q.Options, value) {
				return false
			}
			seen[value] = true
		}
		return true
	}
	return false
}

func (q Questionnaire) NeedsAnswers(answers map[string]json.RawMessage) bool {
	for _, question := range q.Questions {
		if question.Required && !question.validAnswer(answers[question.Key]) {
			return true
		}
	}
	return false
}
