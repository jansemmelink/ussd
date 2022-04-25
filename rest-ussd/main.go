package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"bitbucket.org/vservices/ms-vservices-ussd/ussd"
	"bitbucket.org/vservices/utils/v4/errors"
	"bitbucket.org/vservices/utils/v4/logger"
	"github.com/gorilla/mux"
)

var log = logger.NewLogger()

func main() {
	ussd.SetSessions(httpSessions{addr: "http://localhost:8100"})

	mux := mux.NewRouter()
	mux.HandleFunc("/ussd/{msisdn}", handleUSSDBegin).Methods(http.MethodPost)
	mux.HandleFunc("/ussd/{msisdn}", handleUSSDCont).Methods(http.MethodPut)
	mux.HandleFunc("/ussd/{msisdn}", handleUSSDAbort).Methods(http.MethodPost)
	http.Handle("/", mux)
	http.ListenAndServe(":8080", nil)
}

var initItem ussd.ItemWithInputHandler

func init() {
	menu123 := ussd.NewMenu("123", "*** MAIN MENU ***").
		With("one", nil).
		With("two", nil).
		With("three", nil).
		With("four", nil).
		With("Exit", ussd.NewFinal("exit", "Goodbye."))

	initItem = ussd.NewRouter("mainRouter").
		WithCode("*123#", menu123)
}

type beginRequest struct {
	Text string `json:"text"`
}

func handleUSSDBegin(httpRes http.ResponseWriter, httpReq *http.Request) {
	msisdn := mux.Vars(httpReq)["msisdn"]
	if msisdn == "" {
		http.Error(httpRes, "missing msisdn in URL", http.StatusBadRequest)
		return
	}

	req := beginRequest{}
	if err := json.NewDecoder(httpReq.Body).Decode(&req); err != nil {
		http.Error(httpRes, err.Error(), http.StatusBadRequest)
		return
	}
	log.Debugf("req: %+v", req)

	resChan := make(chan ussdResponse)
	var res ussdResponse
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		select {
		case res = <-resChan:
		case <-time.After(15 * time.Second):
			res = ussdResponse{Type: ussd.TypeFinal, Text: "Timeout. Please try again later"}
		}
		wg.Done()
	}()

	ctx := context.Background()
	id := "http:" + msisdn
	data := map[string]interface{}{
		"msisdn": msisdn,
	}
	if err := ussd.UserStart(ctx, id, data, initItem, req.Text, responder{resChan: resChan}); err != nil {
		http.Error(httpRes, err.Error(), http.StatusInternalServerError)
		return
	}
	wg.Wait()
	httpRes.Header().Set("Content-Type", "application/json")
	json.NewEncoder(httpRes).Encode(res)
}

type contRequest struct {
	Text string `json:"text"`
}

func handleUSSDCont(httpRes http.ResponseWriter, httpReq *http.Request) {
	msisdn := mux.Vars(httpReq)["msisdn"]
	if msisdn == "" {
		http.Error(httpRes, "missing msisdn in URL", http.StatusBadRequest)
		return
	}
	req := contRequest{}
	if err := json.NewDecoder(httpReq.Body).Decode(&req); err != nil {
		http.Error(httpRes, err.Error(), http.StatusBadRequest)
		return
	}
	log.Debugf("req: %+v", req)

	resChan := make(chan ussdResponse)
	var res ussdResponse
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		select {
		case res = <-resChan:
		case <-time.After(15 * time.Second):
			res = ussdResponse{Type: ussd.TypeFinal, Text: "Timeout. Please try again later"}
		}
		wg.Done()
	}()
	id := "http:" + msisdn
	ctx := context.Background()
	if err := ussd.UserContinue(ctx, id, req.Text, responder{resChan: resChan}); err != nil {
		http.Error(httpRes, err.Error(), http.StatusInternalServerError)
		return
	}

	wg.Wait()
	httpRes.Header().Set("Content-Type", "application/json")
	json.NewEncoder(httpRes).Encode(res)
}

func handleUSSDAbort(httpRes http.ResponseWriter, httpReq *http.Request) {
	msisdn := mux.Vars(httpReq)["msisdn"]
	if msisdn == "" {
		http.Error(httpRes, "missing msisdn in URL", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	id := "http:" + msisdn
	if err := ussd.UserAbort(ctx, id); err != nil {
		http.Error(httpRes, err.Error(), http.StatusInternalServerError)
		return
	}
}

type responder struct {
	resChan chan ussdResponse
}

func (r responder) ID() string { return "<no id>" }

func (r responder) Respond(key interface{}, resType ussd.Type, resText string) error {
	r.resChan <- ussdResponse{
		Type: resType,
		Text: resText,
	}
	return nil
}

type ussdResponse struct {
	Type ussd.Type `json:"type"`
	Text string    `json:"text"`
}

//implements ussd.Sessions
type httpSessions struct {
	addr string
}

func (c httpSessions) New(id string, initData map[string]interface{}) (ussd.Session, error) {
	hs := httpSession{
		ID:   id,
		Data: initData,
	}
	buf := bytes.NewBuffer(nil)
	json.NewEncoder(buf).Encode(hs)
	httpReq, _ := http.NewRequest(
		http.MethodPost,
		c.addr+"/session/"+id,
		buf)
	httpRes, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to access HTTP session")
	}
	switch httpRes.StatusCode {
	case http.StatusOK:
		if err := json.NewDecoder(httpRes.Body).Decode(&hs); err != nil {
			return nil, errors.Wrapf(err, "failed to decode HTTP session")
		}
		if hs.StartTime == nil || hs.LastTime == nil {
			t0 := time.Now()
			hs.StartTime = &t0 //just for sanity
			hs.LastTime = &t0  //just for sanity
		}
		return ussd.NewSession(
			c,
			id,
			*hs.StartTime,
			*hs.LastTime,
			initData,
		), nil
	default:
		return nil, errors.Errorf("failed to create session: %+v", httpRes.Status)
	}
}

func (c httpSessions) Get(id string) (ussd.Session, error) {
	httpReq, _ := http.NewRequest(
		http.MethodGet,
		c.addr+"/session/"+id,
		nil)
	httpRes, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to access HTTP session")
	}
	switch httpRes.StatusCode {
	case http.StatusOK:
		var hs httpSession
		if err := json.NewDecoder(httpRes.Body).Decode(&hs); err != nil {
			return nil, errors.Wrapf(err, "failed to decode HTTP session")
		}
		if hs.StartTime == nil || hs.LastTime == nil {
			t0 := time.Now()
			hs.StartTime = &t0 //just for sanity
			hs.LastTime = &t0  //just for sanity
		}
		return ussd.NewSession(
			c,
			id,
			*hs.StartTime,
			*hs.LastTime,
			hs.Data,
		), nil
	case http.StatusNotFound:
		return nil, errors.Errorf("session not found")
	default:
		return nil, errors.Errorf("failed to get session: %+v", httpRes.Status)
	}
}

func (c httpSessions) Del(id string) error {
	httpReq, _ := http.NewRequest(
		http.MethodDelete,
		c.addr+"/session/"+id,
		nil)
	httpRes, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return errors.Wrapf(err, "failed to access HTTP session")
	}
	switch httpRes.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return errors.Errorf("failed to delete session: %+v", httpRes.Status)
	}
}

func (c httpSessions) Sync(id string, set map[string]interface{}, del map[string]bool) error {
	hs := httpSession{
		ID:   id,
		Data: set,
	}
	for n := range del {
		hs.Data[n] = nil
	}
	buf := bytes.NewBuffer(nil)
	json.NewEncoder(buf).Encode(hs)
	httpReq, _ := http.NewRequest(
		http.MethodPut,
		c.addr+"/session/"+id,
		buf)
	httpRes, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return errors.Wrapf(err, "failed to access HTTP session")
	}
	switch httpRes.StatusCode {
	case http.StatusOK:
		if err := json.NewDecoder(httpRes.Body).Decode(&hs); err != nil {
			return errors.Wrapf(err, "failed to decode HTTP session")
		}
		log.Debugf("Synced: %+v", hs)
		return nil
	default:
		return errors.Errorf("failed to sync session: %+v", httpRes.Status)
	}
}

type httpSession struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data,omitempty"`
	StartTime *time.Time             `json:"start_time,omitempty"`
	LastTime  *time.Time             `json:"last_time,omitempty"`
}
