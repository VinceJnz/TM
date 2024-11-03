package viewAccountModels

import "math/big"

type ItemState int

const (
	ItemStateNone     ItemState = iota
	ItemStateFetching           //ItemState = 1
	ItemStateEditing            //ItemState = 2
	ItemStateAdding             //ItemState = 3
	ItemStateSaving             //ItemState = 4
	ItemStateDeleting           //ItemState = 5
	ItemStateSubmitted
)

type ViewState int

const (
	ViewStateNone ViewState = iota
	ViewStateBlock
)

type RecordState int

const (
	RecordStateReloadRequired RecordState = iota
	RecordStateCurrent
)

type MessageStatus int

const (
	MessageStatusEmpty MessageStatus = iota
	MessageStatusInfo
	MessageStatusWarning
	MessageStatusError
)

type Message struct {
	Id     int64
	Text   string
	Status MessageStatus
}

// ServerVerify contains the verify info sent from the server
type ServerVerify struct {
	B     *big.Int `json:"B"`
	Proof []byte   `json:"Proof"`
	Token string   `json:"Token"`
}

// ClientVerify contains the clinet SRP verify info and is sent to the server
type ClientVerify struct {
	UserName string `json:"UserName"`
	Proof    []byte `json:"Proof"`
	Token    string `json:"Token"`
}

// SrpItem contains the user SRP info
type SrpItem struct {
	//Item
	Salt     []byte   `json:"Salt"`     //Not user editable
	Verifier *big.Int `json:"Verifier"` //Not user editable
	Password string   `json:"Password"` //srp means this is not longer needed.
	Server   ServerVerify
	Client   ClientVerify
}
