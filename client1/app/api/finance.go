package api

import (
	"client1/v2/app/httpProcessor"
	"fmt"
	"net/http"
	"strconv"
)

// Finance centralizes mutable payloads and errors for financial views. Writes
// deliberately exclude IDs, timestamps and provider-owned response fields.
type Finance struct{ client *httpProcessor.Client }

func NewFinance(client *httpProcessor.Client) *Finance { return &Finance{client: client} }
func (f *Finance) write(resource string, id int, input any) error {
	if id < 0 {
		return fmt.Errorf("invalid record ID")
	}
	method, path := http.MethodPost, "/"+resource
	if id > 0 {
		method = http.MethodPut
		path += "/" + strconv.Itoa(id)
	}
	return f.client.Request(method, path, nil, input)
}
func (f *Finance) SavePayment(id int, input PaymentWrite) error {
	return f.write("payments", id, input)
}
func (f *Finance) SaveRefund(id int, input RefundWrite) error { return f.write("refunds", id, input) }
func (f *Finance) remove(resource string, id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid record ID")
	}
	return f.client.Request(http.MethodDelete, "/"+resource+"/"+strconv.Itoa(id), nil, nil)
}
func (f *Finance) DeletePayment(id int) error { return f.remove("payments", id) }
func (f *Finance) DeleteRefund(id int) error  { return f.remove("refunds", id) }
