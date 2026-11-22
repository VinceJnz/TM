package api

func TripCostPayload(v TripCostDTO) TripCostWrite {
	return TripCostWrite{Version: v.Version, TripCostGroupID: v.TripCostGroupID, MemberStatusID: v.MemberStatusID, UserAgeGroupID: v.UserAgeGroupID, SeasonID: v.SeasonID, Amount: v.Amount}
}
