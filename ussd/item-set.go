package ussd

import (
	"context"

	"github.com/google/uuid"
)

func Set(name string, value interface{}) Item {
	return set{
		id:    uuid.New().String(),
		name:  name,
		value: value,
	}
}

type set struct {
	id    string
	name  string
	value interface{} //todo: replace with expression from session data
}

func (set set) ID() string {
	return set.id
}

func (set set) Exec(ctx context.Context) ([]Item, error) {
	s := ctx.Value(CtxSession{}).(Session)
	s.Set(set.name, set.value)
	return nil, nil
}
