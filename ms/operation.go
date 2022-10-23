package ms

import (
	"reflect"
)

type Operation struct {
	name     string
	reqType  reflect.Type
	resType  reflect.Type
	fncValue reflect.Value
}

func (o Operation) ReqType() reflect.Type {
	return o.reqType
}

func (o Operation) ResType() reflect.Type {
	return o.resType
}

func (o Operation) FncValue() reflect.Value {
	return o.fncValue
}

// func (o Operation) Run(ctx context.Context) (res interface{}, err error) {
// 	//return o.ResType()

// 	//need responder in session data to use async from session data when called later
// 	return nil, errors.Errorf("NYI")
// }
