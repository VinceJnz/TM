package api

func AccountPayload(v AccountDTO) AccountWrite {
	return AccountWrite{Version: v.Version, Name: v.Name, Username: v.Username, Email: v.Email, Address: v.Address, MemberCode: v.MemberCode, BirthDate: v.BirthDate, UserAgeGroupID: v.UserAgeGroupID, MemberStatusID: v.MemberStatusID, AccountStatusID: v.AccountStatusID, AccountHidden: v.AccountHidden}
}
