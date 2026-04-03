package dbStandardTemplate

import (
	"net/http"

	"github.com/jmoiron/sqlx"
)

// This file contains example implementations of DeleteWithChildCheck for various entities.
// These examples show how to properly check for child records before deletion.

// DeleteUserWithChecks: Example of deleting a user with child record checks
// This prevents deleting users who have bookings, payments, or other related data
func DeleteUserWithChecks(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, userID int) {
	childChecks := []ChildCheckQuery{
		{
			TableName: "bookings (as owner)",
			Query:     "SELECT COUNT(*) FROM at_bookings WHERE owner_id = $1",
		},
		{
			TableName: "booking participants",
			Query:     "SELECT COUNT(*) FROM at_booking_people WHERE user_id = $1",
		},
		{
			TableName: "payments",
			Query:     "SELECT COUNT(*) FROM at_payments p JOIN at_bookings b ON b.id = p.booking_id WHERE b.owner_id = $1",
		},
		{
			TableName: "trips (as owner)",
			Query:     "SELECT COUNT(*) FROM at_trips WHERE owner_id = $1",
		},
		{
			TableName: "group bookings (as owner)",
			Query:     "SELECT COUNT(*) FROM at_group_bookings WHERE owner_id = $1",
		},
	}

	deleteQuery := "DELETE FROM st_users WHERE id = $1"
	DeleteWithChildCheck(w, r, debugStr, Db, deleteQuery, childChecks, userID)
}

// DeleteTripWithChecks: Example of deleting a trip with child record checks
// This prevents deleting trips that have bookings
func DeleteTripWithChecks(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, tripID int) {
	childChecks := []ChildCheckQuery{
		{
			TableName: "bookings",
			Query:     "SELECT COUNT(*) FROM at_bookings WHERE trip_id = $1",
		},
	}

	deleteQuery := "DELETE FROM at_trips WHERE id = $1"
	DeleteWithChildCheck(w, r, debugStr, Db, deleteQuery, childChecks, tripID)
}

// DeleteBookingWithChecks: Example of deleting a booking with child record checks
// Note: If CASCADE is set up in DB, booking_people and payments will auto-delete
// This function is for when you want explicit control or better error messages
func DeleteBookingWithChecks(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, bookingID int) {
	childChecks := []ChildCheckQuery{
		{
			TableName: "booking participants",
			Query:     "SELECT COUNT(*) FROM at_booking_people WHERE booking_id = $1",
		},
		{
			TableName: "payments",
			Query:     "SELECT COUNT(*) FROM at_payments WHERE booking_id = $1",
		},
	}

	deleteQuery := "DELETE FROM at_bookings WHERE id = $1"
	DeleteWithChildCheck(w, r, debugStr, Db, deleteQuery, childChecks, bookingID)
}

// DeleteSecurityGroupWithChecks: Example of deleting a security group with child record checks
// This prevents deleting groups that have users or resource permissions
func DeleteSecurityGroupWithChecks(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, groupID int) {
	childChecks := []ChildCheckQuery{
		{
			TableName: "user memberships",
			Query:     "SELECT COUNT(*) FROM st_user_group WHERE group_id = $1",
		},
		{
			TableName: "resource permissions",
			Query:     "SELECT COUNT(*) FROM st_group_resource WHERE group_id = $1",
		},
	}

	deleteQuery := "DELETE FROM st_group WHERE id = $1"
	DeleteWithChildCheck(w, r, debugStr, Db, deleteQuery, childChecks, groupID)
}

// DeleteEnumValueWithChecks: Generic example for deleting enum values
// Enum values should NEVER be deleted if they're in use
func DeleteBookingStatusWithChecks(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, statusID int) {
	childChecks := []ChildCheckQuery{
		{
			TableName: "bookings using this status",
			Query:     "SELECT COUNT(*) FROM at_bookings WHERE booking_status_id = $1",
		},
	}

	deleteQuery := "DELETE FROM et_booking_status WHERE id = $1"
	DeleteWithChildCheck(w, r, debugStr, Db, deleteQuery, childChecks, statusID)
}

// DeleteTripDifficultyWithChecks: Example for trip difficulty enum
func DeleteTripDifficultyWithChecks(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, difficultyID int) {
	childChecks := []ChildCheckQuery{
		{
			TableName: "trips using this difficulty",
			Query:     "SELECT COUNT(*) FROM at_trips WHERE difficulty_level_id = $1",
		},
	}

	deleteQuery := "DELETE FROM et_trip_difficulty WHERE id = $1"
	DeleteWithChildCheck(w, r, debugStr, Db, deleteQuery, childChecks, difficultyID)
}

// DeleteTripStatusWithChecks: Example for trip status enum
func DeleteTripStatusWithChecks(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, statusID int) {
	childChecks := []ChildCheckQuery{
		{
			TableName: "trips using this status",
			Query:     "SELECT COUNT(*) FROM at_trips WHERE trip_status_id = $1",
		},
	}

	deleteQuery := "DELETE FROM et_trip_status WHERE id = $1"
	DeleteWithChildCheck(w, r, debugStr, Db, deleteQuery, childChecks, statusID)
}

// DeleteTripTypeWithChecks: Example for trip type enum
func DeleteTripTypeWithChecks(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, typeID int) {
	childChecks := []ChildCheckQuery{
		{
			TableName: "trips using this type",
			Query:     "SELECT COUNT(*) FROM at_trips WHERE trip_type_id = $1",
		},
	}

	deleteQuery := "DELETE FROM et_trip_type WHERE id = $1"
	DeleteWithChildCheck(w, r, debugStr, Db, deleteQuery, childChecks, typeID)
}

// Note: For entities that should CASCADE delete (like tokens, credentials),
// you can use the regular Delete() function since the database will handle
// the cascade automatically once the foreign keys are properly configured.
//
// Example entities that can use regular Delete():
// - st_token (cascades from user)
// - st_webauthn_credentials (cascades from user)
// - at_booking_people (cascades from booking, if CASCADE is set)
// - at_payments (cascades from booking, if CASCADE is set)

// Made with Bob
