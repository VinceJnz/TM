// Package fieldrules shares field constraints between services and contracts.
// Ownership and cross-record rules belong in services.
package fieldrules

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

type Rule struct {
	Name        string
	Required    bool
	Constraints map[string]int
}
type compiledRule struct {
	Rule
	index int
}
type definition struct {
	rules []compiledRule
	err   error
}

var definitions sync.Map

func compile(t reflect.Type) definition {
	if t == nil || t.Kind() != reflect.Struct {
		return definition{err: fmt.Errorf("field rules require a struct")}
	}
	if cached, ok := definitions.Load(t); ok {
		return cached.(definition)
	}
	result := definition{}
	seen := map[string]bool{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("rule")
		if tag == "" {
			continue
		}
		r := compiledRule{Rule: Rule{Name: strings.Split(f.Tag.Get("json"), ",")[0], Constraints: map[string]int{}}, index: i}
		fail := func() definition {
			result.err = fmt.Errorf("invalid field rule definition: %s.%s", t.Name(), f.Name)
			definitions.Store(t, result)
			return result
		}
		if f.PkgPath != "" || r.Name == "" || r.Name == "-" || seen[r.Name] {
			return fail()
		}
		seen[r.Name] = true
		for _, part := range strings.Split(tag, ",") {
			if part == "required" {
				r.Required = true
				continue
			}
			pieces := strings.SplitN(part, "=", 2)
			if len(pieces) != 2 {
				return fail()
			}
			n, err := strconv.Atoi(pieces[1])
			if err != nil || n < 0 {
				return fail()
			}
			switch pieces[0] {
			case "minimum":
				if f.Type.Kind() < reflect.Int || f.Type.Kind() > reflect.Int64 {
					return fail()
				}
			case "minLength", "maxLength":
				if f.Type.Kind() != reflect.String {
					return fail()
				}
			default:
				return fail()
			}
			if _, exists := r.Constraints[pieces[0]]; exists {
				return fail()
			}
			r.Constraints[pieces[0]] = n
		}
		if maximum, ok := r.Constraints["maxLength"]; ok && r.Constraints["minLength"] > maximum {
			return fail()
		}
		result.rules = append(result.rules, r)
	}
	definitions.Store(t, result)
	return result
}

// ValidateType is used at startup and by generation, before serving requests.
func ValidateType(t reflect.Type) error { return compile(t).err }
func Fields(t reflect.Type) []Rule {
	compiled := compile(t)
	if compiled.err != nil {
		return nil
	}
	result := []Rule{}
	for _, rule := range compiled.rules {
		copy := rule.Rule
		copy.Constraints = map[string]int{}
		for k, v := range rule.Constraints {
			copy.Constraints[k] = v
		}
		result = append(result, copy)
	}
	return result
}
func Validate(input any) error {
	value := reflect.ValueOf(input)
	if !value.IsValid() {
		return fmt.Errorf("field rules require a struct")
	}
	compiled := compile(value.Type())
	if compiled.err != nil {
		return compiled.err
	}
	for _, rule := range compiled.rules {
		field := value.Field(rule.index)
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
