package handlerAuthTemplate

import (
	"api-server/v2/models"
	"database/sql"
	"log"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jmoiron/sqlx"
)

const (
	//sqlWebAuthnFind   = `SELECT id, FROM st_webauthn_credentials WHERE id = $1`
	sqlWebAuthnFind   = `SELECT id FROM st_webauthn_credentials WHERE credential_id = $1`
	sqlWebAuthnRead   = `SELECT * FROM st_webauthn_credentials WHERE id = $1`
	sqlWebAuthnIdRead = `SELECT * FROM st_webauthn_credentials WHERE credential_id = $1`
	sqlWebAuthnInsert = `INSERT INTO st_webauthn_credentials (user_id, credential_id, public_key, aaguid, sign_count, attestation_type) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	sqlWebAuthnUpdate = `UPDATE st_webauthn_credentials SET user_id = $1, credential_id = $2, public_key = $3, aaguid = $4, sign_count = $5, attestation_type = $6 WHERE id = $7`

	sqlWebAuthnUserRead = `SELECT id, user_id, credential_id, credential_data, created, modified FROM st_webauthn_credentials WHERE user_id = $1`

	sqlUserWebAuthnUpdate = `UPDATE st_users SET webauthn_user_id = $2 WHERE id = $1`
)

// UserWebAuthnUpdate stores the user webAuth info in the user table
// Need to depreciate this ??????????????? why??????????????? it is currently used in by the following...
func UserWebAuthnUpdate(debugStr string, Db *sqlx.DB, user models.User) error {
	var err error

	result, err := Db.Exec(sqlUserWebAuthnUpdate, user.ID, user.WebAuthnUserID)
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %+v", debugTag+"UserAuthUpdate()2 ", "err =", err, "result =", result, "DB =", Db.DB)
		return err //Auth set failed
	}
	return nil //Auth set succeeded
}

// UserWebAuthnReadQry reads the webauthn credentials for a user from the database and returns them as a slice of webauthn.Credential.
func WebAuthnUserReadQry(debugStr string, Db *sqlx.DB, id int) ([]webauthn.Credential, error) {
	var record models.WebAuthnCredential    // database record
	var webAuthnCreds []webauthn.Credential // WebAuthn records
	var webAuthnCred webauthn.Credential

	rows, err := Db.Query(sqlWebAuthnUserRead, id)
	if err != nil {
		log.Printf("%sWebAuthnUserReadQry()1.%s: err = %v, id = %v", debugTag, debugStr, err, id)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&record.ID, &record.UserID, &record.CredentialID, &record.Credential, &record.Created, &record.Modified); err != nil {
			log.Printf("%sWebAuthnUserReadQry()2.%s: err = %v, id = %v webAuthnCreds = %v", debugTag, debugStr, err, id, webAuthnCreds)
			return nil, err
		}
		// Convert DB record to WebAuthnCredential
		record.Credential.Scan(webAuthnCred)
		webAuthnCreds = append(webAuthnCreds, webAuthnCred)
		log.Printf("%sWebAuthnUserReadQry()3.%s: id = %v webAuthnCreds = %v", debugTag, debugStr, id, webAuthnCreds)
	}

	return webAuthnCreds, rows.Err()
}

func WebAuthnReadQry(debugStr string, Db *sqlx.DB, id int) (models.WebAuthnCredential, error) {
	record := models.WebAuthnCredential{}
	err := Db.Get(&record, sqlWebAuthnRead, id)
	if err != nil {
		return models.WebAuthnCredential{}, err
	}
	return record, nil
}

// WebAuthnWriteQry writes the user record to the database, inserting or updating as necessary
func WebAuthnWriteQry(debugStr string, Db *sqlx.DB, record models.WebAuthnCredential) (int, error) {
	var err error
	Tx, err := Db.Beginx() // Start a transaction
	if err != nil {
		log.Printf("%v %v %v", debugTag+"WebAuthnWriteQry()2 - ", "err =", err)
		return 0, err
	}
	defer Tx.Rollback()                               // Ensure the transaction is rolled back if not committed
	_, err = WebAuthnWriteQryTx(debugStr, Tx, record) // Write the webauthn credential record to the database
	if err != nil {
		log.Printf("%v %v %v %v %v %v %T %+v", debugTag+"WebAuthnWriteQry()7 - ", "err =", err, "record.ID =", record.ID, "record =", record, record)
		return 0, err
	}
	if err := Tx.Commit(); err != nil { // Commit the transaction
		log.Printf("%v %v %v %v %+v", debugTag+"WebAuthnWriteQry()8 - ", "err =", err, "record =", record)
		return 0, err
	}
	return record.ID, err
}

func WebAuthnInsertQryTx(debugStr string, Db *sqlx.Tx, record models.WebAuthnCredential) (int, error) {
	err := Db.QueryRow(sqlWebAuthnInsert, record.UserID, record.CredentialID, record.Credential).Scan(&record.ID)
	return record.ID, err
}

func WebAuthnUpdateQryTx(debugStr string, Db *sqlx.Tx, record models.WebAuthnCredential) error {
	_, err := Db.Exec(sqlWebAuthnUpdate, record.UserID, record.CredentialID, record.Credential, record.ID)
	return err
}

// WebAuthnWriteQry writes the user record to the database, inserting or updating as necessary
func WebAuthnWriteQryTx(debugStr string, Db *sqlx.Tx, record models.WebAuthnCredential) (int, error) {
	var err error
	log.Printf("%v %v %v %v %+v", debugTag+"WebAuthnWriteQryTx()1 - ", "err =", err, "record =", record)

	err = Db.QueryRow(sqlWebAuthnFind, record.ID).Scan(&record.CredentialID) // Check to see if a record exists
	switch err {
	case sql.ErrNoRows:
		record.ID, err = WebAuthnInsertQryTx(debugStr, Db, record) //No Existing record found so we are okay to insert the new record
		if err != nil {
			log.Printf("%v %v %v %v %v %v %T %+v", debugTag+"WebAuthnWriteQryTx()5 - ", "err =", err, "record.ID =", record.ID, "record =", record, record)
			return 0, err
		}
	case nil:
		err = WebAuthnUpdateQryTx(debugStr, Db, record) //Existing record found so we are okay to update the record
		if err != nil {
			log.Printf("%v %v %v %v %v %v %T %+v", debugTag+"WebAuthnWriteQryTx()6 - ", "err =", err, "record.ID =", record.ID, "record =", record, record)
			return 0, err
		}
	default:
		log.Printf("%v %v %v %v %v %v %T %+v", debugTag+"WebAuthnWriteQryTx()7 - ", "err =", err, "record.ID =", record.ID, "record =", record, record)
		return 0, err
	}
	return record.ID, err
}

func WebAuthnUserRegisterQry(debugStr string, Db *sqlx.DB, user models.User, cred models.WebAuthnCredential) ([]webauthn.Credential, error) {
	var webAuthnCreds []webauthn.Credential
	// Setup a transaction to ensure atomicity
	tx, err := Db.Beginx()
	if err != nil {
		log.Printf("%v %v %v", debugTag+"WebAuthnUserRegisterQry()1 - ", "err =", err)
		return nil, err
	}
	defer tx.Rollback()

	// Check if the user exists
	err = tx.Get(&user, sqlUserNameRead, user.Username) // Read the user record from the database
	if err != nil {
		log.Printf("%v %v %v %v %v %v %T %+v", debugTag+"WebAuthnUserRegisterQry()2 - ", "err =", err, "user.Username =", user.Username, "user =", user, user)
		return nil, err
	}

	// check if the credential already exists
	err = tx.Get(&cred.CredentialID, sqlWebAuthnFind, cred.CredentialID) // Read the user record from the database
	if err == nil {
		// Credential already exists, return an error
		log.Printf("%v %v %+v", debugTag+"WebAuthnUserRegisterQry()3 - cred already exists", "cred =", cred)
		return nil, models.ErrWebAuthnCredentialExists // Return an error indicating the credential already exists
	}

	// If no errors then Save the user and credential to the db
	user.ID, err = UserWriteQryTx(debugStr, tx, user) // Write the user record to the database
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"WebAuthnUserRegisterQry()4 - ", "err =", err, "user =", user)
		return nil, err
	}

	_, err = WebAuthnWriteQryTx(debugStr, tx, cred) // Write the webauthn credential record to the database
	if err != nil {
		log.Printf("%v %v %v %v %v %v %T %+v", debugTag+"WebAuthnUserRegisterQry()4 - ", "err =", err, "cred =", cred, "user =", user, user)
		return nil, err
	}

	// No errors, so we can commit the transaction
	if err := tx.Commit(); err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"WebAuthnUserRegisterQry()4 - ", "err =", err, "user =", user)
		return nil, err
	}
	return webAuthnCreds, nil
}
