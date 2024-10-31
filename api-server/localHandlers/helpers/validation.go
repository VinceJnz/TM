package helpers

import (
	"fmt"
	"time"
)

func ValidateDatesFromLtTo(FromDate, ToDate time.Time) error {
	if !FromDate.Before(ToDate) {
		return fmt.Errorf("dateError: From-date must be before To-date")
	}
	return nil
}
