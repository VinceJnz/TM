// "go:build js && wasm"

package httpProcessor3

import (
	"encoding/json"
	"fmt"
	"log"
	"syscall/js"
)

// Client provides the connection to the rest interface (is used by a store to read/write/update data????)
type Client struct {
	//Ctx     context.Context
	BaseURL string
	//apiKey    string
	//User       *mdlUser.User
	//Session bool
	//CookieJar  *cookiejar.Jar
	//HTTPClient *http.Client
}

type FieldNames map[string]string

type ReturnData struct {
	FieldNames FieldNames
}

func New(baseURL string) *Client {
	// Create a cookie jar
	//jar, err := cookiejar.New(nil)
	//if err != nil {
	//	log.Fatalf(debugTag+"New() Error creating cookie jar: %v", err)
	//}

	//httpClient := &http.Client{
	//Jar: jar,
	//Timeout: time.Minute,
	//Timeout: 5 * time.Second,
	//Transport: &http.Transport{
	//	TLSClientConfig: &tls.Config{
	//		//Certificates: []tls.Certificate{cert},
	//		//RootCAs:      caCertPool,
	//	},
	//},
	//}

	return &Client{
		BaseURL: baseURL,
		//CookieJar:  jar,
		//HTTPClient: httpClient,
	}
}

// NewRequest (WASM version) - uses browser fetch instead of net/http.
// Signature matches the original so callers don't change.
func (c *Client) NewRequest(method, url string, rxDataStru, txDataStru any, callBacks ...func(error, *ReturnData)) {
	// run async to match behavior of original
	go func() {
		callBackSuccess := func(error, *ReturnData) {
			log.Printf("httpProcessor.NewRequest (wasm) INFO: no success callback provided")
		}
		if len(callBacks) > 0 && callBacks[0] != nil {
			callBackSuccess = callBacks[0]
		}
		callBackFail := func(error, *ReturnData) {
			log.Printf("httpProcessor.NewRequest (wasm) INFO: no fail callback provided")
		}
		if len(callBacks) > 1 && callBacks[1] != nil {
			callBackFail = callBacks[1]
		}

		fullURL := c.BaseURL + url

		// prepare body if needed
		var bodyStr string
		if method == "POST" || method == "PUT" || method == "DELETE" {
			if txDataStru != nil {
				b, err := json.Marshal(txDataStru)
				if err != nil {
					callBackFail(fmt.Errorf("marshal txDataStru: %w", err), nil)
					return
				}
				bodyStr = string(b)
			}
		}

		// build fetch options
		opts := map[string]interface{}{
			"method": method,
			"headers": map[string]interface{}{
				"Content-Type": "application/json",
			},
			// include credentials so cookies/session are sent; remove if not desired
			"credentials": "include",
		}
		if bodyStr != "" {
			opts["body"] = bodyStr
		}

		fetchPromise := js.Global().Call("fetch", fullURL, opts)
		then := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resp := args[0]
			status := resp.Get("status").Int()

			// Read response text (JSON expected)
			textPromise := resp.Call("text")
			textThen := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				text := args[0].String()

				// Non-2xx -> fail
				if status < 200 || status >= 400 {
					callBackFail(fmt.Errorf("server status %d: %s", status, text), nil)
					return nil
				}

				// If caller expects response data, unmarshal into rxDataStru
				if rxDataStru != nil && len(text) > 0 {
					if err := json.Unmarshal([]byte(text), &rxDataStru); err != nil {
						callBackFail(fmt.Errorf("unmarshal response: %w", err), nil)
						return nil
					}
					// simple FieldNames extraction omitted - return empty map
					callBackSuccess(nil, &ReturnData{FieldNames: make(FieldNames)})
					return nil
				}

				// no response body expected
				callBackSuccess(nil, &ReturnData{FieldNames: make(FieldNames)})
				return nil
			})
			textPromise.Call("then", textThen)
			return nil
		})

		// handle network/fetch errors
		catch := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			errVal := args[0]
			callBackFail(fmt.Errorf("fetch error: %s", errVal.String()), nil)
			return nil
		})

		fetchPromise.Call("then", then).Call("catch", catch)
	}()
}
