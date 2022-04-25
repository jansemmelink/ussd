package ussd

import "context"

//Item in a USSD service must either jump to another Item or return a user prompt or fail, only one of those
type Item interface {
	Exec(ctx context.Context) (next Item, prompt string, err error)
}
