package dbAuthTemplate

import (
	"api-server/v2/models"

	"github.com/jmoiron/sqlx"
)

// func FindOrCreateUserByProvider(debugStr string, Db *sqlx.DB, provider, providerID, email, name string) (int, error) {
func FindOrCreateUserByProvider(debugStr string, Db *sqlx.DB, user models.User) (int, error) {
	var userID int
	err := Db.Get(&userID, `SELECT id FROM users WHERE provider=$1 AND provider_id=$2`, user.Provider, user.ProviderID)
	if err == nil {
		// found existing user
		return userID, nil
	}

	userID, err = UserWriteQry(debugStr+"FindOrCreateUserByProvider ", Db, user)
	//err = Db.QueryRow(`INSERT INTO users (provider, provider_id, email, name) VALUES ($1, $2, $3, $4) RETURNING id`, provider, providerID, email, name).Scan(&userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}
