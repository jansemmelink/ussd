package main

import (
	"encoding/json"
	"net/http"
	"time"

	"bitbucket.org/vservices/utils/v4/logger"
	"github.com/gorilla/mux"
)

var log = logger.NewLogger()

func main() {
	mux := mux.NewRouter()
	mux.HandleFunc("/session/{id}", handleNewSession).Methods(http.MethodPost)
	mux.HandleFunc("/session/{id}", handleGetSession).Methods(http.MethodGet)
	mux.HandleFunc("/session/{id}", handleUpdSession).Methods(http.MethodPut)
	mux.HandleFunc("/session/{id}", handleDelSession).Methods(http.MethodDelete)
	http.Handle("/", mux)
	http.ListenAndServe(":8100", nil)
}

type session struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data,omitempty"`
	StartTime *time.Time             `json:"start_time,omitempty"`
	LastTime  *time.Time             `json:"last_time,omitempty"`
}

var (
	sessions = map[string]session{}
)

func handleNewSession(httpRes http.ResponseWriter, httpReq *http.Request) {
	logger.Debugf("new session")
	id := mux.Vars(httpReq)["id"]
	if id == "" {
		http.Error(httpRes, "missing id", http.StatusBadRequest)
		return
	}
	var s session
	json.NewDecoder(httpReq.Body).Decode(&s)
	if s.ID != "" && s.ID != id {
		http.Error(httpRes, "id in URL and body does not match", http.StatusBadRequest)
		return
	}
	if s.StartTime != nil || s.LastTime != nil {
		http.Error(httpRes, "start_time and last_time may not be specified for new session", http.StatusBadRequest)
		return
	}
	s.ID = id
	for n, v := range s.Data {
		if v == nil {
			delete(s.Data, n)
		}
	}
	t0 := time.Now()
	s.StartTime = &t0
	s.LastTime = &t0
	sessions[id] = s
	httpRes.Header().Set("Content-Type", "application/json")
	json.NewEncoder(httpRes).Encode(s)
}

func handleGetSession(httpRes http.ResponseWriter, httpReq *http.Request) {
	logger.Debugf("get session")
	id := mux.Vars(httpReq)["id"]
	if id == "" {
		http.Error(httpRes, "missing id", http.StatusBadRequest)
		return
	}
	if s, ok := sessions[id]; ok {
		httpRes.Header().Set("Content-Type", "application/json")
		json.NewEncoder(httpRes).Encode(s)
		return
	}
	http.Error(httpRes, "session not found", http.StatusNotFound)
	return
}

func handleUpdSession(httpRes http.ResponseWriter, httpReq *http.Request) {
	logger.Debugf("upd session")
	id := mux.Vars(httpReq)["id"]
	if id == "" {
		http.Error(httpRes, "missing id", http.StatusBadRequest)
		return
	}
	s, ok := sessions[id]
	if !ok {
		http.Error(httpRes, "session not found", http.StatusNotFound)
		return
	}

	var upd session
	json.NewDecoder(httpReq.Body).Decode(&upd)
	if upd.ID != "" && upd.ID != id {
		http.Error(httpRes, "id in URL and body does not match", http.StatusBadRequest)
		return
	}
	if upd.StartTime != nil || upd.LastTime != nil {
		http.Error(httpRes, "start_time and last_time may not be specified in request", http.StatusBadRequest)
		return
	}
	for n, v := range upd.Data {
		if v != nil {
			if s.Data == nil {
				s.Data = map[string]interface{}{n: v}
			} else {
				s.Data[n] = v
			}
		} else {
			if s.Data != nil {
				delete(s.Data, n)
			}
		}
	}
	t1 := time.Now()
	s.LastTime = &t1
	sessions[id] = s
	httpRes.Header().Set("Content-Type", "application/json")
	json.NewEncoder(httpRes).Encode(s)
	return
}

func handleDelSession(httpRes http.ResponseWriter, httpReq *http.Request) {
	logger.Debugf("delete session")
	id := mux.Vars(httpReq)["id"]
	if id == "" {
		http.Error(httpRes, "missing id", http.StatusBadRequest)
		return
	}
	delete(sessions, id)
}
