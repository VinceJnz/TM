package models

type GroupWrite struct {
	Version     int64  `json:"version,omitempty"`
	Name        string `json:"name" rule:"required,minLength=1,maxLength=45"`
	Description string `json:"description" rule:"maxLength=45"`
}

type GroupResourceWrite struct {
	Version       int64 `json:"version,omitempty"`
	GroupID       int   `json:"group_id" rule:"required,minimum=1"`
	ResourceID    int   `json:"resource_id" rule:"required,minimum=1"`
	AccessLevelID int   `json:"access_level_id" rule:"required,minimum=1"`
	AccessScopeID int   `json:"access_scope_id" rule:"required,minimum=1"`
}
type UserGroupWrite struct {
	Version int64 `json:"version,omitempty"`
	UserID  int   `json:"user_id" rule:"required,minimum=1"`
	GroupID int   `json:"group_id" rule:"required,minimum=1"`
}
