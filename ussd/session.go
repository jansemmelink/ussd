package ussd

import (
	"fmt"
	"time"

	"bitbucket.org/vservices/utils/v4/logger"
)

var log = logger.NewLogger()

type Session interface {
	ID() string
	Get(name string) interface{}
	Set(name string, value interface{})
	Del(name string)
	StartTime() time.Time
	LastTime() time.Time
	Sync() error //apply all local updates to central storage
}

func NewSession(ss Sessions, id string, t0, t1 time.Time, data map[string]interface{}) Session {
	if ss == nil || id == "" {
		panic(fmt.Sprintf("invalid parameters for NewSession(%p,%s,%p)", ss, id, data))
	}
	s := &session{
		sessions:   ss,
		id:         id,
		startTime:  t0,
		lastTime:   t1,
		data:       data,
		namesToSet: data,
		namesToDel: map[string]bool{},
	}
	if s.data == nil {
		s.data = map[string]interface{}{}
	}
	if s.namesToSet == nil {
		s.namesToSet = map[string]interface{}{}
	}
	log.Debugf("Created Local Session(%s): %+v", s.id, s.data)
	return s
}

type session struct {
	sessions   Sessions
	id         string
	startTime  time.Time
	lastTime   time.Time
	data       map[string]interface{}
	namesToSet map[string]interface{}
	namesToDel map[string]bool
}

func (s session) ID() string {
	return s.id
}

func (s session) StartTime() time.Time { return s.startTime }

func (s session) LastTime() time.Time { return s.lastTime }

func (s session) Get(name string) interface{} {
	if v, ok := s.data[name]; ok {
		return v
	}
	return nil
}

func (s *session) Set(name string, value interface{}) {
	if value == nil {
		s.Del(name)
		return
	}
	s.data[name] = value
	s.namesToSet[name] = value
	delete(s.namesToDel, name) //make sure its not deleted anymore
	return
}

func (s *session) Del(name string) {
	delete(s.data, name)
	delete(s.namesToSet, name)
	s.namesToDel[name] = true
}

func (s *session) Sync() error {
	s.sessions.Sync(s.id, s.namesToSet, s.namesToDel)
	s.namesToSet = map[string]interface{}{}
	s.namesToDel = map[string]bool{}
	return nil
}

type CtxSession struct{}
