package fieldrules

import (
	"reflect"
	"testing"
)

func TestConstraintsValidateTheDocumentedRules(t *testing.T) {
	type payload struct {
		ID   int    `json:"id" rule:"required,minimum=1"`
		Name string `json:"name" rule:"required,minLength=1,maxLength=2"`
	}
	rules := Fields(reflect.TypeOf(payload{}))
	if len(rules) != 2 || !rules[0].Required || rules[0].Constraints["minimum"] != 1 || rules[1].Constraints["maxLength"] != 2 {
		t.Fatal(rules)
	}
	for _, v := range []payload{{0, "A"}, {1, " "}, {1, "abc"}} {
		if Validate(v) == nil {
			t.Fatal(v)
		}
	}
	if err := Validate(payload{1, "éé"}); err != nil {
		t.Fatal("length must count characters", err)
	}
}
func TestInvalidDefinitionsReturnErrorsWithoutPanicking(t *testing.T) {
	for _, v := range []any{
		struct {
			Name string `json:"name" rule:"minimum=1"`
		}{},
		struct {
			ID int `json:"id" rule:"maxLength=1"`
		}{},
		struct {
			Name string `json:"name" rule:"unsupported=1"`
		}{},
		struct {
			Name string `json:"name" rule:"minLength=3,maxLength=1"`
		}{},
		struct {
			Name string `json:"name" rule:"maxLength=-1"`
		}{}, nil, 42,
	} {
		if err := Validate(v); err == nil {
			t.Fatal("invalid definition accepted")
		}
	}
	type valid struct {
		Name string `json:"name" rule:"maxLength=2"`
	}
	metadata := Fields(reflect.TypeOf(valid{}))
	metadata[0].Constraints["maxLength"] = 100
	if err := Validate(valid{"long"}); err == nil {
		t.Fatal("metadata caller changed cached validation")
	}
}
