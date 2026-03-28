package viewHelpers

import (
	"strings"
	"syscall/js"
	"unicode"
)

func SetStyleProperty(element js.Value, property, setting string) {
	element.Get("style").Call("setProperty", normalizeCSSProperty(property), setting)
}

// SetStyles applies multiple CSS properties to an element.
func SetStyles(element js.Value, styles map[string]string) {
	for property, setting := range styles {
		SetStyleProperty(element, property, setting)
	}
}

func normalizeCSSProperty(property string) string {
	if property == "" {
		return property
	}
	if strings.Contains(property, "-") {
		return strings.ToLower(property)
	}

	var builder strings.Builder
	for index, char := range property {
		if unicode.IsUpper(char) {
			if index > 0 {
				builder.WriteByte('-')
			}
			builder.WriteRune(unicode.ToLower(char))
			continue
		}
		builder.WriteRune(char)
	}

	return builder.String()
}
