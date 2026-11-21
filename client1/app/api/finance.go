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
func (f *Finance) write(resource string, id int, input any, reason []string) error {
	if id < 0 {
		return fmt.Errorf("invalid record ID")
	}
	method, path := http.MethodPost, "/"+resource
	if id > 0 {
		method = http.MethodPut
		path += "/" + strconv.Itoa(id)
	}
	why := ""
	if len(reason) > 0 {
		why = reason[0]
	}
	return f.client.RequestWithHeaders(method, path, nil, input, map[string]string{"X-Change-Reason": why})
}
func (f *Finance) SavePayment(id int, input PaymentWrite, reason ...string) error {
	return f.write("payments", id, input, reason)
}
func (f *Finance) SaveRefund(id int, input RefundWrite, reason ...string) error {
	return f.write("refunds", id, input, reason)
}
func (f *Finance) remove(resource string, id int, reason []string) error {
	if id <= 0 {
		return fmt.Errorf("invalid record ID")
	}
	why := ""
	if len(reason) > 0 {
		why = reason[0]
	}
	return f.client.RequestWithHeaders(http.MethodDelete, "/"+resource+"/"+strconv.Itoa(id), nil, nil, map[string]string{"X-Change-Reason": why})
}
func (f *Finance) DeletePayment(id int, reason ...string) error {
	return f.remove("payments", id, reason)
}
func (f *Finance) DeleteRefund(id int, reason ...string) error {
	return f.remove("refunds", id, reason)
}
