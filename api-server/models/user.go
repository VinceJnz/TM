package models

type User struct {
	ID       int    `json:"id" db:"ID"`
	Name     string `json:"name" db:"name"`
	Username string `json:"username" db:"username"`
	Email    string `json:"email" db:"email"`
}

type UserStatus struct {
	ID     int    `json:"id" db:"ID"`
	Status string `json:"status" db:"status"`
}
