package viewHelpers

import (
	"log"
	"syscall/js"
	"time"
)

type DateName int

const (
	DateNameFrom DateName = iota
	DateNameTo
)

func ValidateDatesFromLtTo(dateName DateName, fromDateObj, toDateObj js.Value) {
	from := fromDateObj.Get("value").String()
	FromDate, err := time.Parse(Layout, from)
	if err != nil {
		log.Println("Error parsing from_date:", err)
		return
	}

	to := toDateObj.Get("value").String()
	ToDate, err := time.Parse(Layout, to)
	if err != nil {
		log.Println("Error parsing to_date:", err)
		return
	}

	fromDateObj.Call("setCustomValidity", "")
	toDateObj.Call("setCustomValidity", "")
	switch dateName {
	case DateNameFrom:
		if !FromDate.Before(ToDate) {
			fromDateObj.Call("setCustomValidity", "From-date must be before To-date")
		}
	case DateNameTo:
		if !FromDate.Before(ToDate) {
			toDateObj.Call("setCustomValidity", "To-date must be after From-date")
		}
	}
}

func ValidateNewPassword(passwordObj, passwordChkObj js.Value) {
	password := passwordObj.Get("value").String()
	passwordChk := passwordChkObj.Get("value").String()

	//passwordObj.Call("setCustomValidity", "")
	passwordChkObj.Call("setCustomValidity", "")
	if password != passwordChk {
		//passwordObj.Call("setCustomValidity", "Passwords do not match")
		passwordChkObj.Call("setCustomValidity", "Passwords do not match")
	}
}
