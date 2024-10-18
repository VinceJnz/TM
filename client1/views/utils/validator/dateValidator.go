package validator

import (
	"syscall/js"
	"time"
)

func MainDemo() {
	doc := js.Global().Get("document")

	form := doc.Call("getElementById", "my-form")
	//submitButton := doc.Call("getElementById", "submit-btn")

	// Add event listener for form submission
	form.Call("addEventListener", "submit", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]

		// Prevent default form submission behavior
		event.Call("preventDefault")

		// Validate dates
		fromDate := doc.Call("getElementById", "fromDate").Get("value").String()
		toDate := doc.Call("getElementById", "toDate").Get("value").String()

		if !ValidateDates(fromDate, toDate) {
			// Show error if validation fails
			doc.Call("getElementById", "toDate-error").Set("innerHTML", "To Date must be greater than or equal to From Date")
			return nil
		}

		// Clear errors if validation passes
		ClearErrors2()

		// Proceed with form submission or other actions
		ProcessFormData()

		return nil
	}))
}

// validateDates compares the fromDate and toDate and returns true if toDate >= fromDate
func ValidateDates(fromDate, toDate string) bool {
	from, err1 := time.Parse("2006-01-02", fromDate)
	to, err2 := time.Parse("2006-01-02", toDate)

	if err1 != nil || err2 != nil {
		// Invalid date formats, handle error if needed
		return false
	}

	return !to.Before(from) // toDate must be greater than or equal to fromDate
}

// clearErrors clears any date validation error messages
func ClearErrors2() {
	doc := js.Global().Get("document")
	doc.Call("getElementById", "toDate-error").Set("innerHTML", "")
}

// processFormData handles the form submission logic
func ProcessFormData() {
	println("Form submitted successfully!")
	// Add any form data handling logic here (e.g., sending via API)
}
