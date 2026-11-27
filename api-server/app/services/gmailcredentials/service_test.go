package gmailcredentials

import (
	"crypto/sha256"
	"testing"
	"time"
)

func TestRenewalBoundToSessionAndConsumedOnce(t *testing.T) {
	now := time.Now()
	session := sha256.Sum256([]byte("initiating session"))
	other := sha256.Sum256([]byte("different session"))
	h := &Service{pending: renewal{state: "state", session: session, expires: now.Add(time.Minute)}}
	if _, ok := h.takeRenewal("state", other, now); ok {
		t.Fatal("accepted a different session")
	}
	if _, ok := h.takeRenewal("wrong", session, now); ok {
		t.Fatal("accepted a different state")
	}
	if _, ok := h.takeRenewal("state", session, now.Add(time.Minute)); ok {
		t.Fatal("accepted expired state")
	}
	if _, ok := h.takeRenewal("state", session, now); !ok {
		t.Fatal("rejected the initiating session")
	}
	if _, ok := h.takeRenewal("state", session, now); ok {
		t.Fatal("accepted a replay")
	}
}
