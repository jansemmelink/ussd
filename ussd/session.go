package ussd

import (
	"time"

	"bitbucket.org/vservices/utils/v4/errors"
)

type Session struct {
	Msisdn  string //session ID
	Start   time.Time
	Service Service
}

type BeginRequest struct {
	Msisdn string
	Imsi   string //optional - depending on HLR config to send/omit
	Code   string //USSD code dialed, e.g. "*123#"
}

//New() handles a BEGIN request
func New(req BeginRequest) (session Session, err error) {
	s := Session{
		Msisdn: req.Msisdn,
		Start:  time.Now(),
	}

	//routing: select a service based on the USSD code
	//start by looking up the exact code match, which uses a map hash
	//and will be the quickest match
	if service, ok := serviceByCode[req.Code]; ok {
		s.Service = service //found exact match
	} else {
		//run through prefix matches, e.g. *123* and *123# both go to xyz
		for prefix, service := range serviceByPrefix {
			if len(req.Code) >= len(prefix) && req.Code[0:len(prefix)] == prefix {
				s.Service = service
				break
			}
		}
	}
	if s.Service == nil {
		for regex, service := range serviceByRegex {
			if regex.MatchString(req.Code) {
				s.Service = service
				break
			}
		}
	}

	if s.Service == nil {
		return Session{}, errors.Errorf("unknown ussd code")
	}

	//now let the service execute
	//the services gets the ussd code, e.g. to extract B-number
	//from PCM request "*140*0821234567#"
	return s.Service.Start(s, req.Code)
}

type ContinueRequest struct {
	Input string //user input
}

func (Session) Continue(req ContinueRequest) {
	//menu
	//prompt
	//assigment
	//if or switch
	//service call
	//HTTP
	//SQL
	//cache get/set
	//script
	//switch language
}

func (Session) Set(name string, value string) error {
	//todo
	return errors.Errorf("NYI")
}

func (Session) Get(name string) (value string, err error) {
	return "", errors.Errorf("NYI")
}
