package resourcequery

import (
	"context"
	"github.com/jmoiron/sqlx"
)

type UserOption struct {
	ID         int    `json:"id" db:"id"`
	Name       string `json:"name" db:"name"`
	Username   string `json:"username" db:"username"`
	MemberCode string `json:"member_code,omitempty" db:"member_code"`
	Membership string `json:"membership,omitempty" db:"membership"`
}

// UserOptions returns bounded selector fields, never account data.
func UserOptions(ctx context.Context, db *sqlx.DB, query string, id int) ([]UserOption, error) {
	items := []UserOption{}
	err := db.SelectContext(ctx, &items, `SELECT id,name,username FROM st_users WHERE ($2>0 AND id=$2) OR ($2=0 AND length($1)>=2 AND strpos(lower(name||' '||username),lower($1))>0) ORDER BY id LIMIT 25`, query, id)
	return items, err
}
