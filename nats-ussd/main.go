package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bitbucket.org/vservices/ms-vservices-ussd/examples/pcm"
	"bitbucket.org/vservices/ms-vservices-ussd/ms"
	"bitbucket.org/vservices/ms-vservices-ussd/ms/nats"
	httpSessionsClient "bitbucket.org/vservices/ms-vservices-ussd/rest-sessions/client"
	"bitbucket.org/vservices/ms-vservices-ussd/ussd"
	"bitbucket.org/vservices/utils/v4/errors"
	"bitbucket.org/vservices/utils/v4/logger"
	datatype "bitbucket.org/vservices/utils/v4/type"
)

var log = logger.NewLogger()

func main() {
	//define session storage
	ussd.SetSessions(httpSessionsClient.New("http://localhost:8100"))

	//define NATS interface
	nc := nats.Config{
		Domain:             "ussd",
		Url:                "nats://localhost:4222",
		Secure:             false,
		InsecureSkipVerify: true,
		//Username:           "",
		//Password: "",
		MaxReconnects: 10,
		ReconnectWait: datatype.Duration(time.Second * 5),
	}
	commsHandler, err := nc.New()
	if err != nil {
		panic(fmt.Sprintf("cannot create comms handler: %+v", err))
	}

	//responder sends responses
	... no longer needed? remove request and wait ... r := responder{ch: commsHandler}
	ussd.AddResponder(r)

	s := service{ch: commsHandler}
	if err := commsHandler.Run(s); err != nil {
		panic(err)
	}
}

type service struct {
	ch ms.Handler
}

func (s service) handleRequest(data []byte, replyAddress string) {
	log.Debugf("Received %s", string(data))
	var err error
	defer func() {
		if err != nil {
			log.Errorf("DEFER err: %+v", err)
			if replyAddress != "" {
				log.Errorf("DEFER reply to %s", replyAddress)
				res := Message{
					Header: MessageHeader{
						Timestamp: time.Now().Local().Format(tsformat),
						Result: &MessageHeaderResult{
							Code:        -1,
							Description: "failed",
							Details:     fmt.Sprintf("%+v", err),
						},
					},
				}
				jsonRes, _ := json.Marshal(res)
				s.ch.Send(nil, replyAddress, jsonRes)
			}
		}
	}()

	var m Message
	err = json.Unmarshal(data, &m)
	if err != nil {
		err = errors.Wrapf(err, "cannot unmarshal JSON", string(data))
		return
	}

	log.Debugf("RECV: %+v", m)
	if m.Header.Result != nil || m.Response != nil {
		err = errors.Errorf("discard response message on request subject")
		return
	}
	if m.Request == nil || m.Request.Msisdn == "" || m.Request.Text == "" {
		err = errors.Errorf("discard invalid request (missing msisdn or text): %+v", m.Request)
		return
	}

	ussdData := map[string]interface{}{
		"responder_key": replyAddress,
	}
	ctx := context.Background()
	id := "nats:" + m.Request.Msisdn
	switch m.Request.Type {
	case "START":
		log.Debugf("Starting")
		if err = ussd.Start(ctx, id, ussdData, pcm.Item() /*initItem*/, m.Request.Text, responder{ch: s.ch}, m.Header.ReplyAddress); err != nil {
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

// if len(subject) <= 0 {
// 	subject = strings.Replace(message.Header.Provider.Name, "/", ".", -1)
// 	subject = strings.Replace(subject, ".", "", 1)
// } // if not subject

type responder struct {
	ch ms.Handler
}

func (r responder) ID() string { return "nats" }

func (r responder) Respond(ctx context.Context, key interface{}, res ussd.Response) error {
	log.Debugf("Respond(%v, %s, %s)...", key, res.Type, res.Text)
	subject := key.(string)
	resMsg := Message{
		Header:   MessageHeader{},
		Response: &res,
	}
	jsonRes, _ := json.Marshal(resMsg)
	if err := r.ch.Send(nil, subject, jsonRes); err != nil {
		log.Errorf("failed to respond to subject(%s): %+v", subject, err)
	}
	return nil
}
