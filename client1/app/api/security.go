package api

func GroupPayload(v GroupDTO) GroupWrite {
	return GroupWrite{Version: v.Version, Name: v.Name, Description: v.Description}
}

func SecurityGrantPayload(v GroupResourceDTO) GroupResourceWrite {
	return GroupResourceWrite{Version: v.Version, GroupID: v.GroupID, ResourceID: v.ResourceID, AccessLevelID: v.AccessLevelID, AccessScopeID: v.AccessScopeID}
}
func MembershipPayload(v UserGroupDTO) UserGroupWrite {
	return UserGroupWrite{Version: v.Version, UserID: v.UserID, GroupID: v.GroupID}
}
