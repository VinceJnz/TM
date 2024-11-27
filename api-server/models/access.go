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
	GroupID  int       `json:"group_id" db:"group_id"`
	Created  time.Time `json:"created" db:"created"`
	Modified time.Time `json:"modified" db:"modified"`
}

type GroupResource struct {
	ID            int       `json:"id" db:"id"`
	GroupID       int       `json:"group_id" db:"group_id"`
	ResourceID    int       `json:"resource_id" db:"resource_id"`
	AccessLevelID int       `json:"access_level_id" db:"access_level_id"`
	AccessTypeID  int       `json:"access_type_id" db:"access_type_id"`
	AdminFlag     bool      `json:"admin_flag" db:"admin_flag"`
	Created       time.Time `json:"created" db:"created"`
	Modified      time.Time `json:"modified" db:"modified"`
}
