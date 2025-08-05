package viewAccountModels

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
