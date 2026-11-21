package handlerBooking

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/testdb"
	"api-server/v2/models"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
)

func TestActualBookingQueryPagesSameOwnerWithoutDuplicates(t *testing.T) {
	db := testdb.Open(t, testdb.Bookings+`INSERT INTO at_bookings(id,owner_id,trip_id,booking_status_id,notes,from_date,to_date) VALUES(12,1,1,1,'',NOW(),NOW()+INTERVAL '1 day'),(13,1,1,1,'',NOW(),NOW()+INTERVAL '1 day');`)
	app := appCore.New(false)
	app.Db = db
	handler := New(app)
	for page, want := range []int{11, 12, 13} {
		request := httptest.NewRequest("GET", fmt.Sprintf("/?limit=1&offset=%d", page), nil)
		request = request.WithContext(context.WithValue(request.Context(), app.SessionIDKey, &models.Session{UserID: 1, AccessScope: "own"}))
		response := httptest.NewRecorder()
		handler.GetAll(response, request)
		var items []struct {
			ID int `json:"id"`
		}
		if response.Code != 200 {
			t.Fatal(response.Code, response.Body.String())
		}
		if err := json.Unmarshal(response.Body.Bytes(), &items); err != nil || len(items) != 1 || items[0].ID != want {
			t.Fatal(items, err)
		}
	}
}
