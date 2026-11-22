package models

type TripCostWrite struct {
	Version         int64 `json:"version"`
	TripCostGroupID int   `json:"trip_cost_group_id" rule:"required,minimum=1"`
	MemberStatusID  int   `json:"member_status_id" rule:"required,minimum=1"`
	UserAgeGroupID  int   `json:"user_age_group_id" rule:"required,minimum=1"`
	SeasonID        int   `json:"season_id" rule:"required,minimum=1"`
	Amount          Money `json:"amount"`
}
type TripCostGroupWrite struct {
	Version     int64  `json:"version"`
	Description string `json:"description" rule:"required,minLength=1,maxLength=50"`
}
