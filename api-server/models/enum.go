package models

import (
	"time"

	"github.com/guregu/null/v5/zero"
)

type AccessLevel struct { // -- Example: 'none', 'get', 'post', 'put', 'delete' (OR: 'none', 'select', 'insert', 'update', 'delete')
	ID          int         `json:"id" db:"id"`
	Name        string      `json:"name" db:"name"`
	Description zero.String `json:"description" db:"description"`
	Created     time.Time   `json:"created" db:"created"`
	Modified    time.Time   `json:"modified" db:"modified"`
}

type AccessScope struct { // -- Example scope: 'any', 'own'
	ID          int         `json:"id" db:"id"`
	Name        string      `json:"name" db:"name"`
	Description zero.String `json:"description" db:"description"`
	Created     time.Time   `json:"created" db:"created"`
	Modified    time.Time   `json:"modified" db:"modified"`
}

// Deprecated: AccessType is a legacy compatibility alias. Prefer AccessScope.
type AccessType = AccessScope

type Resource struct { //-- Example: 'trips', 'users', 'bookings', 'member_status' (the url to to access the resource)
	ID          int         `json:"id" db:"id"`
	Name        string      `json:"name" db:"name"`
	Description zero.String `json:"description" db:"description"`
	Created     time.Time   `json:"created" db:"created"`
	Modified    time.Time   `json:"modified" db:"modified"`
}

// Resource is the enumeration of the url name of the Resource being accessed
type AccessCheck struct {
	AccessScopeID int
	AccessScope   string
	//AdminFlag    bool
	Group string
}

// Resource is the enumeration of the url name of the Resource being accessed
type MenuUser struct {
	UserID int    `json:"user_id" db:"user_id"`
	Name   string `json:"name" db:"name"`
	Group  string `json:"group" db:"group"`
}

// Resource is the enumeration of the url name of the Resource being accessed
type MenuItem struct {
	UserID      int    `json:"user_id" db:"user_id"`
	Name        string `json:"resource" db:"resource"`
	AccessLevel string `json:"access_level" db:"access_level"`
	AccessScope string `json:"access_scope" db:"access_scope"`
}
