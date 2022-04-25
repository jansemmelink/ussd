package ussd

import (
	"context"
)

type Service interface {
	//execute the item with optional user input
	//respond with either:
	//	next if started an operation and need to wait for a reply, then this is next item to continue on
	//	res  if need to respond to the user
	//	err  if failed and must terminate on some system error
	Exec(ctx context.Context, user string) (next Item, res *Response, err error)
}

type Response struct {
	Type Type
	Text string
}
