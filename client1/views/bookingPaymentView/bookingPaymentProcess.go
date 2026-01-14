package bookingPaymentView

/*

package storeBookingReport

import (
	"errors"
	"log"
	"strconv"
)

//MakePayment tells the server to contact the payment gateway with the information for setting up the payment
//and return the payment page url to the callback function.
func (s *Store) MakePayment(BookingID int64, callback func(string, error)) error { //int64 {
	var paymentURL string
	callbackSuccess := func(errIn error) {
		log.Printf("%v %v %v", debugTag+"Store.MakePayment()1", "paymentURL  =", paymentURL)
		if callback != nil {
			callback(paymentURL, errIn)
		} else {
			log.Println(debugTag + "MakePayment: callback is nil")
		}
	}

	callbackFail := func(errIn error) {
		log.Printf("%v %v %v %v %v", debugTag+"Store.MakePayment()4 ****Make Payment failed****", "errIn =", errIn, "len(Items) =", len(s.Items))
	}

	err := s.Client.SendGetRequest("checkoutSession/create/"+strconv.FormatInt(BookingID, 10), &paymentURL, callbackSuccess, callbackFail) //Send the REST request
	if err != nil {
		log.Printf("%v %v %v", debugTag+"Store.MakePayment()2 ", "err =", err) //Log the error in the browser
		return errors.New(debugTag + "MakePayment: what is the error description?????")
	}
	return nil
}

//ClosePayment
func (s *Store) ClosePayment(BookingID int64, callback func(error)) error { //int64 {
	callbackSuccess := func(errIn error) {
		log.Printf("%v", debugTag+"Store.ClosePayment()1")
		if callback != nil {
			callback(errIn)
		} else {
			log.Println(debugTag + "ClosePayment: callback is nil")
		}
	}

	callbackFail := func(errIn error) {
		log.Printf("%v %v %v %v %v", debugTag+"Store.MakePayment()4 ****Close Payment failed****", "errIn =", errIn, "len(Items) =", len(s.Items))
	}

	err := s.Client.SendGetRequest("checkoutSession/closed/"+strconv.FormatInt(BookingID, 10), nil, callbackSuccess, callbackFail) //Send the REST request
	if err != nil {
		log.Printf("%v %v %v", debugTag+"Store.ClosePayment()2 ", "err =", err) //Log the error in the browser
		return errors.New(debugTag + "ClosePayment: what is the error description?????")
	}
	return nil
}

//CheckPayment
func (s *Store) CheckPayment(BookingID int64, callback func(error)) error { //int64 {
	var response string
	callbackSuccess := func(errIn error) {
		log.Printf("%v", debugTag+"Store.CheckPayment()1")
		if callback != nil {
			callback(errors.New(response))
		} else {
			log.Println(debugTag + "CheckPayment: callback is nil")
		}
	}

	callbackFail := func(errIn error) {
		log.Printf("%v %v %v %v %v", debugTag+"Store.CheckPayment()4 ****Close Payment failed****", "errIn =", errIn, "len(Items) =", len(s.Items))
	}

	err := s.Client.SendGetRequest("checkoutSession/check/"+strconv.FormatInt(BookingID, 10), &response, callbackSuccess, callbackFail) //Send the REST request
	if err != nil {
		log.Printf("%v %v %v", debugTag+"Store.CheckPayment()2 ", "err =", err) //Log the error in the browser
		return errors.New(debugTag + "ClosePayment: what is the error description?????")
	}
	return nil
}



// func (c *Client) SendGetRequest(url string, dataStru interface{}, callBacks ...func(error)) error {
*/
