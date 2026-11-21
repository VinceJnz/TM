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
