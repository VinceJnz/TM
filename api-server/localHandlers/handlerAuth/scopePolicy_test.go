package handlerAuth

import (
	"api-server/v2/models"
	"testing"
)

func TestMenuExcludesUnsupportedGrants(t *testing.T) {
	items := []models.MenuItem{
		{Name: "myBookings", AccessLevel: "get", AccessScope: "own"},
		{Name: "tripsReport", AccessLevel: "get", AccessScope: "own"},
		{Name: "tripDifficulty", AccessLevel: "get", AccessScope: "own"},
		{Name: "tripDifficulty", AccessLevel: "get", AccessScope: "any"},
	}
	got := supportedMenuItems(items)
	if len(got) != 3 || got[0] != items[0] || got[1] != items[1] || got[2] != items[3] {
		t.Fatalf("unexpected menu capabilities: %+v", got)
	}
}

func TestUnsupportedOwnAndUnknownScopesFailClosed(t *testing.T) {
	for _, resource := range []string{"users", "groupBooking", "payments", "refunds", "securityUserGroup", "tripDifficulty", "future-resource"} {
		for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
			if supportsScope(resource, method, "own") {
				t.Fatalf("unsupported ownership grant accepted: %s %s", method, resource)
			}
			if !supportsScope(resource, method, "any") {
				t.Fatal("explicit any capability rejected")
			}
		}
	}
	if supportsScope("bookings", "GET", "unknown") || supportsScope("trips", "GET", "own") {
		t.Fatal("unscoped reads accepted")
	}
	for _, resource := range []string{"bookings", "myBookings", "bookingPeople", "myProfile"} {
		if !supportsScope(resource, "GET", "own") {
			t.Fatal("scoped member resource rejected")
		}
	}
}
