package viewHelpers

import (
	"slices"
	"testing"
)

func TestAlphabeticalChoices(t *testing.T) {
	type choice struct {
		id    int
		label string
	}
	items := []choice{{3, "users"}, {2, "Booking Status"}, {0, "active"}, {1, "Active"}}
	original := slices.Clone(items)
	got := AlphabeticalChoices(items, func(item choice) string { return item.label })
	want := []choice{items[2], items[3], items[1], items[0]}
	if !slices.Equal(got, want) {
		t.Fatalf("choices = %v, want %v", got, want)
	}
	if !slices.Equal(items, original) {
		t.Fatalf("source choices changed: %v", items)
	}
}
