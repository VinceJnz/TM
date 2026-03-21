package helpers

import (
	"fmt"
	"regexp"
	"time"
)

func ValidateDatesFromLtTo(FromDate, ToDate time.Time) error {
	if FromDate.Compare(ToDate) > 0 {
		return fmt.Errorf("dateError: From-date must be equal to or before To-date")
	}
	return nil
}

// IsValidEmail validates email format using a basic regex pattern.
// Returns true if the email matches the expected format.
func IsValidEmail(email string) bool {
	// Basic email validation regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}
