package viewHelpers

import (
	"slices"
	"strings"
)

// AlphabeticalChoices leaves the source records and the order of equal labels intact.
func AlphabeticalChoices[T any](items []T, label func(T) string) []T {
	ordered := slices.Clone(items)
	slices.SortStableFunc(ordered, func(a, b T) int {
		return strings.Compare(strings.ToLower(label(a)), strings.ToLower(label(b)))
	})
	return ordered
}
