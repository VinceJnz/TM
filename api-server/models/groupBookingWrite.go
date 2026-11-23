package models

type GroupBookingWrite struct {
	Version   int64  `json:"version"`
	GroupName string `json:"group_name" rule:"required,maxLength=255"`
	OwnerID   int    `json:"owner_id"`
}
