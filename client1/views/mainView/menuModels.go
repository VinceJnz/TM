package mainView

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
