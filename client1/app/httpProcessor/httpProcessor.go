package httpProcessor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"syscall/js"
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

type FieldNames map[string]string // This needs to be removed it is not needed ???

type ReturnData struct {
	FieldNames FieldNames // This needs to be removed it is not needed ??? // Iy sould br delt with by having different api calls for different data views
}

func New(baseURL string) *Client {
	// Create a cookie jar
	//jar, err := cookiejar.New(nil)
	//if err != nil {
	//	log.Fatalf(debugTag+"New() Error creating cookie jar: %v", err)
	//}

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

	return &Client{
		BaseURL: baseURL,
		//CookieJar:  jar,
		HTTPClient: httpClient,
	}
}

func (c *Client) NewRequest(method, url string, rxDataStru, txDataStru any, callBacks ...func(error)) {
	err := c.newRequest(method, url, rxDataStru, txDataStru, callBacks...)
	if err != nil {
		log.Printf(debugTag+"NewRequest()1 err = %v", err)
	}
}

func (c *Client) newRequest(method, url string, rxDataStru, txDataStru any, callBacks ...func(error)) error {
	var err error
	var req *http.Request
	var res *http.Response
	//var fieldNames FieldNames

	url = c.BaseURL + url

	callBackSuccess := func(error) {
		err = fmt.Errorf(debugTag+"newRequest()2 INFORMATION: No success-callback has been provided: %w", err)
		log.Println(err, "req.URL =", req.URL) //This is the default returned if renderOk is called
	} //The function to be called to render the request results
	if len(callBacks) > 0 {
		if callBacks[0] != nil {
			callBackSuccess = callBacks[0]
		}
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

	// Do not set response-only or browser-controlled headers (e.g. Access-Control-Allow-Credentials, Origin).
	// Use fetch credentials instead so cookies are sent: this is handled in doRequest.
	res, err = c.doRequest(req) // This is the call to send the https request and receive the response
	if err != nil {
		err = fmt.Errorf(debugTag+"newRequest()4 from calling HTTPSClient.Do: %w", err)
		callBackFail(err)
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
			callBackFail(err)
		}
		err = fmt.Errorf(debugTag+"newRequest()7 server response StatusCode=%v: server message=%s", res.StatusCode, resBody)
		callBackFail(err)
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
		err = decodeJSON(res, &rxDataStru)
		if err != nil { //This decodes the JSON data in the body and puts it in the supplied structure.
			resBody, _ := io.ReadAll(res.Body)
			log.Printf("%v %v %v %v %v %v %v %v %+v %v %v", debugTag+"NewRequest()8a ", "err =", err, "req =", req, "res.Body =", string(resBody), "rxDataStru =", rxDataStru, "req.URL =", req.URL)
			err = fmt.Errorf(debugTag+"newRequest()8b failed to decode JSON data: %w", err)
			callBackFail(err)
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

	//returnData := &ReturnData{FieldNames: fieldNames}
	//callBackSuccess(nil, returnData)
	callBackSuccess(nil)
	return nil
}

// decodeJSON decodes the JSON data in the body and puts it in the supplied structure, and returns the field names.
func decodeJSON(res *http.Response, rxDataStru interface{}) error {
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	// Unmarshal into the provided structure.
	if err := json.Unmarshal(bodyBytes, rxDataStru); err != nil {
		// Provide a helpful error; return it so the caller can decide how to proceed.
		return fmt.Errorf("%vdecodeJSON: failed to unmarshal response body: %w", debugTag, err)
	}

	return nil
}

// Replace the problematic Do() call with this function
func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	// Create fetch options
	fetchOptions := map[string]interface{}{
		"method":      req.Method,
		"headers":     make(map[string]interface{}),
		"mode":        "cors",    // allow CORS
		"credentials": "include", // ensure cookies are sent
	}

	// Copy headers (but do not attempt to set Origin or other forbidden headers)
	headers := fetchOptions["headers"].(map[string]interface{})
	for key, values := range req.Header {
		if len(values) > 0 {
			headers[key] = values[0] // Take first value
		}
	}

	// Add body for POST/PUT/DELETE
	if req.Body != nil {
		// Read body safely (do not rely on ContentLength)
		bodyBytes, _ := io.ReadAll(req.Body)
		fetchOptions["body"] = string(bodyBytes)
	}

	// Use AbortController to support timeouts and cancellation when available
	var controller js.Value
	var timer *time.Timer
	if !js.Global().Get("AbortController").IsUndefined() {
		controller = js.Global().Get("AbortController").New()
		fetchOptions["signal"] = controller.Get("signal")
		if c != nil && c.HTTPClient != nil && c.HTTPClient.Timeout > 0 {
			d := c.HTTPClient.Timeout
			// start a timer that aborts the fetch when timeout expires
			timer = time.AfterFunc(d, func() {
				controller.Call("abort")
			})
		}
	}

	// Call JavaScript fetch
	promise := js.Global().Call("fetch", req.URL.String(), fetchOptions)

	// Create a channel to wait for the promise
	done := make(chan *http.Response, 1)
	errChan := make(chan error, 1)

	// Prepare handlers so we can Release them after use
	var thenHandler js.Func
	var catchHandler js.Func
	var thenHandlerSet bool
	var catchHandlerSet bool

	// Handle promise resolution
	thenHandler = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Release catch handler if set
		if catchHandlerSet {
			catchHandler.Release()
			catchHandlerSet = false
		}

		response := args[0]

		// Create Go http.Response
		resp := &http.Response{
			StatusCode: response.Get("status").Int(),
			Status:     response.Get("statusText").String(),
			Header:     make(http.Header),
		}

		// Copy response headers
		headersJS := response.Get("headers")
		if !headersJS.IsUndefined() && !headersJS.IsNull() {
			headersHandler := js.FuncOf(func(this js.Value, a []js.Value) interface{} {
				// headers.forEach callback signature: (value, key)
				value := a[0].String()
				key := a[1].String()
				resp.Header.Set(key, value)
				return nil
			})
			// Run forEach and release handler immediately
			headersJS.Call("forEach", headersHandler)
			headersHandler.Release()
		}

		// Get response text
		textPromise := response.Call("text")
		var textHandler js.Func
		textHandler = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			// Stop abort timer if running
			if timer != nil {
				timer.Stop()
			}
			if len(args) > 0 {
				text := args[0].String()
				resp.Body = &stringReadCloser{strings.NewReader(text)}
			}
			done <- resp
			// Release handlers
			textHandler.Release()
			if thenHandlerSet {
				thenHandler.Release()
				thenHandlerSet = false
			}
			return nil
		})
		textPromise.Call("then", textHandler)

		return nil
	})

	// Handle promise rejection
	catchHandler = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Stop abort timer if running
		if timer != nil {
			timer.Stop()
		}
		// Release thenHandler if set
		if thenHandlerSet {
			thenHandler.Release()
			thenHandlerSet = false
		}
		// Attempt to read message
		msg := "fetch failed"
		if len(args) > 0 {
			m := args[0]
			if !m.IsUndefined() && !m.IsNull() {
				if m.Get("message").Truthy() {
					msg = m.Get("message").String()
				}
			}
		}
		errChan <- fmt.Errorf("fetch failed: %v", msg)
		if catchHandlerSet {
			catchHandler.Release()
			catchHandlerSet = false
		}
		return nil
	})

	promise.Call("then", thenHandler)
	thenHandlerSet = true
	promise.Call("catch", catchHandler)
	catchHandlerSet = true

	// Wait for completion
	select {
	case resp := <-done:
		return resp, nil
	case err := <-errChan:
		return nil, err
	}
}

// OpenPopup opens a new browser window/tab pointing to path (relative to Client.BaseURL) with the given name and window features.
// If an absolute URL is provided (starts with http/https), it will be used as-is.
// Security notes:
// - Do not include "noopener" or "noreferrer" in features if you expect the popup to use window.opener.postMessage back to the opener — those disable opener access.
// - We sanitize features to remove noopener/noreferrer if present.
func (c *Client) OpenPopup(path, name, features string) {
	var urlStr string
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		urlStr = path
	} else {
		// Normalize slashes when joining BaseURL and path
		if strings.HasSuffix(c.BaseURL, "/") && strings.HasPrefix(path, "/") {
			urlStr = c.BaseURL[:len(c.BaseURL)-1] + path
		} else if !strings.HasSuffix(c.BaseURL, "/") && !strings.HasPrefix(path, "/") {
			urlStr = c.BaseURL + "/" + path
		} else {
			urlStr = c.BaseURL + path
		}
	}

	// Sanitize features: remove noopener/noreferrer which would break postMessage to opener
	lower := strings.ToLower(features)
	if strings.Contains(lower, "noopener") || strings.Contains(lower, "noreferrer") {
		sanitized := features
		sanitized = strings.ReplaceAll(sanitized, "noopener", "")
		sanitized = strings.ReplaceAll(sanitized, "noreferrer", "")
		// Remove duplicate commas and trim
		sanitized = strings.ReplaceAll(sanitized, ",,", ",")
		sanitized = strings.Trim(sanitized, ", ")
		log.Printf(debugTag+"OpenPopup removed unsafe feature(s) from features: original=%q sanitized=%q", features, sanitized)
		features = sanitized
	}

	js.Global().Call("open", urlStr, name, features)
}

// Destroy releases resources associated with the Client (closes idle connections and clears HTTP client).
// Call this when the application is shutting down or the client is no longer needed.
func (c *Client) Destroy() {
	if c == nil {
		return
	}
	if c.HTTPClient != nil {
		// Not all platforms implement CloseIdleConnections; simply clear the client reference.
		c.HTTPClient = nil
		log.Printf(debugTag + "Destroy() cleared HTTP client")
	}
}

// Helper type for response body
type stringReadCloser struct {
	*strings.Reader
}

func (s *stringReadCloser) Close() error {
	return nil
}
