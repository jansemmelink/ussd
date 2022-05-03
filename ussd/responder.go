package ussd

import (
	"context"
	"fmt"
	"sync"
)

//Responder sends a cont/final response to the user
//different responders can be user, based on how you got the user input
type Responder interface {
	ID() string
	Respond(ctx context.Context, key interface{}, res Response) error
}

var (
	responderMutex sync.Mutex
	responderByID  = map[string]Responder{}
)

func AddResponder(r Responder) {
	responderMutex.Lock()
	defer responderMutex.Unlock()
	if _, ok := responderByID[r.ID()]; ok {
		panic(fmt.Sprintf("responder(%s) already registered", r.ID()))
	}
	responderByID[r.ID()] = r
}
