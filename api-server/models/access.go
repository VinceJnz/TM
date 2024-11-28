package models

import (
	"time"

	"github.com/guregu/null/v5/zero"
)

type Group struct {
	ID          int         `json:"id" db:"id"`
	Name        string      `json:"name" db:"name"`
	Description zero.String `json:"description" db:"description"`
	AdminFlag   bool        `json:"admin_flag" db:"admin_flag"`
	Created     time.Time   `json:"created" db:"created"`
	Modified    time.Time   `json:"modified" db:"modified"`
}

type UserGroup struct {
	ID       int       `json:"id" db:"id"`
	UserID   int       `json:"user_id" db:"user_id"`
	User     string    `json:"user_name" db:"user_name"`
	GroupID  int       `json:"group_id" db:"group_id"`
	Group    string    `json:"group_name" db:"group_name"`
	Created  time.Time `json:"created" db:"created"`
	Modified time.Time `json:"modified" db:"modified"`
}

type GroupResource struct {
	ID            int       `json:"id" db:"id"`
	GroupID       int       `json:"group_id" db:"group_id"`
	Group         string    `json:"group_name" db:"group_name"`
	ResourceID    int       `json:"resource_id" db:"resource_id"`
	Resource      string    `json:"resource" db:"resource"`
	AccessLevelID int       `json:"access_level_id" db:"access_level_id"`
	AccessLevel   string    `json:"access_level" db:"access_level"`
	AccessTypeID  int       `json:"access_type_id" db:"access_type_id"`
	AccessType    string    `json:"access_type" db:"access_type"`
	AdminFlag     bool      `json:"admin_flag" db:"admin_flag"`
	Created       time.Time `json:"created" db:"created"`
	Modified      time.Time `json:"modified" db:"modified"`
}
