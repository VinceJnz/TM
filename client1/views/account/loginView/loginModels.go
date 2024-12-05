package loginView

import "math/big"

//SrpItem contains the user SRP info
//type SrpItem struct {
//	Salt     []byte   `json:"Salt"`     //Not user editable
//	Verifier *big.Int `json:"Verifier"` //Not user editable
//	Password string   `json:"Password"` //srp means this is not longer needed.
//	//Server   ServerVerify
//	//Client   ClientVerify
//}

//ServerVerify contains the verify info sent from the server
type ServerVerify struct {
	B     *big.Int `json:"B"`
	Proof []byte   `json:"Proof"`
	Token string   `json:"Token"`
}

//ClientVerify contains the clinet SRP verify info and is sent to the server
type ClientVerify struct {
	UserName string `json:"UserName"`
	Proof    []byte `json:"Proof"`
	Token    string `json:"Token"`
}

//MenuUserItem contains the basic user info for driving the display of the client menu
type MenuUser struct {
	UserID    int    `json:"user_id"`
	Name      string `json:"name"`
	Group     string `json:"group"`
	AdminFlag bool   `json:"admin_flag"`
}

//MenuUserItem contains a list of valid menu items to display
type MenuItem struct {
	UserID   int    `json:"user_id"`
	Resource string `json:"resource"`
}

type MenuList []MenuItem

type UpdateMenu struct {
	MenuUser MenuUser
	MenuList MenuList
}
