package models

type TripCostWrite struct {
	TripCostGroupID int   `json:"trip_cost_group_id"`
	MemberStatusID  int   `json:"member_status_id"`
	UserAgeGroupID  int   `json:"user_age_group_id"`
	SeasonID        int   `json:"season_id"`
	Amount          Money `json:"amount"`
}
type TripCostGroupWrite struct {
	Description string `json:"description"`
}
