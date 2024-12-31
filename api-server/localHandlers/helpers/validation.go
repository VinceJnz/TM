package helpers

import (
	"fmt"
	"time"
)

func ValidateDatesFromLtTo(FromDate, ToDate time.Time) error {
	if FromDate.Compare(ToDate) > 0 {
		return fmt.Errorf("dateError: From-date must be equal to or before To-date")
	}
	return nil
}
