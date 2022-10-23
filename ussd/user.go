package ussd

import (
	"context"
	"fmt"

	"bitbucket.org/vservices/utils/v4/errors"
	"github.com/google/uuid"
)

//Start() is called when user initiates a new session
//	id must be unique session id, e.g. made up of "<source>:<msisdn>" when from GSM MAP USSD,
//		which will prevent multiple sessions to exist for the same subscriber
//		it could be new uuid for each request, but then you must ensure old sessions are cleaned up
//	data is optional and added to new session
//	initItem is first item to exec and it must define next, i.e. must be ItemSvcExec()
//	initRequest would be the ussd code that was dialed and will be stored in session.Set("init_request",initRequest)
//	responder is used to respond to the user once (redefined each time user provides input)
func Start(ctx context.Context, id string, data map[string]interface{}, initItem ItemSvcExec, initRequest string, responder Responder, responderKey string) error {
	if id == "" {
		id = uuid.New().String()
	}
	if initItem == nil {
		return errors.Errorf("cannot start with init==nil")
	}
	if responder == nil {
		return errors.Errorf("cannot start with responder==nil")
	}
	s, err := sessions.New(id, data)
	if err != nil {
		return errors.Wrapf(err, "failed to create session(%s)", id)
	}
	s.Set("init_request", initRequest)
	ctx = context.WithValue(ctx, CtxSession{}, s)

	nextItems, err := initItem.Exec(ctx)
	if err != nil {
		return errors.Wrapf(err, "init item(%s).Exec() failed", initItem.ID())
	}
	s.Set("responder_id", responder.ID())
	s.Set("responder_key", responderKey)
	return proceed(ctx, s, nextItems)
}

//UserInput() continues a waiting session with input entered by the user
//	id must be same as was used for UserStart()
//	data is optional and will be set in existing session
//	input is from user
//	responder is used to respond to the user (it could be different from previous responder)
func UserInput(ctx context.Context, id string, data map[string]interface{}, input string, responder Responder, responderKey string) error {
	if responder == nil {
		return errors.Errorf("cannot continue with responder==nil")
	}
	s, err := sessions.Get(id)
	if err != nil {
		return errors.Wrapf(err, "failed to get session(%s)", id)
	}
	if s == nil {
		return errors.Errorf("session(%s) does not exist", id)
	}
	for n, v := range data {
		s.Set(n, v)
	}
	ctx = context.WithValue(ctx, CtxSession{}, s)
	currentItemID, _ := s.Get("current_item_id").(string)
	currentItem, ok := itemByID[currentItemID]
	if !ok {
		return errors.Errorf("session(%s).currentItemID(%s) not defined", s.ID(), currentItemID)
	}
	itemUsrPrompt, ok := currentItem.(ItemUsrPrompt)
	if !ok {
		return errors.Errorf("session(%s).currentItemID(%s) type %T does not handle user input", s.ID(), currentItemID, currentItem)
	}
	nextItems, err := itemUsrPrompt.Process(ctx, input)
	if err != nil {
		//display error to user and repeat the prompt
		text := err.Error()
		if text != "" {
			text += "\n"
		}
		text += itemUsrPrompt.Render(ctx)
		responder.Respond(ctx, responderKey, Response{Type: ResponseTypeResponse, Message: text})
		return nil
	}

	//input accepted, proceed and respond later
	s.Set("responder_id", responder.ID())
	s.Set("responder_key", responderKey)
	return proceed(ctx, s, nextItems)
}

//process() is called from Start() or Continue() to process the user input or service response
func proceed(ctx context.Context, s Session, moreNextItems []Item) (err error) {
	var currentItem Item
	defer func() {
		if err != nil {
			//end the session on error
			log.Errorf("USSD Failed: %+v", err)
			if xerr := sessions.Del(s.ID()); xerr != nil {
				log.Errorf("failed to delete session after error: %+v", xerr)
			}
		} else if currentItem == nil {
			//end the session normally
			log.Debugf("USSD Ended")
			if xerr := sessions.Del(s.ID()); xerr != nil {
				log.Errorf("failed to delete session after ended: %+v", xerr)
			}
		} else {
			//responded to user or requested something
			//now wait for continuation
			s.Set("current_item_id", currentItem.ID())
			if xerr := s.Sync(); xerr != nil {
				log.Errorf("failed to sync session data: %+v", xerr)
			}
			log.Debugf("Synced session(%s)", s.ID())
		}
	}()

	//load next items already queued for this session
	nextItems, err := loadNextItems(s)
	if err != nil {
		return errors.Wrapf(err, "failed to load queued next items")
	}
	if len(moreNextItems) > 0 {
		nextItems = append(moreNextItems, nextItems...)
	}

	for len(nextItems) > 0 {
		currentItem = nextItems[0]
		nextItems = nextItems[1:]
		{
			ids := []string{}
			for _, i := range nextItems {
				ids = append(ids, fmt.Sprintf("%T(%s)", i, i.ID()))
			}
			log.Debugf("Current item(%s), %d next: %+v", currentItem.ID(), len(nextItems), ids)
		}

		if itemUsr, ok := currentItem.(ItemUsr); ok {
			log.Debugf("item(%s)=%T is ItemUser", currentItem.ID(), currentItem)
			//user item: needs responder
			responderID := s.Get("responder_id").(string)
			responderKey := s.Get("responder_key").(string)
			if responderID == "" {
				return errors.Errorf("responder_id not defined")
			}
			responder := responderByID[responderID]
			if responder == nil {
				return errors.Errorf("responder[%s] not found", responderID)
			}
			res := Response{
				Message: itemUsr.Render(ctx),
			}
			if _, ok := itemUsr.(ItemUsrPrompt); !ok {
				currentItem = nil //final response
				res.Type = ResponseTypeRelease
			} else {
				res.Type = ResponseTypeResponse
			}
			return responder.Respond(ctx, responderKey, res)
		} //if user interaction

		//server side item (not rendering to user)
		if svcExec, ok := currentItem.(ItemSvcExec); ok {
			log.Debugf("item(%s)=%T is ItemSvcExec", currentItem.ID(), currentItem)
			moreNextItems, err := svcExec.Exec(ctx)
			if err != nil {
				return errors.Wrapf(err, "item(%s).Exec() failed", currentItem.ID())
			}
			if len(moreNextItems) > 0 {
				nextItems = append(moreNextItems, nextItems...)
			}
			continue
		}

		if svcWait, ok := currentItem.(ItemSvcWait); ok {
			log.Debugf("item(%s)=%T is ItemSvcWait", currentItem.ID(), currentItem)
			if err := svcWait.Request(ctx); err != nil {
				return errors.Wrapf(err, "item(%s) failed to request", currentItem.ID())
			}
			return nil //wait for response
		}
		log.Errorf("item(%s)=%T is ???", currentItem.ID(), currentItem)
		return errors.Errorf("not expected to get here: item(%s)=%T", currentItem.ID(), currentItem)
	} //loop
	return errors.Errorf("not expected to get here - should have ended with final response!")
} //proceed()

func UserAbort(ctx context.Context, id string) error {
	log.Errorf("USSD Aborted by user")
	if xerr := sessions.Del(id); xerr != nil {
		log.Errorf("failed to delete session after error: %+v", xerr)
	}
	return nil
}

func loadNextItems(s Session) ([]Item, error) {
	nextItemIDs, _ := s.Get("next_item_ids").([]string)
	nextItems := []Item{}
	for i, itemID := range nextItemIDs {
		if item, ok := ItemByID(itemID); !ok {
			return nil, errors.Errorf("unknown item(%s) in next_item_ids[%d]=%v", itemID, i, nextItemIDs)
		} else {
			nextItems = append(nextItems, item)
		}
	}
	return nextItems, nil
} //loadNextItems()
