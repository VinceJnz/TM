package api

func TripCostPayload(v TripCostDTO) TripCostWrite {
	return TripCostWrite{TripCostGroupID: v.TripCostGroupID, MemberStatusID: v.MemberStatusID, UserAgeGroupID: v.UserAgeGroupID, SeasonID: v.SeasonID, Amount: v.Amount}
}
