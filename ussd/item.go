package ussd

import "context"

//Item is any type of USSD service processing step
type Item interface {
	ID() string
}

type ItemSvcExec interface {
	Item
	Exec(ctx context.Context) (nextItems []Item, err error) //err to stop
}

type ItemSvcWait interface {
	Item
	Request(ctx context.Context) (err error)                    //err to stop
	Process(ctx context.Context, value interface{}) (err error) //err to stop
}

type ItemUsr interface {
	Item
	Render(ctx context.Context) string
}

type ItemUsrPrompt interface {
	ItemUsr
	Process(ctx context.Context, input string) (nextItems []Item, err error) //return self to repeat prompt, err to display to user
}

var (
	itemByID = map[string]Item{}
)

func ItemByID(id string) (Item, bool) {
	if item, ok := itemByID[id]; ok {
		return item, true
	}
	return nil, false
}
