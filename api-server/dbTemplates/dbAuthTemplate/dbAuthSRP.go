package dbAuthTemplate

import (
	"api-server/v2/localHandlers/handlerUserAccountStatus"
	"api-server/v2/models"
	"errors"
	"log"
	"math/big"

	"github.com/jmoiron/sqlx"
)

const (
	//Returns a result if the username/password combination is correct and the user account is current.
	//sqlGetUserSalt   = `SELECT ID, User_name, Salt FROM st_user WHERE User_name=? and user_account_status_id=1`
	sqlGetUserSalt = `SELECT ID, username, user_account_status_id, salt FROM st_users WHERE username=$1` //The controller needs to check the user_account_status_id to make sure the auth process should proceed.
	//sqlCheckUserAuth = `SELECT ID, User_name, Salt FROM st_user WHERE User_name=? and Password=? and user_account_status_id=1`
	sqlCheckUserAuth = `SELECT ID, username, Salt FROM st_users WHERE username=$1 and user_account_status_id=$2`
	//sqlPutUserAuth   = `UPDATE st_user SET Password=?, Salt=?, Verifier=? WHERE ID=?`
	sqlUserAuthUpdate = `UPDATE st_users SET salt=$1, verifier=$2 WHERE id=$3`
	//sqlGetUserAuth   = `SELECT ID, User_name, Email, Salt, Verifier FROM st_user WHERE User_name=? and user_account_status_id=1`
	sqlGetUserAuth  = `SELECT ID, username, name, email, user_account_status_id, salt, verifier FROM st_users WHERE username=$1` //The controller needs to check the user_account_status_id to make sure the auth process should proceed.
	sqlGetAdminList = `SELECT su.ID, su.username, su.name, su.email
		FROM st_users su JOIN st_user_group sug ON sug.user_id = su.id
		WHERE su.user_account_status_id=$2 AND sug.group_id=$1`
)

// CheckPassword checks that the users Auth info matches the Auth info stored in the DB
// Need to depreciate this ??????????????? why??????????????? it is currently used by ctrlMainLogin
func CheckUserAuthXX(debugStr string, Db *sqlx.DB, username, password string) (models.User, error) {
	var err error
	var result models.User

	//err = r.DBConn.QueryRow(sqlCheckUserAuth, username, password).Scan(&result.ID, &result.UserName, &result.Salt)
	err = Db.QueryRow(sqlCheckUserAuth, username, handlerUserAccountStatus.AccountActive).Scan(&result.ID, &result.Username, &result.Salt)
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %+v", debugTag+"CheckPassword()2 ", "err =", err, "result =", result, "DB =", Db.DB)
		return models.User{}, err //password check failed
	}
	//return result, nil //nil error = password check succeeded, return User.ID, User.UserName
	return result, errors.New("Depreciated")
}

// UserAuthUpdate stores the user auth info in the user table
// Need to depreciate this ??????????????? why??????????????? it is currently used in by the following...
// handler\rest\ctrlAuth\ctrlAuthRegister.go:107:14: h.srvc.PutUserAuth undefined (type *store.Service has no field or method PutUserAuth)
// handler\rest\ctrlAuth\ctrlAuthReset.go:127:14: h.srvc.PutUserAuth undefined (type *store.Service has no field or method PutUserAuth)
func UserAuthUpdate(debugStr string, Db *sqlx.DB, user models.User) error {
	var err error
	v, err := user.Verifier.GobEncode() //Might need to do the encoding here
	log.Printf("%v %v %+v %v %+v %v %+v", debugTag+"UserAuthUpdate()1 ", "user =", user, "err =", err, "v =", v)

	result, err := Db.Exec(sqlUserAuthUpdate, user.Salt, v, user.ID)
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %+v", debugTag+"UserAuthUpdate()2 ", "err =", err, "result =", result, "DB =", Db.DB)
		return err //Auth set failed
	}
	return nil //Auth set succeeded
	//return errors.New("Depreciated")
}

// GetUserAuth retrieves the user auth info from the user table
func GetUserAuth(debugStr string, Db *sqlx.DB, username string) (models.User, error) {
	var err error
	//var result User
	var v []byte
	result := &models.User{
		Verifier: &big.Int{},
	}

	err = Db.QueryRow(sqlGetUserAuth, username).Scan(&result.ID, &result.Username, &result.Name, &result.Email, &result.AccountStatusID, &result.Salt, &v) //??????? Validator might fail, so we will need to post process it ???????
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %+v", debugTag+"GetUserAuth()2 ", "err =", err, "result =", result, "DB =", Db.DB)
		return models.User{}, err //get UserAuth failed
	}

	err = result.Verifier.GobDecode(v) //Decode the verifier
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %+v", debugTag+"GetUserAuth()3 ", "err =", err, "result =", result, "v =", v)
		return models.User{}, err //get UserAuth failed
	}

	//log.Printf("%v %v %v %v %+v %v %+v", debugTag+"GetUserAuth()4 ", "err =", err, "result =", result, "v =", v)

	return *result, nil //return UserAuth info
}

// GetUserSalt gets the Auth salt for the specified username
func GetUserSalt(debugStr string, Db *sqlx.DB, username string) (models.User, error) {
	var err error
	var result models.User

	err = Db.QueryRow(sqlGetUserSalt, username).Scan(&result.ID, &result.Username, &result.AccountStatusID, &result.Salt)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %v %v %+v %v %+v", debugTag+"GetUserSalt()2 ", "err =", err, "username =", username, "sqlGetUserSalt =", sqlGetUserSalt, "result =", result, "DB =", Db.DB)
		return models.User{}, err //GetSalt failed
	}
	return result, nil //return User.ID, User.UserName, User.Salt
}

// GetAdminList retrieves the user details (email address, etc) for the specified group
func GetAdminList(debugStr string, Db *sqlx.DB, groupID int64) ([]models.User, error) {
	var err error
	var result models.User
	var list []models.User

	rows, err := Db.Query(sqlGetAdminList, groupID, handlerUserAccountStatus.AccountActive)
	if err != nil {
		log.Println(debugTag+"GetAdminList()1 - ", "groupID =", groupID, "sqlGetAdminList =", sqlGetAdminList, "err =", err)
		//log.Fatal(err)
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&result.ID, &result.Username, &result.Name, &result.Email)
		if err != nil {
			log.Println(debugTag+"GetAdminList()2 - ", "sqlGetAdminList =", sqlGetAdminList, "result =", result, "err =", err)
			//log.Fatal(err)
			return nil, err
		}
		list = append(list, result) //append each result to the results array.
	}
	return list, nil
}
