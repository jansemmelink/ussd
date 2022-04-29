package ussd

import "context"

//Item is any type of USSD service processing step
type Item interface {
	ID() string
}

//ItemStep is any item that does not change what happens next
type ItemStep interface {
	Item
	Exec(ctx context.Context) (err error)
}

//ItemRoute is any item that can change what happens next
type ItemRoute interface {
	Item
	Exec(ctx context.Context) (next []Item, err error)
}

//item that display to the user (question, menu or final)
type ItemUser interface {
	Item
	Render() string
}

//ItemFinal display final message and terminate the session
type ItemFinal interface {
	ItemUser
}

//ItemPrompt ask a question and store the answer (it is a step, does not change next)
//by the time Exec() is called, session["input"] contains user input and is cleared
type ItemPrompt interface {
	ItemStep
	ItemUser
}

//ItemMenu display a menu and the selection determine next
//by the time Exec() is called, session["input"] contains user input and is cleared
type ItemMenu interface {
	ItemRoute
	ItemUser
}

//ItemReqAndWait is a step where item has to wait for a reply before proceeding
type ItemReqWait interface {
	ItemStep
	Request(ctx context.Context) error
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
