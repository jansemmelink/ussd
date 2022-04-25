package ussd

import "regexp"

type Service interface {
	Start(s Session, code string) (Session, error)
}

var (
	serviceByCode   = map[string]Service{}
	serviceByPrefix = map[string]Service{}
	serviceByRegex  = map[regexp.Regexp]Service{}
)
