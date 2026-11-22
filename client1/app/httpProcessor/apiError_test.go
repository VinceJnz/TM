package httpProcessor

import "testing"

func TestStructuredAndLegacyErrorMessages(t *testing.T) {
	for _, body := range []string{`{"error":{"code":"conflict","message":"Refresh this record"}}`, `{"error":"conflict","message":"Refresh this record"}`, "Refresh this record"} {
		err := responseError(409, []byte(body))
		if err.Error() != "Refresh this record" {
			t.Fatal(err)
		}
	}
}
