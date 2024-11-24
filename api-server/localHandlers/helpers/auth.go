package helpers

import (
	"api-server/v2/app/appCore"
	"log"
)

type AccessLevel int

const (
	AccessLevelNone AccessLevel = iota
	AccessLevelSelect
	AccessLevelInsert
	AccessLevelUpdate
	AccessLevelDelete
)

const (
	sqlAccessCheck = `SELECT DISTINCT eal.ID, eal.Name
	FROM st_user stu
	 JOIN st_user_group stug ON sug.User_ID=stu.ID
	 JOIN st_group_resource stgr ON sgr.Group_ID=sug.Group_ID
	 JOIN et_resource etr ON etr.ID=sgr.Resource_ID
	 JOIN et_access_level etal ON etal.ID = sgr.Access_level_ID 
	WHERE stu.ID=$1 AND etr.ID=$2`
)

func AccessCheck(userID int, resourceID int, accessLevel AccessLevel, core *appCore.Config) error {
	var err error
	var result int
	err = core.Db.QueryRow(sqlAccessCheck, userID, resourceID, accessLevel).Scan(&result)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %+v", debugTag+"AccessCheck1", "err =", err, "sqlAccessCheck =", sqlAccessCheck, "result =", result)
		return err
	}
	return nil
}
