package viewHelpers

import (
	"errors"
	"log"
	"syscall/js"
	"time"
)

type DateName int

const (
	DateNameFrom DateName = iota
	DateNameTo
)

func ValidateDatesFromLtTo(fromDateObj, toDateObj, msgObj js.Value, warningMsg string) error {
	from := fromDateObj.Get("value").String()
	FromDate, err := time.Parse(Layout, from)
	if err != nil {
		log.Println("Error parsing from_date:", err)
		return err
	}

	to := toDateObj.Get("value").String()
	ToDate, err := time.Parse(Layout, to)
	if err != nil {
		log.Println("Error parsing to_date:", err)
		return err
	}

	if warningMsg == "" {
		log.Println("warning message not set")
		return errors.New("warning message not set")
	}

	log.Printf(debugTag+"ValidateDatesFromLtTo()1 fromDateObj=%v, toDateObj=%v, msgObj=%v, warningMsg=%v", fromDateObj.Get("id"), toDateObj.Get("id"), msgObj.Get("id"), warningMsg)
	fromDateObj.Call("setCustomValidity", "")
	toDateObj.Call("setCustomValidity", "")
	if !FromDate.Before(ToDate) {
		msgObj.Call("setCustomValidity", warningMsg)
		log.Printf(debugTag+"ValidateDatesFromLtTo()2 fromDateObj=%v, toDateObj=%v, msgObj=%v, warningMsg=%v", fromDateObj.Get("id"), toDateObj.Get("id"), msgObj.Get("id"), warningMsg)
		return errors.New(warningMsg)
	}
	return nil
}

func ValidateDatesFromLtTo2(dateName DateName, fromDateObj, toDateObj js.Value) {
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
