package dbAuthTemplate

import (
	authStore "api-server/v2/modelMethods/authentication"
	"api-server/v2/models"
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

const sqlUserIDByProvider = `SELECT id FROM st_users WHERE provider=$1 AND provider_id=$2`

type userState int

const (
	userStateUnknown userState = iota
	userStateUpdated
	userStateCreated
	userStateNotFound
)

// Compatibility entry point for registration; existing accounts only link identity.
func FindOrCreateUserByProvider(debugStr string, Db *sqlx.DB, user models.User) (int, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	existing, found, err := FindUserByProviderOrEmail(debugStr, Db, user)
	if err != nil {
		return 0, false, err
	}
	if found {
		id, err := (authStore.Store{DB: Db}).LinkIdentity(ctx, existing.ID, user.Provider.String, user.ProviderID.String, user.Email.String)
		return id, false, err
	}
	id, err := UserWriteQry(debugStr, Db, user)
	return id, true, err
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
	if err != sql.ErrNoRows {
		return models.User{}, false, err
	}

	if user.Email.String == "" {
		return models.User{}, false, nil
	}

	record, err := UserEmailReadQry(debugStr+"FindUserByProviderOrEmail:byEmail ", Db, user.Email.String)
	if err == nil {
		return record, true, nil
	}
	if err != sql.ErrNoRows {
		return models.User{}, false, err
	}

	return models.User{}, false, nil
}
