package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"bitbucket.org/vservices/ms-vservices-ussd/ms"
	"bitbucket.org/vservices/utils/v4/errors"
	"bitbucket.org/vservices/utils/v4/logger"
	"github.com/nats-io/nats.go"
)

var log = logger.NewLogger()

type handler struct {
	config             Config
	conn               *nats.Conn
	headersSupported   bool
	subscriptionsLock  sync.Mutex
	subscriptions      map[string]*nats.Subscription
	replySubjectPrefix string
	replySubscription  *nats.Subscription
	replyChannelsLock  sync.Mutex
	replyChannels      map[string]chan *nats.Msg
	service            ms.Service
}

func (h *handler) Run(s ms.Service) error {
	h.service = s //must set before subscription to have it in handleRequest
	if err := h.Subscribe(h.config.Domain+".*", false, h.handleRequest); err != nil {
		return errors.Wrapf(err, "failed to subscribe to request subject")
	}

	var err error
	if h.replySubscription, err = h.conn.Subscribe(h.replySubjectPrefix+"*", h.handleReply); err != nil {
		return errors.Wrapf(err, "failed to subscribe to reply subject")
	}

	//todo: graceful shutdown
	log.Debugf("NATS service(%s) running...", h.config.Domain)
	x := make(chan bool)
	<-x
	log.Debugf("NATS service(%s) stopped...", h.config.Domain)
	return nil
}

//Subscribe() to group queue (only one instance get the request) or broadcast
// queue (each instance get it)
func (h *handler) Subscribe(subject string, broadcast bool, callback ms.HandlerFunc) error {
	if h == nil {
		return errors.Errorf("nil.Subscribe()")
	}
	h.subscriptionsLock.Lock()
	defer h.subscriptionsLock.Unlock()
	if _, ok := h.subscriptions[subject]; ok {
		return nil //already subscribed, assuming with same callback
	}
	var subscription *nats.Subscription
	var err error
	if !broadcast {
		subscription, err = h.conn.QueueSubscribe(subject+".*", fmt.Sprintf("Q.%s", subject), func(msg *nats.Msg) {
			callback(msg.Data, msg.Reply)
		})
		if err != nil {
			return errors.Wrapf(err, "queue subscribe(%s) failed", subject)
		}
	} else {
		subscription, err = h.conn.Subscribe(subject+".*", func(msg *nats.Msg) {
			callback(msg.Data, msg.Reply)
		})
		if err != nil {
			return errors.Wrapf(err, "subscribe(%s) failed", subject)
		}
	}
	h.subscriptions[subject] = subscription
	//h.defaultReplyQ = subject + ".reply"
	return nil
} //handler.Subscribe()

func (h *handler) handleRequest(data []byte, replyAddress string) {
	log.Debugf("Received %s", string(data))
	var err error
	var echoRequest interface{}
	var res interface{}
	defer func() {
		resMessage := ms.Message{
			Header: ms.MessageHeader{
				Timestamp: time.Now().Local().Format(ms.TimestampFormat),
			},
			Request:  echoRequest,
			Response: res,
		}
		if err != nil {
			log.Errorf("DEFER err: %+v", err)
			resMessage.Header.Result = &ms.MessageHeaderResult{
				Code:        -1, //todo: get code & code stack from err
				Description: "failed",
				Details:     fmt.Sprintf("%+v", err),
			}
		} else {
			resMessage.Header.Result = &ms.MessageHeaderResult{
				Code:        0,
				Description: "success",
				Details:     "",
			}
		}
		if replyAddress != "" {
			log.Errorf("DEFER reply to %s", replyAddress)
			jsonRes, _ := json.Marshal(resMessage)
			h.Send(nil, replyAddress, jsonRes)
		}
	}()

	var m ms.Message
	if err = json.Unmarshal(data, &m); err != nil {
		err = errors.Wrapf(err, "cannot unmarshal JSON", string(data))
		return
	}

	log.Debugf("RECV: %+v", m)
	if m.Header.Result != nil || m.Response != nil {
		err = errors.Errorf("discard response message on request subject")
		return
	}

	//determine operation name from provider.name="/<domain>/<operName>"
	var operName string
	if parts := strings.SplitN(m.Header.Provider.Name, "/", 3); len(parts) == 3 {
		operName = parts[2]
	} else {
		err = errors.Errorf("provider.name=\"%s\" != \"/<domain>/<oper>\"", m.Header.Provider.Name)
		return
	}
	o, ok := h.service.GetOper(operName)
	if !ok {
		err = errors.Errorf("unknown operName(%s) in provider.name=\"%s\"", operName, m.Header.Provider.Name)
		return
	}

	if m.Header.EchoRequest {
		echoRequest = m.Request
	}

	var reqValuePtr reflect.Value
	if o.ReqType() != nil {
		if m.Request == nil {
			err = errors.Errorf("missing request")
			return
		}
		reqValuePtr = reflect.New(o.ReqType())
		jsonRequest, _ := json.Marshal(m.Request)
		if err = json.Unmarshal(jsonRequest, reqValuePtr.Interface()); err != nil {
			err = errors.Wrapf(err, "failed to decode request into %v", o.ReqType())
			return
		}

		if m.Header.EchoRequest {
			//note: echoes how we interpreted it, not always as it was sent
			//so caller can see our interpretation
			echoRequest = reqValuePtr.Elem().Interface()
		}

		if validator, ok := (reqValuePtr.Interface()).(ms.Validator); ok {
			if err = validator.Validate(); err != nil {
				err = errors.Wrapf(err, "invalid request")
				return
			}
		}
	}

	//has a valid request
	ctx := context.Background()

	//call the operation handler function
	results := o.FncValue().Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(reqValuePtr.Elem().Interface()),
	})

	//get err from last result
	if err, ok := results[len(results)-1].Interface().(error); ok && err != nil {
		err = errors.Wrapf(err, "handler failed")
		return
	}

	if len(results) == 2 {
		res = results[0].Interface()
	}
	return
} //handler.HandleRequest()

//Send() sends a message to Nats on a given subject
func (h *handler) Send(header map[string]string, subject string, data []byte) error {
	if h == nil {
		return errors.Errorf("nil.Send()")
	}
	sendMsg := nats.NewMsg(subject)
	sendMsg.Data = []byte(data)
	if h.headersSupported {
		for n, v := range header {
			sendMsg.Header.Set(n, v)
		}
	}
	if err := h.conn.PublishMsg(sendMsg); err != nil {
		return errors.Wrap(err, "failed to publish message")
	}
	return nil
} //handler.Send()

//handleReply() handles reply messages from nats after we sent with conn.Request()
func (h *handler) handleReply(msg *nats.Msg) {
	logger.Debugf("Received reply \"%s\" on subject %s", msg.Data, msg.Subject)
	var replyChan chan *nats.Msg
	var ok bool
	key := msg.Subject
	h.replyChannelsLock.Lock()
	if replyChan, ok = h.replyChannels[key]; !ok {
		h.replyChannelsLock.Unlock()
		logger.Errorf("%+v", errors.Errorf("reply key(%s) not found, discarding \"%s\"", key, msg.Data))
		return
	}
	delete(h.replyChannels, key)
	h.replyChannelsLock.Unlock()
	replyChan <- msg
	close(replyChan)
	logger.Tracef("Replied for %s", key)
} //handler.handleReply()

// // SendReply sends a reply to the reply queue of domain and operation
// func (handler *NatsHandler) SendReply(message *Message) error {

// 	return handler.SendSubject(
// 		message.Header.ReplyAddress,
// 		message)

// } // NatsHandler.SendReply()

// // Send sends message to Nats
// func (handler *NatsHandler) Send(message *Message) error {

// 	return handler.SendSubject(
// 		"",
// 		message)

// } // NatsHandler.Send()

// // SendAndReceive sends a message on Nats, and waits for a reply
// func (handler *NatsHandler) SendAndReceive(reqMessage *Message, resMessage *Message) error {

// 	const method = "NatsHandler.SendAndReceive"

// 	if handler == nil {
// 		return errors.Errorf("invalid parameters %p.%s ()",
// 			handler,
// 			method)
// 	} // if invalid params

// 	if err := handler.SendAndReceiveSubject(
// 		"",
// 		reqMessage,
// 		resMessage); err != nil {

// 		return errors.Wrapf(err, "Error sending message")

// 	} // if failed to send

// 	return nil

// } // NatsHandler.SendAndReceive()

// // SendAndReceiveSubject sends a message on Nats on a given subject, and waits
// // for a reply
// func (handler *NatsHandler) SendAndReceiveSubject(subject string, reqMessage *Message,
// 	resMessage *Message) error {

// 	const method = "NatsHandler.SendAndReceiveSubject"

// 	if handler == nil || reqMessage == nil || resMessage == nil {
// 		return errors.Errorf("invalid parameters %p.%s ()",
// 			handler,
// 			method)
// 	} // if invalid params

// 	log := natsLogger.Named(method)
// 	defer log.Sync()

// 	if len(subject) <= 0 {
// 		subject = strings.Replace(reqMessage.Header.Provider.Name, "/", ".", -1)
// 		subject = strings.TrimSpace(strings.Replace(subject, ".", "", 1))
// 	} // if subject not supplied

// 	// Get the reply subject and set it in the header
// 	replySubject := handler.newReplySubject()
// 	reqMessage.Header.ReplyAddress = replySubject

// 	// Convert the message to JSON
// 	msgData, err := reqMessage.ToJSON()
// 	if err != nil {
// 		*resMessage = *reqMessage
// 		return errors.Wrap(err,
// 			"Failed to convert message to JSON")
// 	} // failed to get json msg

// 	log.Debugf("Sending Message \"%s\" on subject %s. "+
// 		"Expecting reply on subject %s.",
// 		msgData,
// 		subject,
// 		replySubject)

// 	// Make a buffered channel to prevent the go routine potentially
// 	// blocking when sending data into the channel. This can happen when
// 	// attempting to send after the receiving go routine has timedout.
// 	replyChan := make(chan *nats.Msg, 1)

// 	handler.replyChannelsLock.Lock()
// 	if _, ok := handler.replyChannels[replySubject]; ok {
// 		handler.replyChannelsLock.Unlock()
// 		return errors.Errorf("Reply subject %s already added",
// 			replySubject)
// 	} // if key
// 	handler.replyChannels[replySubject] = replyChan
// 	handler.replyChannelsLock.Unlock()

// 	// Defer remove the key from the map
// 	defer func() {

// 		log.Tracef("Attempting to remove reply channel for %s",
// 			replySubject)

// 		handler.replyChannelsLock.Lock()
// 		delete(handler.replyChannels, replySubject)
// 		handler.replyChannelsLock.Unlock()

// 		log.Tracef("Reply channel removed for %s",
// 			replySubject)

// 	}() // defer ()

// 	// Send the message
// 	sendMsg := nats.NewMsg(subject)
// 	sendMsg.Reply = replySubject
// 	sendMsg.Data = []byte(msgData)
// 	if handler.headersSupported {
// 		sendMsg.Header.Set(headerVServicesProvider, reqMessage.Header.Provider.Name)
// 	} // if headers

// 	if err = handler.conn.PublishMsg(
// 		sendMsg); err != nil {

// 		*resMessage = *reqMessage
// 		return errors.Wrapf(err, "Failed to publish request on subject %s",
// 			subject)

// 	} // if failed to publish

// 	// Wait for response
// 	ttl := time.Duration(reqMessage.Header.Ttl) * time.Millisecond
// 	log.Debugf("Waiting for reply on %s with TTL %s",
// 		replySubject,
// 		ttl)

// 	select {
// 	case replyMsg := <-replyChan:

// 		if len(replyMsg.Data) <= 0 {
// 			err := errors.Errorf("No responders for subject %s for message with GUID %s",
// 				subject,
// 				reqMessage.Header.IntGuid)
// 			log.Errorf("%+v", err)
// 			*resMessage = *reqMessage
// 			reqMessage.Header.Result = &Result{Code: -99, Description: "No responders", Details: err.Error()}
// 			return nil
// 		} // if no data

// 		if err := resMessage.FromJSON(
// 			string(replyMsg.Data)); err != nil {

// 			*resMessage = *reqMessage
// 			return errors.Wrapf(err,
// 				"Failed to create message from JSON [%s]",
// 				replyMsg.Data)

// 		} // if failed to create from JSON

// 		resMessage.Header.ReplyAddress = strings.TrimPrefix(
// 			resMessage.Header.ReplyAddress,
// 			"Q:")

// 		if !strings.Contains(resMessage.Header.ReplyAddress, ".") {
// 			resMessage.Header.ReplyAddress = resMessage.Header.ReplyAddress + ".reply"
// 		} // if does not include operation

// 		return nil

// 	case <-time.After(ttl):
// 		err := errors.Errorf("Timeout after %s waiting for reply on subject %s from provider %s for GUID %s",
// 			ttl,
// 			replySubject,
// 			reqMessage.Header.Provider.Name,
// 			reqMessage.Header.IntGuid)
// 		log.Errorf("%+v", err)
// 		*resMessage = *reqMessage
// 		reqMessage.Header.Result = &Result{Code: -99, Description: "Request Timed out", Details: err.Error()}
// 		return nil
// 	} // select

// } // NatsHandler.SendAndReceiveSubject()

// // SubscribeWithNoFilter ...
// func (handler *NatsHandler) SubscribeWithNoFilter(subject string,
// 	callback HandlerSubscribe) error {

// 	const method = "NatsHandler.SubscribeWithNoFilter"

// 	if handler == nil {
// 		return errors.Errorf("invalid parameters %p.%s ()",
// 			handler,
// 			method)
// 	} // if invalid params

// 	log := natsLogger.Named(method)
// 	defer log.Sync()

// 	handler.subscriptionsLock.Lock()
// 	defer handler.subscriptionsLock.Unlock()

// 	if _, ok := handler.subscriptions[subject]; ok {

// 		log.Errorf("%+v", errors.Errorf("Subject [%s] already subscribed to",
// 			subject))
// 		return nil

// 	} // if exists

// 	var subscription *nats.Subscription
// 	var err error

// 	if !handler.config.NormalSubscription {

// 		subscription, err = handler.conn.QueueSubscribe(subject, fmt.Sprintf("Q.%s", subject), func(msg *nats.Msg) {
// 			callback(msg.Data, msg.Reply)
// 		})

// 		if err != nil {
// 			log.Errorf("Queue Subscribe failed. Error: [%s]",
// 				err.Error())
// 		} // if failed

// 	} else {

// 		subscription, err = handler.conn.Subscribe(subject+".*", func(msg *nats.Msg) {
// 			callback(msg.Data, msg.Reply)
// 		})

// 		if err != nil {
// 			log.Errorf("Subscribe failed. Error: [%s]",
// 				err.Error())
// 		} // if failed

// 	}

// 	handler.subscriptions[subject] = subscription
// 	handler.defaultReplyQ = subject + ".reply"
// 	return nil

// } //NatsHandler.SubscribeWithNoFilter()

// // UnSubscribe unsubscribes from the given subject
// func (handler *NatsHandler) UnSubscribe(subject string) error {

// 	const method = "NatsHandler.UnSubscribe"

// 	if handler == nil {
// 		return errors.Errorf("invalid parameters %p.%s ()",
// 			handler,
// 			method)
// 	} // if invalid params

// 	handler.subscriptionsLock.Lock()
// 	defer handler.subscriptionsLock.Unlock()

// 	if subscription, ok := handler.subscriptions[subject]; ok {

// 		if err := subscription.Unsubscribe(); err != nil {
// 			return errors.Wrapf(err,
// 				"Failed to unsubscribe")
// 		} // if failed to un-subscribe

// 		delete(handler.subscriptions, subject)

// 	} // if exists

// 	return nil

// } // RedisHandler.UnSubscribe()

// func (handler *NatsHandler) Terminate() error {

// 	const method = "NatsHandler.Terminate"

// 	if handler == nil {
// 		return errors.Errorf("invalid parameters %p.%s ()",
// 			handler,
// 			method)
// 	} // if invalid params

// 	handler.subscriptionsLock.Lock()
// 	defer handler.subscriptionsLock.Unlock()

// 	for _, subscription := range handler.subscriptions {
// 		if subscription != nil {
// 			if err := subscription.Unsubscribe(); err != nil {
// 				return errors.Wrapf(err,
// 					"error un-subscribing")
// 			} // if failed to un sub
// 		} // if subscription
// 	} // for each subscription

// 	handler.subscriptions = make(map[string]*nats.Subscription)

// 	return nil

// } // NatsHandler.Terminate()

// // Generate a new reply subject
// func (handler *NatsHandler) newReplySubject() string {
// 	// Max length that NGF can handle is 63
// 	var sb strings.Builder
// 	n := nuid.Next()
// 	sb.Grow(len(handler.replySubjectPrefix) + len(n))
// 	sb.WriteString(handler.replySubjectPrefix)
// 	sb.WriteString(n)
// 	return sb.String()
// } // NatsHandler.newReplySubject()

// // DefaultReplyQ ...
// func (handler *NatsHandler) DefaultReplyQ() string {
// 	return handler.defaultReplyQ
// } // NatsHandler.DefaultReplyQ()
