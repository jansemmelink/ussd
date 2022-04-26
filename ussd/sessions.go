package ussd

import (
	"sync"
	"time"
)

type Sessions interface {
	New(id string, initData map[string]interface{}) (Session, error)
	Get(id string) (Session, error)
	Del(id string) error
	Sync(id string, set map[string]interface{}, del map[string]bool) error
}

//SetSessions() changes the session manager
//it panics if sessions have been created before this is called
func SetSessions(ss Sessions) {
	if ss == nil {
		panic("SetSessions(nil)")
	}
	if sessionsStarted {
		panic("SetSessions() called after first session was used")
	}
	sessions = ss
}

var (
	//by default sessions are stored in memory
	//change to another session manager with SetSession()
	//  (before using any sessions!)
	sessionsStarted bool     = false
	sessions        Sessions = &inMemorySessions{
		sessionByID: map[string]inMemSession{},
	}
)

type inMemorySessions struct {
	sync.Mutex
	sessionByID map[string]inMemSession
}

type inMemSession struct {
	startTime time.Time
	lastTime  time.Time
	data      map[string]interface{}
}

func (ss *inMemorySessions) New(id string, initData map[string]interface{}) (Session, error) {
	sessionsStarted = true
	//create new session in memory only
	//it does not exist centrally until it is synced
	//it may even clash with another when synced
	//unless the caller first checked/deleted existing session
	//because: we do not want to delete session creation with external calls if not required
	t0 := time.Now()
	return NewSession(ss, id, t0, t0, initData), nil
}

func (ss *inMemorySessions) Get(id string) (Session, error) {
	sessionsStarted = true
	ss.Lock()
	defer ss.Unlock()
	if ims, ok := ss.sessionByID[id]; ok {
		s := NewSession(ss, id, ims.startTime, ims.lastTime, ims.data)
		log.Debugf("retrieved ims(%s): %+v", id, ims.data)
		return s, nil
	}
	return nil, nil //session not found, not an error, just nil value, err is for real errors when Get fail and session could actually exist
}

func (ss *inMemorySessions) Del(id string) error {
	sessionsStarted = true
	ss.Lock()
	defer ss.Unlock()
	delete(ss.sessionByID, id)
	return nil
}

func (ss *inMemorySessions) Sync(id string, set map[string]interface{}, del map[string]bool) error {
	sessionsStarted = true
	ss.Lock()
	defer ss.Unlock()
	t := time.Now()
	ims, ok := ss.sessionByID[id]
	if !ok {
		ims = inMemSession{
			startTime: t,
			data:      map[string]interface{}{},
		}
	}
	for name := range del {
		delete(ims.data, name)
		log.Debugf("  ims[%s] deleted", name)
	}
	for name, value := range set {
		ims.data[name] = value
		log.Debugf("  ims[%s]=%v", name, value)
	}
	ims.lastTime = t
	ss.sessionByID[id] = ims
	log.Debugf("synced ims(%s): %+v", id, ims.data)
	return nil
}
