// Package fieldrules shares simple field constraints between services and the
// API contract generator. Ownership and cross-record rules belong in services.
package fieldrules

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Rule struct {
	Name        string
	Required    bool
	Constraints map[string]int
}

func Fields(t reflect.Type) []Rule {
	var rules []Rule
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("rule")
		if tag == "" {
			continue
		}
		rule := Rule{Name: strings.Split(field.Tag.Get("json"), ",")[0], Constraints: map[string]int{}}
		for _, part := range strings.Split(tag, ",") {
			if part == "required" {
				rule.Required = true
				continue
			}
			pieces := strings.SplitN(part, "=", 2)
			if len(pieces) != 2 {
				panic("invalid field rule: " + part)
			}
			switch pieces[0] {
			case "minimum", "minLength", "maxLength":
			default:
				panic("unsupported field rule: " + part)
			}
			value, err := strconv.Atoi(pieces[1])
			if err != nil {
				panic("invalid field rule: " + part)
			}
			rule.Constraints[pieces[0]] = value
		}
		rules = append(rules, rule)
	}
	return rules
}
func Validate(input any) error {
	value := reflect.ValueOf(input)
	t := value.Type()
	rules := Fields(t)
	for _, rule := range rules {
		var field reflect.Value
		for i := 0; i < t.NumField(); i++ {
			if strings.Split(t.Field(i).Tag.Get("json"), ",")[0] == rule.Name {
				field = value.Field(i)
				break
			}
		}
		if rule.Required && (field.IsZero() || (field.Kind() == reflect.String && strings.TrimSpace(field.String()) == "")) {
			return fmt.Errorf("%s is required", rule.Name)
		}
		for constraint, limit := range rule.Constraints {
			switch constraint {
			case "minimum":
				if field.Int() < int64(limit) {
					return fmt.Errorf("%s is too small", rule.Name)
				}
			case "minLength":
				if utf8.RuneCountInString(field.String()) < limit {
					return fmt.Errorf("%s is too short", rule.Name)
				}
			case "maxLength":
				if utf8.RuneCountInString(field.String()) > limit {
					return fmt.Errorf("%s is too long", rule.Name)
				}
			}
		}
	}
	return nil
}
