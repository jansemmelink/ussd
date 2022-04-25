package ussd

import "context"

//Item is any type of USSD service processing step
//Return:
//	"",   <next>, nil   (next!=current) to jump to <next> immediately and call <next>.Exec()
//  "",   <next>, nil   (next==current) to waiting for a reply then call next.HandleReply()
//	text, <next>, nil   (next==current) to wait for user input then call next.HandleInput()
//	text, nil,    nil   for final response
//	"",   nil,    <err> to end with system error
type Item interface {
	ID() string
	Exec(ctx context.Context) (text string, next Item, err error)
}

type ItemWithInputHandler interface {
	Item
	HandleInput(ctx context.Context, input string) (text string, next Item, err error) //handles user dialed USSD string or user input after menu/prompt
}

type ItemWithReplyHandler interface {
	Item
	HandleReply(ctx context.Context, reply interface{}) (text string, next Item, err error) //handles reply from a service operation
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
