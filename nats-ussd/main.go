package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bitbucket.org/vservices/ms-vservices-ussd/comms"
	"bitbucket.org/vservices/ms-vservices-ussd/comms/nats"
	"bitbucket.org/vservices/ms-vservices-ussd/examples/pcm"
	httpSessionsClient "bitbucket.org/vservices/ms-vservices-ussd/rest-sessions/client"
	"bitbucket.org/vservices/ms-vservices-ussd/ussd"
	"bitbucket.org/vservices/utils/v4/errors"
	"bitbucket.org/vservices/utils/v4/logger"
	datatype "bitbucket.org/vservices/utils/v4/type"
)

var log = logger.NewLogger()

func main() {
	ussd.SetSessions(httpSessionsClient.New("http://localhost:8100"))
	nc := nats.Config{
		Name:               "ussd",
		Url:                "nats://localhost:4222",
		Secure:             false,
		InsecureSkipVerify: true,
		Username:           "",
		//Password: "",
		MaxReconnects: 10,
		ReconnectWait: datatype.Duration(time.Second * 5),
	}
	commsHandler, err := nc.New()
	if err != nil {
		panic(fmt.Sprintf("cannot create comms handler: %+v", err))
	}

	r := responder{ch: commsHandler}
	ussd.AddResponder(r)

	s := service{ch: commsHandler}
	if err := commsHandler.Subscribe("ussd", false, s.handleRequest); err != nil {
		panic(fmt.Sprintf("failed to subscriber: %+v", err))
	}
	x := make(chan bool)
	<-x
}

// var initItem ussd.ItemWithInputHandler

// func init() {
// 	menu123 := ussd.NewMenu("123", "*** MAIN MENU ***").
// 		With("one", nil).
// 		With("two", nil).
// 		With("three", nil).
// 		With("four", nil).
// 		With("Exit", ussd.NewFinal("exit", "Goodbye."))

// 	initItem = ussd.NewRouter("mainRouter").
// 		WithCode("*123#", menu123)
// }

type service struct {
	ch comms.Handler
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

type Message struct {
	Header   MessageHeader  `json:"header"`
	Request  *ussdRequest   `json:"request,omitempty"`
	Response *ussd.Response `json:"response,omitempty"`
}

type MessageHeader struct {
	Timestamp    string               `json:"timestamp"`
	TTL          int                  `json:"ttl"`
	ReplyAddress string               `json:"reply_address"`
	Result       *MessageHeaderResult `json:"result,omitempty"`
}

type MessageHeaderResult struct {
	Code        int    `json:"code"`
	Description string `json:"description,omitempty"`
	Details     string `json:"details,omitempty"`
}

type ussdRequest struct {
	Msisdn string `json:"msisdn"`
	Text   string `json:"text"`
	Type   string `json:"type"`
}

type ussdResponse struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

const tsformat = "2006-01-02 15:04:05.000"

// if len(subject) <= 0 {
// 	subject = strings.Replace(message.Header.Provider.Name, "/", ".", -1)
// 	subject = strings.Replace(subject, ".", "", 1)
// } // if not subject

type responder struct {
	ch comms.Handler
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
