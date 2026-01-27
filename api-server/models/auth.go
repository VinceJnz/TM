package models

import (
	"github.com/guregu/null/v5/zero"
)

type TokenValid int

const (
	TokenFalse TokenValid = iota + 1
	TokenTrue
)

// Token stores cookies for user sessions
type Token struct {
	ID          int
	UserID      int
	Name        zero.String
	Host        zero.String
	TokenStr    zero.String
	SessionData zero.String // This can be used to store session data in JSON format
	Valid       zero.Bool   //A flag for the application to know if the cookie is valid or not
	ValidFrom   zero.Time
	ValidTo     zero.Time
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
	Email          string
}
