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

//type FieldNames map[string]string

//func ExtractFieldNameData(data ...any) FieldNames {
//	for _, v := range data {
//		switch x := v.(type) {
//		case FieldNames: // Field names
//			log.Printf(debugTag+"ExtractFieldNameData()1 fieldNames=%+v", x)
//			return x
//		}
//	}
//	return nil
//}

func ValidateDatesFromLtTo(fromDateObj, toDateObj, msgObj js.Value, warningMsg string) error {
	from := fromDateObj.Get("value").String()
	FromDate, err := time.Parse(DateLayout, from)
	if err != nil {
		log.Println("Error parsing from_date:", err)
		return err
	}

	to := toDateObj.Get("value").String()
	ToDate, err := time.Parse(DateLayout, to)
	if err != nil {
		log.Println("Error parsing to_date:", err)
		return err
	}

	if warningMsg == "" {
		log.Println("warning message not set")
		return errors.New("warning message not set")
	}
	//log.Printf(debugTag+"ValidateDatesFromLtTo()1 from=%v, to=%v, FromDate=%v, ToDate=%v, compare=%v", from, to, FromDate, ToDate, FromDate.Compare(ToDate))
	//log.Printf(debugTag+"ValidateDatesFromLtTo()2 fromDateObj=%v, toDateObj=%v, msgObj=%v, warningMsg=%v", fromDateObj.Get("id"), toDateObj.Get("id"), msgObj.Get("id"), warningMsg)
	fromDateObj.Call("setCustomValidity", "")
	toDateObj.Call("setCustomValidity", "")
	if FromDate.Compare(ToDate) > 0 { //!FromDate.Before(ToDate) {
		log.Printf(debugTag+"ValidateDatesFromLtTo()3 fromDateObj=%v, toDateObj=%v, msgObj=%v, warningMsg=%v", fromDateObj.Get("id"), toDateObj.Get("id"), msgObj.Get("id"), warningMsg)
		msgObj.Call("setCustomValidity", warningMsg)
		//return errors.New(warningMsg) // This seems to cause the js to hang
	} else {
		msgObj.Call("setCustomValidity", "") // Not sure if this is needed????
	}

	return nil
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
