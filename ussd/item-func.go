package ussd

import (
	"context"

	"bitbucket.org/vservices/utils/v4/errors"
)

func NewFunc(id string, fnc func(context.Context) error) ItemSvcExec {
	return ussdFunc{
		id:  id,
		fnc: fnc,
	}
}

type ussdFunc struct {
	id  string
	fnc func(context.Context) error
}

func (f ussdFunc) ID() string { return f.id }

func (f ussdFunc) Exec(ctx context.Context) ([]Item, error) {
	if err := f.fnc(ctx); err != nil {
		return nil, errors.Wrapf(err, "failed in %s", f.id)
	}
	return nil, nil
}
