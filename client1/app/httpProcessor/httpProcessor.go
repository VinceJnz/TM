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
	CookieJar  *cookiejar.Jar
	HTTPClient *http.Client
}

func New(baseURL string) *Client {
	// Create a cookie jar
	jar, _ := cookiejar.New(nil)

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
		BaseURL:    baseURL,
		CookieJar:  jar,
		HTTPClient: httpClient,
	}
}

func (c *Client) NewRequest(method, url string, rxDataStru, txDataStru interface{}, callBacks ...func(error)) {
	c.newRequest(method, url, rxDataStru, txDataStru, callBacks...)
}

func (c *Client) newRequest(method, url string, rxDataStru, txDataStru interface{}, callBacks ...func(error)) (*http.Request, error) {
	var err error
	var req *http.Request
	var res *http.Response

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

	switch method {
	case http.MethodDelete:
		itemJSON, err := json.Marshal(txDataStru)
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest().MethodDelete.2 failed to marshal item data: %w", err)
			return nil, err
		}
		req, err = http.NewRequest(http.MethodDelete, url, bytes.NewReader(itemJSON))
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest().MethodDelete.3 failed to create request: %w", err)
			return nil, err
		}
	case http.MethodGet:
		req, err = http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest().MethodGet.1 failed to create request: %w", err)
			return nil, err
		}
	case http.MethodPut:
		itemJSON, err := json.Marshal(txDataStru)
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest().MethodPut.1 failed to marshal item data: %w", err)
			return nil, err
		}
		req, err = http.NewRequest("PUT", url, bytes.NewBuffer(itemJSON))
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest().MethodPut.3 failed to create request: %w", err)
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	case http.MethodPost:
		itemJSON, err := json.Marshal(txDataStru)
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest().MethodPost.1 failed to marshal item data: %w", err)
			return nil, err
		}
		req, err = http.NewRequest(http.MethodPost, url, bytes.NewBuffer(itemJSON))
		if err != nil {
			err = fmt.Errorf(debugTag+"NewRequest().MethodPost.3 failed to create request: %w", err)
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	default:
		err = fmt.Errorf(debugTag+"NewRequest()1 invalid request type: %s", method)
		return nil, err
	}

	callBackSuccess := func(error) {
		cookies := res.Cookies()
		log.Printf(debugTag+"NewRequest()0a Cookies: %v", c.CookieJar.Cookies(req.URL))
		if len(cookies) == 0 {
			log.Printf(debugTag + "NewRequest()0b there are no cookies")
		} else {
			log.Printf(debugTag + "NewRequest()0c checking cookie list...")
			for i, v := range cookies {
				log.Printf(debugTag+"NewRequest()0d cookie=%v, details=%+v", i, v)
			}
		}

		err = fmt.Errorf(debugTag+"newRequest()2a INFORMATION: Using default success-callback: %w", err)
		log.Println(err, "req.URL =", req.URL) //This is the default returned if renderOk is called
	} //The function to be called to render the request results
	if len(callBacks) > 0 {
		if callBacks[0] != nil {
			callBackSuccess = callBacks[0]
		}
	} else {
		err = fmt.Errorf(debugTag+"newRequest()2b INFORMATION: No success-callback has been provided, will use the default function: %w", err)
		log.Println(err, "req.URL =", req.URL)
	}

	callBackFail := func(error) {
		err = fmt.Errorf(debugTag+"newRequest()3 INFORMATION: No error-callback function has been provided: %w", err)
		log.Println(err, "req.URL =", req.URL) //This is the default returned if renderErr is called
	} //The function to be called to render an error
	if len(callBacks) > 1 {
		if callBacks[1] != nil {
			callBackFail = callBacks[1]
		}
	}

	//req.Header.Set("Authorization", "Bearer your_token_here")
	req.Header.Set("Access-Control-Allow-Credentials", "true")
	req.Header.Set("Content-Type", "application/json")

	res, err = c.HTTPClient.Do(req) // This is the call to send the https request and receive the response
	if err != nil {
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
			log.Printf("%v %v %v %v %v %v %v", debugTag+"NewRequest()6a ", "err =", err, "res.StatusCode =", res.StatusCode, "req.URL =", req.URL)
			err = fmt.Errorf(debugTag+"newRequest()6b server response StatusCode=%v: error=%w", res.StatusCode, err)
			callBackFail(err)
		}
		err = fmt.Errorf(debugTag+"newRequest()7 server response StatusCode=%v: server message=%s", res.StatusCode, resBody)
		callBackFail(err)
		return nil, err
	}

	if rxDataStru != nil {
		if err = json.NewDecoder(res.Body).Decode(&rxDataStru); err != nil { //This decodes the JSON data in the body and puts it in the supplied structure.
			resBody, _ := io.ReadAll(res.Body)
			log.Printf("%v %v %v %v %v %v %v %v %+v %v %v", debugTag+"NewRequest()8a ", "err =", err, "req =", req, "res.Body =", string(resBody), "rxDataStru =", rxDataStru, "req.URL =", req.URL)
			err = fmt.Errorf(debugTag+"newRequest()8b failed to decode JSON data: %w", err)
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

	cookies := res.Cookies()
	if len(cookies) == 0 {
		log.Printf(debugTag + "NewRequest()10 there are no cookies")
	} else {
		for i, v := range cookies {
			log.Printf(debugTag+"NewRequest()10 cookie=%v, details=%+v", i, v)
		}
	}

	callBackSuccess(nil)
	return req, nil
}
