package models

type GroupResourceWrite struct {
	GroupID       int `json:"group_id"`
	ResourceID    int `json:"resource_id"`
	AccessLevelID int `json:"access_level_id"`
	AccessScopeID int `json:"access_scope_id"`
}
type UserGroupWrite struct {
	UserID  int `json:"user_id"`
	GroupID int `json:"group_id"`
}
