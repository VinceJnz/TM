package models

import (
	"time"

	"github.com/guregu/null/v5/zero"
)

type GroupDTO struct {
	Version     int64       `json:"version"`
	ID          int         `json:"id"`
	Name        string      `json:"name"`
	Description zero.String `json:"description"`
	Created     time.Time   `json:"created"`
	Modified    time.Time   `json:"modified"`
}

type GroupResourceDTO struct {
	Version       int64     `json:"version"`
	ID            int       `json:"id"`
	GroupID       int       `json:"group_id"`
	Group         string    `json:"group_name"`
	ResourceID    int       `json:"resource_id"`
	Resource      string    `json:"resource"`
	AccessLevelID int       `json:"access_level_id"`
	AccessLevel   string    `json:"access_level"`
	AccessScopeID int       `json:"access_scope_id"`
	AccessScope   string    `json:"access_scope"`
	Created       time.Time `json:"created"`
	Modified      time.Time `json:"modified"`
}

type UserGroupDTO struct {
	Version  int64     `json:"version"`
	ID       int       `json:"id"`
	UserID   int       `json:"user_id"`
	User     string    `json:"user_name"`
	GroupID  int       `json:"group_id"`
	Group    string    `json:"group_name"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
}
