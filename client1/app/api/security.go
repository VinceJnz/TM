package api

func GroupPayload(v GroupDTO) GroupWrite {
	return GroupWrite{Name: v.Name, Description: v.Description}
}

func SecurityGrantPayload(v GroupResourceDTO) GroupResourceWrite {
	return GroupResourceWrite{GroupID: v.GroupID, ResourceID: v.ResourceID, AccessLevelID: v.AccessLevelID, AccessScopeID: v.AccessScopeID}
}
func MembershipPayload(v UserGroupDTO) UserGroupWrite {
	return UserGroupWrite{UserID: v.UserID, GroupID: v.GroupID}
}
