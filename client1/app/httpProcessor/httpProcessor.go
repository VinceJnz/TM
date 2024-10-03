package httpProcessor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const debugTag = "restProcessor."

// Client provides the connection to the rest interface (is used by a store to read/write/update data????)
type Client struct {
	Ctx     context.Context
	BaseURL string
	//apiKey    string
	//User       *mdlUser.User
	Session bool
	//CookieJar  http.CookieJar
	HTTPSClient *http.Client
}

// New uses basic authentication instead of an apiKey.
// baseURL is the URL for the Rest API, e.g. https://localhost:8080/api/v1
func New(ctx context.Context, baseURL string) *Client {
	return &Client{
		Ctx: ctx,
		//Ctx:     context.TODO(),
		BaseURL: baseURL,
		//apiKey:  apiKey,
		//User: mdlUser.User{Username: username, Password: password},
		HTTPSClient: &http.Client{
			//Jar: jar,
			//Timeout: time.Minute,
			Timeout: 5 * time.Second,
			//Transport: &http.Transport{
			//	TLSClientConfig: &tls.Config{
			//		//Certificates: []tls.Certificate{cert},
			//		//RootCAs:      caCertPool,
			//	},
			//},
		},
	}
	//*/
	//return nil
}

func NewRequest(method, url string, rxDataStru, txDataStru interface{}, callBacks ...func(error)) (*http.Request, error) {
	var err error
	var req *http.Request

	httpClient := &http.Client{
		//Jar: jar,
		//Timeout: time.Minute,
		Timeout: 5 * time.Second,
		//Transport: &http.Transport{
		//	TLSClientConfig: &tls.Config{
		//		//Certificates: []tls.Certificate{cert},
		//		//RootCAs:      caCertPool,
		//	},
		//},
	}

	switch method {
	case http.MethodDelete:
		itemJSON, err := json.Marshal(txDataStru)
		if err != nil {
			log.Printf("%v %v %v %v %v %v %+v %v %v", debugTag+"NewRequest()1 ", "err =", err, "url =", url, "txDataStru =", txDataStru, "itemJSON =", itemJSON)
			err = fmt.Errorf(debugTag+"NewRequest()1 failed to marshal item data: %w", err)
			return nil, err
		}
		req, err = http.NewRequest(http.MethodDelete, url, bytes.NewReader(itemJSON))
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest()1 failed to create request: %w", err)
			return nil, err
		}
	case http.MethodGet:
		req, err = http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest()1 failed to create request: %w", err)
			return nil, err
		}
	case http.MethodPut:
		itemJSON, err := json.Marshal(txDataStru)
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest()1 failed to marshal item data: %w", err)
			return nil, err
		}
		req, err := http.NewRequest("PUT", url, bytes.NewBuffer(itemJSON))
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest()1 failed to create request: %w", err)
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	case http.MethodPost:
		itemJSON, err := json.Marshal(txDataStru)
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest()1 failed to marshal item data: %w", err)
			return nil, err
		}
		req, err = http.NewRequest(http.MethodPost, url, bytes.NewBuffer(itemJSON))
		if err != nil {
			log.Printf("%v %v %v %v %v %v %+v %v %v", debugTag+"NewRequest()2 ", "err =", err, "url =", url, "txDataStru =", txDataStru, "itemJSON =", itemJSON)
			err = fmt.Errorf(debugTag+"NewRequest()1 failed to create request: %w", err)
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	default:
		err = fmt.Errorf(debugTag+"NewRequest()1 invalid request type: %s", method)
		return nil, err
	}

	callBackSuccess := func(error) {
		err = fmt.Errorf(debugTag+"newRequest()2 WARNING: No success-callback has been provided (e.g. render function): %w", err)
		log.Println(err, "req.URL =", req.URL) //This is the default returned if renderOk is called
	} //The function to be called to render the request results
	if len(callBacks) > 0 {
		if callBacks[0] != nil {
			callBackSuccess = callBacks[0]
		}
	} else {
		err = fmt.Errorf(debugTag+"newRequest()2 INFORMATION: No success-callback has been provided (e.g. render function): %w", err)
		log.Println(err, "req.URL =", req.URL)
	}

	callBackFail := func(error) {
		err = fmt.Errorf(debugTag+"newRequest()3 No error function has been provided: %w", err)
		log.Println(err, "req.URL =", req.URL) //This is the default returned if renderErr is called
	} //The function to be called to render an error
	if len(callBacks) > 1 {
		if callBacks[1] != nil {
			callBackFail = callBacks[1]
		}
	}

	res, err := httpClient.Do(req) // This is the call to send the https request and receive the response
	if err != nil {
		log.Printf("%v %v %v %v %v %v %v %v %+v", debugTag+"NewRequest()4 ", "err =", err, "res =", res, "req.URL =", req.URL, "req =", req)
		err = fmt.Errorf(debugTag+"newRequest()4 from calling HTTPSClient.Do: %w", err)
		callBackFail(err)
		return nil, err
	}
	defer res.Body.Close()

	//The following deals with http error responses
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		//The following decodes a json structured error message: Not currently used
		//var errRes errorResponse
		//if err = json.NewDecoder(res.Body).Decode(&errRes); err != nil { //Decoding happens here ???
		//	log.Printf("%v %v %v %v %v %v %+v %v %v %v %v", debugTag+"Client.SendRequest()6 ", "err =", err, "errRes =", errRes, "res =", res, "res.StatusCode =", res.StatusCode, "req.URL =", req.URL)
		//callBackFail(fmt.Errorf("error6 from http response fail: %w", err))
		//return
		//}
		//The following decodes a string error message
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			log.Printf("%v %v %v %v %v %v %v", debugTag+"NewRequest()6 ", "err =", err, "res.StatusCode =", res.StatusCode, "req.URL =", req.URL)
			err = fmt.Errorf(debugTag+"newRequest()6 server response StatusCode=%v: error=%w", res.StatusCode, err)
			callBackFail(err)
		}
		err = fmt.Errorf(debugTag+"newRequest()7 server response StatusCode=%v: server message=%s", res.StatusCode, resBody)
		callBackFail(err)
		return nil, err
	}

	if rxDataStru != nil {
		if err = json.NewDecoder(res.Body).Decode(&rxDataStru); err != nil { //This decodes the JSON data in the body and puts it in the supplied structure.
			resBody, _ := io.ReadAll(res.Body)
			log.Printf("%v %v %v %v %v %v %v %v %+v %v %v", debugTag+"NewRequest()8 ", "err =", err, "req =", req, "res.Body =", string(resBody), "rxDataStru =", rxDataStru, "req.URL =", req.URL)
			err = fmt.Errorf(debugTag+"newRequest()8 failed to decode JSON data: %w", err)
			callBackFail(err)
			return nil, err
		}
	} else {
		//Data struct is nil - this is not necesssarily an error, e.g. we might be deleting an item?????
		//Should the deleted item ID be returned???
		//log.Printf("%v %v %v %v %p %+v", debugTag+"Client.SendRequest()8c ", "req.URL =", req.URL, "dataStru =", dataStru, dataStru)
		resBody, err := io.ReadAll(res.Body)
		log.Printf("%v %v %v %v %p %+v %v %+v %v %+v", debugTag+"NewRequest()9 - data is nil ", "req.URL =", req.URL, "rxDataStru =", rxDataStru, rxDataStru, "resBody =", string(resBody), "err =", err)
	}

	callBackSuccess(nil)
	return req, nil
}
