package handlerAuth

import (
	"net/http/httptest"
	"testing"
)

func TestSetRestResource_CheckoutRouteUsesBookingsResource(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest("POST", "/api/v1/bookings/checkout/create/37", nil)

	resource, err := h.setRestResource(req)
	if err != nil {
		t.Fatalf("setRestResource returned error: %v", err)
	}

	if resource.ResourceName != "bookings" {
		t.Fatalf("expected resource name bookings, got %q", resource.ResourceName)
	}
	if resource.AccessMethod != "POST" {
		t.Fatalf("expected access method POST, got %q", resource.AccessMethod)
	}
}

func TestSetRestResource_LegacySevenSegmentBehaviorPreserved(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest("GET", "/api/v1/trips/12/bookings/77", nil)

	resource, err := h.setRestResource(req)
	if err != nil {
		t.Fatalf("setRestResource returned error: %v", err)
	}

	if resource.ResourceName != "bookings" {
		t.Fatalf("expected resource name bookings, got %q", resource.ResourceName)
	}
	if resource.AccessMethod != "GET" {
		t.Fatalf("expected access method GET, got %q", resource.AccessMethod)
	}
}
