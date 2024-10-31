package models

import "github.com/guregu/null/v5/zero"

// Item = UserSession information that is passed around in the server side application
type Session struct {
	//Session mdlToken.Item
	//User    mdlUser.Item
	//Control Control
	//Message mdlMessage.PageMsg
}

// Token stores cookies for user sessions
type Token struct {
	ID        int
	UserID    int
	Name      zero.String
	Host      zero.String
	TokenStr  zero.String
	Valid     string   //Text representation of the validID state
	ValidID   zero.Int //A flag for the application to know if the cookie is valid or not
	ValidFrom zero.Time
	ValidTo   zero.Time
}

// Control = access control information derived from a user's access levels and the requested resource
type Control struct {
	PrevURL       string //????
	ResourceName  string
	AccessLevel   string
	AccessLevelID int
	AccessType    string
	AccessTypeID  int
}

// AccessLevel is the enumeration of the data access level
type AccessLevel struct {
	ID          int64
	Name        string // Examples:
	Description string
}

// AccessType is the enumeration of the data access type
type AccessType struct {
	ID          int64
	Name        string // Examples: get, post, put, delete
	Description string
}

// Resource is the enumeration of the url name of the Resource being accessed
type Resource struct {
	ID   int64
	Name string // Example: trip, booking, user, etc
}
