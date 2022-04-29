package ussd

import (
	"context"

	"bitbucket.org/vservices/utils/v4/errors"
	"github.com/google/uuid"
)

//UserStart() is called when user initiates a new USSD session
//	id must be unique session id, e.g. made up of "<source>:<msisdn>" when from GSM MAP USSD,
//		which will prevent multiple sessions to exist for the same subscriber
//		it could be new uuid for each request, but then you must ensure old sessions are cleaned up
//	data is optional and added to new session
//	initItem is first item to exec and it must define next, i.e. must be ItemRoute()
//	input would be the ussd code that was dialed
//	responder is used to respond to the user
func UserStart(ctx context.Context, id string, data map[string]interface{}, initItem ItemRoute, input string, responder Responder) error {
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
	s.Set("input", input)
	ctx = context.WithValue(ctx, CtxSession{}, s)
	return userInput(ctx, s, initItem, input, responder)
}

//UserContinue() is called when user provides input after a prompt
//	id must be same as was used for UserStart()
//	data is optional and will be set in existing session
//	input is from user
//	responder is used to respond to the user
func UserContinue(ctx context.Context, id string, data map[string]interface{}, input string, responder Responder) error {
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
	_, okPrompt := currentItem.(ItemPrompt)
	_, okMenu := currentItem.(ItemMenu)
	if !okPrompt && !okMenu {
		return errors.Errorf("session(%s).currentItemID(%s) type %T does not handle input", s.ID(), currentItemID, currentItem)
	}
	return userInput(ctx, s, currentItem, input, responder)
}

func UserAbort(ctx context.Context, id string) error {
	log.Errorf("USSD Aborted by user")
	if xerr := sessions.Del(id); xerr != nil {
		log.Errorf("failed to delete session after error: %+v", xerr)
	}
	return nil
}

//userInput is called from UserStart() or UserCont() to process the user input
func userInput(ctx context.Context, s Session, fromItem Item, input string, responder Responder) (err error) {
	var currentItem Item = fromItem

	defer func() {
		if err != nil {
			log.Errorf("USSD Failed: %+v", err)
			if xerr := sessions.Del(s.ID()); xerr != nil {
				log.Errorf("failed to delete session after error: %+v", xerr)
			}
		} else if currentItem == nil {
			log.Debugf("USSD Ended")
			if xerr := sessions.Del(s.ID()); xerr != nil {
				log.Errorf("failed to delete session after ended: %+v", xerr)
			}
		} else {
			s.Set("current_item_id", currentItem.ID())
			if xerr := s.Sync(); xerr != nil {
				log.Errorf("failed to sync session data: %+v", xerr)
			}
			log.Debugf("Synced session(%s)", s.ID())
		}
	}()

	//start by processing user input then loop until need to wait or respond
	var text string
	var nextItem Item
	text, nextItem, err = fromItem.HandleInput(ctx, input)
	_nextID := "nil"
	if nextItem != nil {
		_nextID = nextItem.ID()
	}
	log.Debugf("%T(%s).HandleInput(%s) -> text=%s,next(%s)=%T,err=%+v", fromItem, fromItem.ID(), input, text, _nextID, nextItem, err)
	for {
		//Expect one of these:
		//	"",   <next>, nil   (next!=current) to jump to <next> immediately and call <next>.Exec()
		//  "",   <next>, nil   (next==current) to waiting for a reply then call <next>.HandleReply()
		//	text, <next>, nil   (next!=nil)     to wait for user input then call <next>.HandleInput() (usually next would be current item)
		//	text, nil,    nil   for final response
		//	"",   nil,    <err> to end with system error
		if err != nil {
			log.Errorf("err: %+v", err)
			return errors.Wrapf(err, "item(%s).Exec() failed", currentItem.ID())
		}
		if text == "" && nextItem == nil {
			log.Errorf("no return")
			return errors.Wrapf(err, "item(%s).Exec() returned no text/next/err", currentItem.ID())
		}

		if text != "" {
			log.Debugf("responding...")
			//need to respond to the user with final/prompt
			responderKey, _ := s.Get("responder_key").(string)
			log.Debugf("key=(%T)%+v", responderKey, responderKey)
			if nextItem == nil {
				//no next, i.e. final response to the user and end the session
				currentItem = nil
				log.Debugf("sending final response...")
				if err = responder.Respond(responderKey, TypeFinal, text); err != nil {
					log.Errorf("failed to send final response")
					return errors.Wrapf(err, "failed to send final response")
				}
				log.Debugf("sent final")
				return nil
			}

			//not final, it is a prompt of some kind (question or menu)
			//next must be same as current item and be able to deal with user input
			if nextItem != currentItem {
				return errors.Errorf("item(%s).Exec() returned prompt(%s) with next(%s) != current item", currentItem.ID(), text, nextItem.ID())
			}
			if _, ok := nextItem.(ItemWithInputHandler); !ok {
				return errors.Errorf("item(%s).Exec() returned prompt(%s) with next(%s) which does not handle input", currentItem.ID(), text, nextItem.ID())
			}

			currentItem = nextItem
			if err = responder.Respond(responderKey, TypePrompt, text); err != nil {
				currentItem = nil
				return errors.Wrapf(err, "failed to send prompt")
			}
			return nil
		} //if sending user response/prompt

		//text == "", so not yet responding to the user
		if nextItem != currentItem {
			log.Debugf("jump...")
			//immediately jump to and execute another item
			currentItem = nextItem
			text, nextItem, err = currentItem.Exec(ctx)
			continue
		}

		//nextItem == currentItem, thus
		//Exec() started some operation and need to wait for reply
		if _, ok := currentItem.(ItemWithReplyHandler); !ok {
			log.Errorf("cannot handle reply")
			return errors.Errorf("item(%s) started operation but cannot handle a reply", currentItem.ID())
		}
		//return to let session wait for reply
		//it will resume when reply is received at any instance
		log.Debugf("waiting...")
		return nil
	} //loop
} //userInput()

//menu
//prompt
//assigment
//if or switch
//service call
//HTTP
//SQL
//cache get/set
//script
//switch language
