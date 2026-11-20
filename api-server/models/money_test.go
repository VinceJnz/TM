package models

import (
	"encoding/json"
	"testing"
)

func TestMoneyNumericContractAndPrecision(t *testing.T) {
	for _, input := range []string{`0.29`, `"0.29"`} {
		var m Money
		if err := json.Unmarshal([]byte(input), &m); err != nil {
			t.Fatal(err)
		}
		out, err := json.Marshal(m)
		if err != nil || string(out) != "0.29" {
			t.Fatal(string(out), err)
		}
	}
	for _, input := range []string{`-1`, `0.001`, `"NaN"`} {
		var m Money
		if json.Unmarshal([]byte(input), &m) == nil {
			t.Fatal("invalid amount accepted", input)
		}
	}
}
