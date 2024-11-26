package models

import "time"

type Group struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	AdminFlag   bool      `json:"admin_flag" db:"admin_flag"`
	Created     time.Time `json:"created" db:"created"`
	Modified    time.Time `json:"modified" db:"modified"`
}

type UserGroup struct {
	ID       int       `json:"id" db:"id"`
	UserID   string    `json:"user_id" db:"user_id"`
	GroupID  string    `json:"group_id" db:"group_id"`
	Created  time.Time `json:"created" db:"created"`
	Modified time.Time `json:"modified" db:"modified"`
}

type GroupResource struct {
	ID            int       `json:"id" db:"id"`
	GroupID       string    `json:"group_id" db:"group_id"`
	ResourceID    string    `json:"resource_id" db:"resource_id"`
	AccessLevelID string    `json:"access_level_id" db:"access_level_id"`
	AccessTypeID  string    `json:"access_type_id" db:"access_type_id"`
	AdminFlag     string    `json:"admin_flag" db:"admin_flag"`
	Created       time.Time `json:"created" db:"created"`
	Modified      time.Time `json:"modified" db:"modified"`
}
