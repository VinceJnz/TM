package api

func SecurityGrantPayload(v GroupResourceDTO) GroupResourceWrite {
	return GroupResourceWrite{GroupID: v.GroupID, ResourceID: v.ResourceID, AccessLevelID: v.AccessLevelID, AccessScopeID: v.AccessScopeID}
}
func MembershipPayload(v UserGroupDTO) UserGroupWrite {
	return UserGroupWrite{UserID: v.UserID, GroupID: v.GroupID}
}
