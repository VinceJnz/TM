package httpProcessor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"time"
)

const debugTag = "httpProcessor."

// Client provides the connection to the rest interface (is used by a store to read/write/update data????)
type Client struct {
	//Ctx     context.Context
	BaseURL string
	//apiKey    string
	//User       *mdlUser.User
	//Session bool
	//CookieJar  *cookiejar.Jar
	HTTPClient *http.Client
}

type FieldNames map[string]string

type ReturnData struct {
	FieldNames FieldNames
}

func New(baseURL string) *Client {
	// Create a cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalf(debugTag+"New() Error creating cookie jar: %v", err)
	}

	httpClient := &http.Client{
		Jar: jar,
		//Timeout: time.Minute,
		Timeout: 5 * time.Second,
		//Transport: &http.Transport{
		//	TLSClientConfig: &tls.Config{
		//		//Certificates: []tls.Certificate{cert},
		//		//RootCAs:      caCertPool,
		//	},
		//},
	}

	return &Client{
		BaseURL: baseURL,
		//CookieJar:  jar,
		HTTPClient: httpClient,
	}
}

func (c *Client) NewRequest(method, url string, rxDataStru, txDataStru any, callBacks ...func(error, *ReturnData)) {
	err := c.newRequest(method, url, rxDataStru, txDataStru, callBacks...)
	if err != nil {
		log.Printf(debugTag+"NewRequest()1 err = %v", err)
	}
}

func (c *Client) newRequest(method, url string, rxDataStru, txDataStru any, callBacks ...func(error, *ReturnData)) error {
	var err error
	var req *http.Request
	var res *http.Response
	var fieldNames FieldNames

	url = c.BaseURL + url
	// Create a cookie jar
	//jar, _ := cookiejar.New(nil)

	//httpClient := &http.Client{
	//	Jar: jar,
	//	//Timeout: time.Minute,
	//	Timeout: 5 * time.Second,
	//	//Transport: &http.Transport{
	//	//	TLSClientConfig: &tls.Config{
	//	//		//Certificates: []tls.Certificate{cert},
	//	//		//RootCAs:      caCertPool,
	//	//	},
	//	//},
	//}

	callBackSuccess := func(error, *ReturnData) {
		err = fmt.Errorf(debugTag+"newRequest()2 INFORMATION: No success-callback has been provided: %w", err)
		log.Println(err, "req.URL =", req.URL) //This is the default returned if renderOk is called
	} //The function to be called to render the request results
	if len(callBacks) > 0 {
		if callBacks[0] != nil {
			callBackSuccess = callBacks[0]
		}
	}

	callBackFail := func(error, *ReturnData) {
		err = fmt.Errorf(debugTag+"newRequest()3 INFORMATION: No error-callback function has been provided: %w", err)
		log.Println(err, "req.URL =", req.URL) //This is the default returned if renderErr is called
	} //The function to be called to render an error
	if len(callBacks) > 1 {
		if callBacks[1] != nil {
			callBackFail = callBacks[1]
		}
	}

	switch method {
	case http.MethodDelete:
		itemJSON, err := json.Marshal(txDataStru)
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest().MethodDelete.2 failed to marshal item data: %w", err)
			return err
		}
		req, err = http.NewRequest(http.MethodDelete, url, bytes.NewReader(itemJSON))
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest().MethodDelete.3 failed to create request: %w", err)
			return err
		}
	case http.MethodGet:
		req, err = http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest().MethodGet.1 failed to create request: %w", err)
			return err
		}
	case http.MethodPut:
		itemJSON, err := json.Marshal(txDataStru)
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest().MethodPut.1 failed to marshal item data: %w", err)
			return err
		}
		req, err = http.NewRequest(http.MethodPut, url, bytes.NewBuffer(itemJSON))
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest().MethodPut.3 failed to create request: %w", err)
			return err
		}
		req.Header.Set("Content-Type", "application/json")
	case http.MethodPost:
		itemJSON, err := json.Marshal(txDataStru)
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest().MethodPost.1 failed to marshal item data: %w", err)
			return err
		}
		req, err = http.NewRequest(http.MethodPost, url, bytes.NewBuffer(itemJSON))
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest().MethodPost.3 failed to create request: %w", err)
			return err
		}
		req.Header.Set("Content-Type", "application/json")
	default:
		err = fmt.Errorf(debugTag+"NewRequest()1 invalid request type: %s", method)
		return err
	}

	//req.Header.Set("Authorization", "Bearer your_token_here")
	req.Header.Set("Access-Control-Allow-Credentials", "true")
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Origin", "http://localhost:8081") // Set the Origin header
	req.Header.Set("Origin", "https://localhost:8081") // Set the Origin header

	res, err = c.HTTPClient.Do(req) // This is the call to send the https request and receive the response
	if err != nil {
		err = fmt.Errorf(debugTag+"newRequest()4 from calling HTTPSClient.Do: %w", err)
		callBackFail(err, nil)
		return err
	}
	defer res.Body.Close()
	log.Printf("%v %v %v %v %v %v %v", debugTag+"NewRequest()4a ", "err =", err, "res.StatusCode =", res.StatusCode, "req.URL =", req.URL)

	//The following deals with http error responses
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		//The following decodes a json structured error message: Not currently used
		//var errRes errorResponse
		//if err = json.NewDecoder(res.Body).Decode(&errRes); err != nil { //Decoding happens here
		//	log.Printf("%v %v %v %v %v %v %+v %v %v %v %v", debugTag+"Client.SendRequest()6 ", "err =", err, "errRes =", errRes, "res =", res, "res.StatusCode =", res.StatusCode, "req.URL =", req.URL)
		//  callBackFail(fmt.Errorf("error6 from http response fail: %w", err))
		//  return
		//}
		//The following decodes a string error message
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			log.Printf("%v %v %v %v %v %v %v", debugTag+"NewRequest()6a ", "err =", err, "res.StatusCode =", res.StatusCode, "req.URL =", req.URL)
			err = fmt.Errorf(debugTag+"newRequest()6b server response StatusCode=%v: error=%w", res.StatusCode, err)
			callBackFail(err, nil)
		}
		err = fmt.Errorf(debugTag+"newRequest()7 server response StatusCode=%v: server message=%s", res.StatusCode, resBody)
		callBackFail(err, nil)
		return err
	}

	if rxDataStru != nil {
		//if err = json.NewDecoder(res.Body).Decode(&rxDataStru); err != nil { //This decodes the JSON data in the body and puts it in the supplied structure.
		//	resBody, _ := io.ReadAll(res.Body)
		//	log.Printf("%v %v %v %v %v %v %v %v %+v %v %v", debugTag+"NewRequest()8a ", "err =", err, "req =", req, "res.Body =", string(resBody), "rxDataStru =", rxDataStru, "req.URL =", req.URL)
		//	err = fmt.Errorf(debugTag+"newRequest()8b failed to decode JSON data: %w", err)
		//	callBackFail(err)
		//	return err
		//}
		fieldNames, err = decodeJSON(res, &rxDataStru)
		if err != nil { //This decodes the JSON data in the body and puts it in the supplied structure.
			resBody, _ := io.ReadAll(res.Body)
			log.Printf("%v %v %v %v %v %v %v %v %+v %v %v", debugTag+"NewRequest()8a ", "err =", err, "req =", req, "res.Body =", string(resBody), "rxDataStru =", rxDataStru, "req.URL =", req.URL)
			err = fmt.Errorf(debugTag+"newRequest()8b failed to decode JSON data: %w", err)
			callBackFail(err, nil)
			return err
		}
	} else {
		//Data struct is nil - this is not necesssarily an error, e.g. we might be deleting an item?????
		//Should the deleted item ID be returned???
		resBody, err := io.ReadAll(res.Body)
		if len(resBody) != 0 { // if the resBody in not empty then we should log it because there was no rxDataStru provided for the data in resBody
			log.Printf("%v %v %v %v %v %v %p %+v %v %+v %v %+v", debugTag+"NewRequest()9 - rx data structure is nil but the response body contains data ", "res.StatusCode =", res.StatusCode, "req.URL =", req.URL, "rxDataStru =", rxDataStru, rxDataStru, "resBody =", string(resBody), "err =", err)
		}
	}

	returnData := &ReturnData{FieldNames: fieldNames}
	callBackSuccess(nil, returnData)
	return nil
}

// decodeJSON decodes the JSON data in the body and puts it in the supplied structure, and returns the field names.
func decodeJSON(res *http.Response, rxDataStru interface{}) (FieldNames, error) {
	// Read the body into a byte slice
	//fieldNames := make(viewHelpers.FieldNames)
	//var fieldNames map[string]string
	fieldNames := make(FieldNames)

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return fieldNames, err
	}

	err = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&rxDataStru)
	if err != nil {
		fmt.Println("Error:", err)
		return fieldNames, err
	}

	//*
	// Decode the body into a map to get field names
	var result []map[string]interface{}
	err = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&result)
	if err != nil {
		log.Println(debugTag+"decodeJSON()3 Warning: failed to get field names, probably because there are none to retreive, Error:", err) // Don't return an error here. Just log it. The error is not fatal. The data has already been decoded and the field names are not critical.
	} else {
		record := result[0]
		for key := range record {
			fieldNames[key] = key
		}
	}

	return fieldNames, nil
}
