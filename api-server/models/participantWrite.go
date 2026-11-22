package models

type ParticipantWrite struct {
	BookingID int    `json:"booking_id"`
	PersonID  int    `json:"person_id"`
	Notes     string `json:"notes"`
}
