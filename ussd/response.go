package ussd

import (
	"fmt"
	"strings"

	"bitbucket.org/vservices/utils/v4/errors"
)

type Response struct {
	Type    ResponseType `json:"type"`
	Message string       `json:"message"`
}

func (res Response) Validate() error {
	if res.Message == "" {
		return errors.Errorf("missing text")
	}
	if _, ok := resTypeString[res.Type]; !ok {
		return errors.Errorf("invalid type:%d", res.Type)
	}
	return nil
}

type ResponseType int

const (
	ResponseTypeRedirect ResponseType = iota
	ResponseTypeResponse
	ResponseTypeRelease
)

var (
	resTypeString = map[ResponseType]string{
		ResponseTypeRedirect: "REDIRECT",
		ResponseTypeResponse: "RESPONSE",
		ResponseTypeRelease:  "RELEASE",
	}
	resTypeValue = map[string]ResponseType{}
)

func init() {
	for t, s := range resTypeString {
		resTypeValue[s] = t
	}
}

func (t ResponseType) String() string {
	if s, ok := resTypeString[t]; ok {
		return s
	}
	return fmt.Sprintf("unknown ussd.ResponseType(%d)", t)
}

func (t *ResponseType) Parse(s string) error {
	if v, ok := resTypeValue[strings.ToLower(s)]; ok {
		*t = v
		return nil
	}
	return errors.Errorf("unknown ussd.ResponseType(%d)", t)
}

func (t *ResponseType) UnmarshalJSON(v []byte) error {
	s := string(v)
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return errors.Errorf("ResponseType(%s) expected quoted value", s)
	}
	if err := t.Parse(s[1 : len(s)-2]); err != nil {
		return errors.Wrapf(err, "unable to unmarshal ResponseType(%s)", s)
	}
	return nil
}

func (t ResponseType) MarshalJSON() ([]byte, error) {
	if s, ok := resTypeString[t]; ok {
		return []byte("\"" + s + "\""), nil
	}
	return nil, errors.Errorf("unknown ussd.ResponseType(%d)", t)
}
