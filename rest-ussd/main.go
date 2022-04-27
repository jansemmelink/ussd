package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	httpSessionsClient "bitbucket.org/vservices/ms-vservices-ussd/rest-sessions/client"
	"bitbucket.org/vservices/ms-vservices-ussd/ussd"
	"bitbucket.org/vservices/utils/v4/logger"
	"github.com/gorilla/mux"
)

var log = logger.NewLogger()

func main() {
	ussd.SetSessions(httpSessionsClient.New("http://localhost:8100"))
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
	if err := ussd.UserContinue(ctx, id, nil, req.Text, responder{resChan: resChan}); err != nil {
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
