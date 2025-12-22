package dbAuthTemplate

import (
	"api-server/v2/models"
	"log"

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

	userFromDB, err := UserEmailReadQry(debugStr+"FindOrCreateUserByProvider ", Db, user.Email.String)
	if err == nil {
		// found existing user by email, update provider info
		log.Printf("%vFindOrCreateUserByProvider - user found by email, updating provider info: user = %+v, userFromDB = %+v", debugStr, user, userFromDB)
		user.ID = userFromDB.ID
		user.Provider = userFromDB.Provider
		user.ProviderID = userFromDB.ProviderID
		userID, err = UserWriteQry(debugStr+"FindOrCreateUserByProvider ", Db, user)
		if err != nil {
			return 0, err
		}
	}
	return userID, nil
}
