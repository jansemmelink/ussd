package ussd

import (
	"fmt"
	"strings"

	"bitbucket.org/vservices/utils/v4/errors"
)

type Request struct {
	Type   RequestType `json:"type"`
	Msisdn string      `json:"msisdn"`
	Text   string      `json:"text"`
}

func (req Request) Validate() error {
	if req.Msisdn == "" {
		return errors.Errorf("missing msisdn")
	}
	if req.Text == "" {
		return errors.Errorf("missing text")
	}
	if _, ok := reqTypeString[req.Type]; !ok {
		return errors.Errorf("invalid type:%d", req.Type)
	}
	return nil
}

type RequestType int

const (
	RequestTypeRequest RequestType = iota
	RequestTypeResponse
	RequestTypeRelease
)

var (
	reqTypeString = map[RequestType]string{
		RequestTypeRequest:  "REQUEST",  //user request to begin a new session
		RequestTypeResponse: "RESPONSE", //user input that continues a session
		RequestTypeRelease:  "RELEASE",  //user request to abort the session
	}
	reqTypeValue = map[string]RequestType{}
)

func init() {
	for t, s := range reqTypeString {
		reqTypeValue[s] = t
	}
}

func (t RequestType) String() string {
	if s, ok := reqTypeString[t]; ok {
		return s
	}
	return fmt.Sprintf("unknown ussd.RequestType(%d)", t)
}

func (t *RequestType) Parse(s string) error {
	if v, ok := reqTypeValue[strings.ToLower(s)]; ok {
		*t = v
		return nil
	}
	return errors.Errorf("unknown ussd.RequestType(%d)", t)
}

func (t *RequestType) UnmarshalJSON(v []byte) error {
	s := string(v)
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return errors.Errorf("RequestType(%s) expected quoted value", s)
	}
	if err := t.Parse(s[1 : len(s)-2]); err != nil {
		return errors.Wrapf(err, "unable to unmarshal RequestType(%s)", s)
	}
	return nil
}

func (t RequestType) MarshalJSON() ([]byte, error) {
	if s, ok := reqTypeString[t]; ok {
		return []byte("\"" + s + "\""), nil
	}
	return nil, errors.Errorf("unknown ussd.RequestType(%d)", t)
}
