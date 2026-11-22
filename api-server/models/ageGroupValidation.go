package models

import (
	"strings"
	"time"
)

func AgeGroupMatches(birth time.Time, label string, now time.Time) bool {
	if birth.After(now) {
		return false
	}
	age := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		age--
	}
	bands := map[string][2]int{"infant": {0, 1}, "toddler": {1, 3}, "child": {3, 12}, "youth": {12, 18}, "adult": {18, 65}, "senior": {65, 150}, "life": {0, 150}}
	band, known := bands[strings.ToLower(strings.TrimSpace(label))]
	return !known || (age >= band[0] && age <= band[1])
}
