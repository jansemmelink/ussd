package ussd

import (
	"context"

	"github.com/pkg/errors"
)

func Call(ctx context.Context, caller Caller, next Item) (Item, string, error) {
	c := call{}

	//send the request
	if err := Caller.Exec(ctx); err != nil {
		//failed to send the request
		ctx.Set("error", errors.Wrapf(err, "failed to send request"))
		return next.Exec()
	}

	//wait for response or timeout
	//todo: ideally in any instance, so timer must be global waiting for response

	
	???

	return c
}

type call struct {
}
