package handlerAuth

import (
	"api-server/v2/models"
	"errors"
	"log"
	"math/big"
)

//const debugTag = "mdlUser."

//**********************************************************
// Project Manager Database: Structures and methods
//**********************************************************

// UserCheckRepo ??
//type UtilsRepo struct {
//	DBConn *dataStore.DB
//}

const (
	//Returns a result if the username/password combination is correct and the user account is current.
	//sqlGetUserSalt   = `SELECT ID, User_name, Salt FROM st_user WHERE User_name=? and User_status_ID=1`
	sqlGetUserSalt = `SELECT ID, User_name, User_status_ID, Salt FROM st_user WHERE User_name=$1` //The controller needs to check the User_status_ID to make sure the auth process should proceed.
	//sqlCheckUserAuth = `SELECT ID, User_name, Salt FROM st_user WHERE User_name=? and Password=? and User_status_ID=1`
	sqlCheckUserAuth = `SELECT ID, User_name, Salt FROM st_user WHERE User_name=$1 and User_status_ID=1`
	//sqlPutUserAuth   = `UPDATE st_user SET Password=?, Salt=?, Verifier=? WHERE ID=?`
	sqlPutUserAuth = `UPDATE st_user SET Salt=$1, Verifier=$2 WHERE ID=$3`
	//sqlGetUserAuth   = `SELECT ID, User_name, Email, Salt, Verifier FROM st_user WHERE User_name=? and User_status_ID=1`
	sqlGetUserAuth  = `SELECT ID, User_name, Display_name, Email, User_status_ID, Salt, Verifier FROM st_user WHERE User_name=$1` //The controller needs to check the User_status_ID to make sure the auth process should proceed.
	sqlGetAdminList = `SELECT su.ID, su.User_name, su.Display_name, su.Email
FROM st_user su JOIN st_user_group sug ON sug.User_ID = su.ID
WHERE su.User_status_ID=1 AND sug.Group_ID=$1`
)

// CheckPassword checks that the users Auth info matches the Auth info stored in the DB
// Need to depreciate this ??????????????? why??????????????? it is currently used by ctrlMainLogin
func (h *Handler) CheckUserAuth(username, password string) (models.User, error) {
	var err error
	var result models.User

	//err = r.DBConn.QueryRow(sqlCheckUserAuth, username, password).Scan(&result.ID, &result.UserName, &result.Salt)
	err = h.appConf.Db.QueryRow(sqlCheckUserAuth, username).Scan(&result.ID, &result.Username, &result.Salt)
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %+v", debugTag+"UserUtilsRepo.CheckPassword()2 ", "err =", err, "result =", result, "r.DBConn.DB =", h.appConf.Db.DB)
		return models.User{}, err //password check failed
	}
	//return result, nil //nil error = password check succeeded, return User.ID, User.UserName
	return result, errors.New("Depreciated")
}

// PutUserAuth stores the user auth info in the user table
// Need to depreciate this ??????????????? why??????????????? it is currently used in by the following...
// handler\rest\ctrlAuth\ctrlAuthRegister.go:107:14: h.srvc.PutUserAuth undefined (type *store.Service has no field or method PutUserAuth)
// handler\rest\ctrlAuth\ctrlAuthReset.go:127:14: h.srvc.PutUserAuth undefined (type *store.Service has no field or method PutUserAuth)
func (h *Handler) UserAuthUpdate(user models.User) error {
	var err error
	v, _ := user.Verifier.GobEncode() //Might need to do the encoding here

	result, err := h.appConf.Db.Exec(sqlPutUserAuth, user.Salt, v, user.ID)
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %+v", debugTag+"UserUtilsRepo.PutUserAuth()2 ", "err =", err, "result =", result, "r.DBConn.DB =", h.appConf.Db.DB)
		return err //Auth set failed
	}
	return nil //Auth set succeeded
	//return errors.New("Depreciated")
}

// GetUserAuth retrieves the user auth info from the user table
func (h *Handler) GetUserAuth(username string) (models.User, error) {
	var err error
	//var result User
	var v []byte
	result := &models.User{
		Verifier: &big.Int{},
	}

	err = h.appConf.Db.QueryRow(sqlGetUserAuth, username).Scan(&result.ID, &result.Username, &result.Name, &result.Email, &result.AccountStatusID, &result.Salt, &v) //??????? Validator might fail, so we will need to post process it ???????
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %+v", debugTag+"UserUtilsRepo.GetUserAuth()2 ", "err =", err, "result =", result, "r.DBConn.DB =", h.appConf.Db.DB)
		return models.User{}, err //get UserAuth failed
	}

	err = result.Verifier.GobDecode(v) //Decode the verifier
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %+v", debugTag+"UserUtilsRepo.GetUserAuth()3 ", "err =", err, "result =", result, "v =", v)
		return models.User{}, err //get UserAuth failed
	}

	return *result, nil //return UserAuth info
}

// GetUserSalt gets the Auth salt for the specified username
func (h *Handler) GetUserSalt(username string) (models.UserAuth, error) {
	var err error
	var result models.UserAuth

	err = h.appConf.Db.QueryRow(sqlGetUserSalt, username).Scan(&result.ID, &result.Username, &result.AccountStatusID, &result.Salt)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %v %v %+v %v %+v", debugTag+"UserUtilsRepo.GetUserSalt()2 ", "err =", err, "username =", username, "sqlGetUserSalt =", sqlGetUserSalt, "result =", result, "r.DBConn.DB =", h.appConf.Db.DB)
		return models.UserAuth{}, err //GetSalt failed
	}
	return result, nil //return User.ID, User.UserName, User.Salt
}

// GetAdminList retrieves the user details (email address, etc) for the specified group
func (h *Handler) GetAdminList(groupID int64) ([]models.User, error) {
	var err error
	var result models.User
	var list []models.User

	rows, err := h.appConf.Db.Query(sqlGetAdminList, groupID)
	if err != nil {
		log.Println(debugTag+"UserUtilsRepo.GetAdminList()1 - ", "groupID =", groupID, "sqlGetAdminList =", sqlGetAdminList, "err =", err)
		//log.Fatal(err)
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&result.ID, &result.Username, &result.Name, &result.Email)
		if err != nil {
			log.Println(debugTag+"UserUtilsRepo.GetAdminList()2 - ", "sqlGetAdminList =", sqlGetAdminList, "result =", result, "err =", err)
			//log.Fatal(err)
			return nil, err
		}
		list = append(list, result) //append each result to the results array.
	}
	return list, nil
}
