package dbAuthTemplate

import (
	"api-server/v2/models"
	"log"

	"github.com/jmoiron/sqlx"
)

const sqlUserIDByProvider = `SELECT id FROM users WHERE provider=$1 AND provider_id=$2`

type userState int

const (
	userStateUnknown userState = iota
	userStateUpdated
	userStateCreated
	userStateNotFound
)

// func FindOrCreateUserByProvider(debugStr string, Db *sqlx.DB, provider, providerID, email, name string) (int, error) {
func FindOrCreateUserByProvider(debugStr string, Db *sqlx.DB, user models.User) (int, bool, error) {
	var userID int
	//var userStatus userState
	err := Db.Get(&userID, sqlUserIDByProvider, user.Provider, user.ProviderID)
	if err == nil {
		// found existing user
		return userID, false, nil
	}

	userFromDB, err := UserEmailReadQry(debugStr+"FindOrCreateUserByProvider ", Db, user.Email.String)
	if err == nil {
		// found existing user by email, merge provider info into the existing user
		log.Printf("%vFindOrCreateUserByProvider - user found by email, merging provider info: incoming user = %+v, userFromDB = %+v", debugStr, user, userFromDB)
		user.ID = userFromDB.ID
		// Preserve existing fields when incoming values are empty.
		if user.Name == "" {
			user.Name = userFromDB.Name
		}
		if user.Username == "" {
			user.Username = userFromDB.Username
		}
		if user.Email.String == "" {
			user.Email = userFromDB.Email
		}
		if user.AccountHidden.Valid == false {
			user.AccountHidden = userFromDB.AccountHidden
		}
		if user.BirthDate.Valid == false {
			user.BirthDate = userFromDB.BirthDate
		}
		if user.Address.Valid == false {
			user.Address = userFromDB.Address
		}
		// CRITICAL: Preserve existing account status - admins set this manually
		user.AccountStatusID = userFromDB.AccountStatusID
		// For provider fields, prefer incoming (new) values; fall back to existing DB values if incoming is empty
		if !user.Provider.Valid || user.Provider.String == "" {
			user.Provider = userFromDB.Provider
		}
		if !user.ProviderID.Valid || user.ProviderID.String == "" {
			user.ProviderID = userFromDB.ProviderID
		}
		log.Printf("%vFindOrCreateUserByProvider - merged user data: %+v", debugStr, user)

		userID, err = UserWriteQry(debugStr+"FindOrCreateUserByProvider ", Db, user)
		if err != nil {
			return 0, false, err
		}
		return userID, false, nil
	} else {
		// No existing user found by provider or email: insert a new user so provider info is persisted
		log.Printf("%vFindOrCreateUserByProvider - no existing user found; inserting new user: %+v", debugStr, user)
		userID, err = UserWriteQry(debugStr+"FindOrCreateUserByProvider:insert ", Db, user)
		if err != nil {
			return 0, false, err
		}
		return userID, true, nil
	}
}

func FindUserByProviderOrEmail(debugStr string, Db *sqlx.DB, user models.User) (models.User, bool, error) {
	var userID int
	err := Db.Get(&userID, sqlUserIDByProvider, user.Provider, user.ProviderID)
	if err == nil {
		record, readErr := UserReadQry(debugStr+"FindUserByProviderOrEmail:byProvider ", Db, userID)
		if readErr != nil {
			return models.User{}, false, readErr
		}
		return record, true, nil
	}

	if user.Email.String == "" {
		return models.User{}, false, nil
	}

	record, err := UserEmailReadQry(debugStr+"FindUserByProviderOrEmail:byEmail ", Db, user.Email.String)
	if err == nil {
		return record, true, nil
	}

	return models.User{}, false, nil
}
