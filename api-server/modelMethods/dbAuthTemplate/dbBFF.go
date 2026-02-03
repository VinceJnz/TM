package dbAuthTemplate

import (
	"api-server/v2/models"
	"log"

	"github.com/jmoiron/sqlx"
)

// GetUserDeviceCredential retrieves the last used credential for a user for a specific device
func GetUserDeviceCredential(debugStr string, Db *sqlx.DB, userID int, deviceName, userAgent string) (*models.UserCredential, error) {
	var webUserCred models.UserCredential

	query := `SELECT id, user_id, credential_id, credential_data, last_used, device_name, device_metadata FROM st_webauthn_credentials WHERE user_id = $1 AND device_name = $2 AND device_metadata->>'user_agent' = $3 ORDER BY last_used DESC LIMIT 1`
	err := Db.QueryRow(query, userID, deviceName, userAgent).Scan(&webUserCred.ID, &webUserCred.UserID, &webUserCred.CredentialID, &webUserCred.LastUsed, &webUserCred.DeviceName, &webUserCred.DeviceMetadata)
	if err != nil {
		log.Printf("%sGetUserDeviceCredential()1.%s Failed to query last used credential: err = %v, userID = %v, deviceName = %v", debugTag, debugStr, err, userID, deviceName)
		return nil, err
	}

	return &webUserCred, nil
}
