package appearance

import (
	"strings"
	"testing"
)

func TestValidateTheme(t *testing.T) {
	cases := []struct {
		name  string
		theme Theme
		valid bool
	}{
		{"defaults", Theme{}, true},
		{"custom", Theme{Name: "Club", Primary: "#166534", Font: "serif"}, true},
		{"low contrast", Theme{Primary: "#ffff00"}, false},
		{"invalid color", Theme{Primary: "#123456;display:none"}, false},
		{"invalid font", Theme{Font: "custom"}, false},
		{"invalid upload", Theme{Logo: "c2NyaXB0"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.theme.Validate(); (err == nil) != tc.valid {
				t.Fatalf("Validate() = %v", err)
			}
		})
	}
}

func TestThemeCSS(t *testing.T) {
	css := (Theme{Primary: "#166534", Font: "serif"}).CSS()
	for _, expected := range []string{"--tm-primary:#166534", "--tm-font:Georgia", "--tm-selected-bg:color-mix"} {
		if !strings.Contains(css, expected) {
			t.Errorf("missing %s", expected)
		}
	}
}
