package ms

import (
	"reflect"
	"regexp"

	"bitbucket.org/vservices/utils/v4/errors"
)

type Service struct {
	operByName map[string]Operation
}

func NewService() Service {
	return Service{
		operByName: map[string]Operation{},
	}
}

const operNamePattern = `[a-z][a-z0-9_]*[a-z0-9]`

var operNameRegex = regexp.MustCompile("^" + operNamePattern + "$")

func (s Service) Handle(operName string, fnc interface{}) Service {
	if !operNameRegex.MatchString(operName) {
		panic(errors.Errorf("invalid oper name(%s)", operName))
	}
	if _, ok := s.operByName[operName]; ok {
		panic(errors.Errorf("duplicate service operation name(%s)", operName))
	}
	if fnc == nil {
		panic(errors.Errorf("oper(%s).fnc==nil", operName))
	}
	fncType := reflect.TypeOf(fnc)
	if fncType.Kind() != reflect.Func {
		panic(errors.Errorf("oper(%s).fnc=%T is not a function", operName, fnc))
	}

	//fnc must:
	//	arg[0] == context.Context
	//	arg[1] == request type or omitted
	//	result[last] = error
	//	result[last-1] = optional response type only when return 2 results
	o := Operation{
		name:     operName,
		reqType:  nil,
		resType:  nil,
		fncValue: reflect.ValueOf(fnc),
	}
	if fncType.NumIn() > 2 {
		panic(errors.Errorf("open(%s).fnt=%T takes more than 2 arguments", operName, fnc))
	}
	if fncType.NumIn() == 2 {
		o.reqType = fncType.In(1)
	}
	if fncType.NumOut() > 2 {
		panic(errors.Errorf("open(%s).fnt=%T returns more than 2 results", operName, fnc))
	}
	if fncType.NumOut() == 2 {
		o.resType = fncType.Out(0)
	}
	s.operByName[operName] = o
	return s
} //Handle()

func (s Service) GetOper(name string) (Operation, bool) {
	if o, ok := s.operByName[name]; ok {
		return o, true
	}
	return Operation{}, false
}

type Validator interface {
	Validate() error
}
