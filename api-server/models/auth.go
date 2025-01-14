package models

import (
	"time"

	"github.com/guregu/null/v5/zero"
)

type TokenValid int

const (
	TokenFalse TokenValid = iota + 1
	TokenTrue
)

// Token stores cookies for user sessions
type Token struct {
	ID        int
	UserID    int
	Name      zero.String
	Host      zero.String
	TokenStr  zero.String
	Valid     zero.Bool //A flag for the application to know if the cookie is valid or not
	ValidFrom zero.Time
	ValidTo   zero.Time
}

// Session = access control information derived from a user's access levels and the requested resource. This info is passed to handlers in the ctx.
type Session struct {
	UserID         int
	PrevURL        string //????
	ResourceName   string
	ResourceID     int
	AccessMethod   string
	AccessMethodID int
	AccessType     string
	AccessTypeID   int
	AdminFlag      bool
}

type AccessLevel struct { // -- Example: 'none', 'get', 'post', 'put', 'delete' (OR: 'none', 'select', 'insert', 'update', 'delete')
	ID          int         `json:"id" db:"id"`
	Name        string      `json:"name" db:"name"`
	Description zero.String `json:"description" db:"description"`
	Created     time.Time   `json:"created" db:"created"`
	Modified    time.Time   `json:"modified" db:"modified"`
}

type AccessType struct { //-- Example: 'admin', 'owner', 'user' ????? don't know if this is useful
	ID          int         `json:"id" db:"id"`
	Name        string      `json:"name" db:"name"`
	Description zero.String `json:"description" db:"description"`
	Created     time.Time   `json:"created" db:"created"`
	Modified    time.Time   `json:"modified" db:"modified"`
}

type Resource struct { //-- Example: 'trips', 'users', 'bookings', 'member_status' (the url to to access the resource)
	ID          int         `json:"id" db:"id"`
	Name        string      `json:"name" db:"name"`
	Description zero.String `json:"description" db:"description"`
	Created     time.Time   `json:"created" db:"created"`
	Modified    time.Time   `json:"modified" db:"modified"`
}

// Resource is the enumeration of the url name of the Resource being accessed
type AccessCheck struct {
	AccessTypeID int
	AdminFlag    bool
}

// Resource is the enumeration of the url name of the Resource being accessed
type MenuUser struct {
	UserID    int    `json:"user_id" db:"user_id"`
	Name      string `json:"name" db:"name"`
	Group     string `json:"group" db:"group"`
	AdminFlag bool   `json:"admin_flag" db:"admin_flag"`
}

// Resource is the enumeration of the url name of the Resource being accessed
type MenuItem struct {
	UserID    int    `json:"user_id" db:"user_id"`
	Name      string `json:"resource" db:"resource"`
	AdminFlag bool   `json:"admin_flag" db:"admin_flag"`
}
