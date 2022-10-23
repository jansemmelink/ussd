package ussd

import (
	"context"

	"bitbucket.org/vservices/ms-vservices-ussd/examples/pcm"
	"bitbucket.org/vservices/ms-vservices-ussd/ms"
	"bitbucket.org/vservices/utils/v4/errors"
)

type StartRequest struct {
	ID           string                 `json:"id" doc:"Unique session ID, typically made up of source and user e.g. 'sigtran:27821234567'. It could be UUID as well, but using same string always for a user ensures the user can only have one session at any time and starting new session will delete any old session."`
	Data         map[string]interface{} `json:"data" doc:"Initial data values to set in the new session"`
	ItemID       string                 `json:"item_id" doc:"ID of USSD item to start the session. It must be a server side item to return next item, typically a ussd router to process the dialed USSD string."`
	Input        string                 `json:"input" doc:"User input is initially dialed USSD string for start, and prompt/menu input for continuation."`
	ResponderID  string                 `json:"responder_id" doc:"Identifies the responder to use"`
	ResponderKey string                 `json:"responder_key" doc:"Key given to the responder to send to the correct user"`
}

type ContinueRequest StartRequest

type AbortRequest struct {
	ID string `json:"id" doc:"Unique session ID also used in start/continue Request."`
}

func NewService() ms.Service {
	return ms.NewService().
		Handle(RequestTypeRequest.String(), s.HandleStart).
		Handle(RequestTypeResponse.String(), s.HandleInput).
		Handle(RequestTypeRelease.String(), s.HandleAbort)
}

func (s Service) HandleStart(ctx context.Context) error {
	//session ID on this provider will only be specified if consumer is continuing
	//on an existing session
	// sid := m.Header.Provider.Sid //"ussd:" + m.Request.Msisdn
	// ...sid.... ussd start open must expect "" while continue/abort expects an id
	// or ussd still create/get session because other services does not need sesison...
	// so ignore sid in consumer and provider? I think so...

	switch m.Request.Type {
	case "START":
		log.Debugf("Starting")
		if err = ussd.Start(ctx, id, ussdData, pcm.Item() /*initItem*/, m.Request.Message, responder{ch: s.ch}, m.Header.ReplyAddress); err != nil {
			err = errors.Wrapf(err, "failed to start USSD")
			return
		}

	case "CONTINUE":
		if err = ussd.UserInput(ctx, id, ussdData, m.Request.Text, responder{ch: s.ch}, m.Header.ReplyAddress); err != nil {
			err = errors.Wrapf(err, "failed to continue USSD")
			return
		}

	case "ABORT":
		if err = ussd.UserAbort(ctx, id); err != nil {
			err = errors.Wrapf(err, "failed to abort USSD")
			return
		}

	default:
		err = errors.Errorf("invalid request (unknown type): %+v", m.Request)
		return
	}

	//handling started successfully
	//responder will take care of response
	log.Debugf("processing done")
	return

}
