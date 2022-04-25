package pcm

import (
	"context"
	"database/sql"
	"strings"

	"bitbucket.org/vservices/ms-vservices-ussd/ussd"
	"bitbucket.org/vservices/utils/v4/errors"
)

func New() ussd.Service {
	pcm := pcm{
		profileDb: nil, //todo - define connection to db or ext service
	}
	enterBNumber := ussd.NewPrompt("Enter phone number", "bnumber", nil)
	enterName := ussd.NewPrompt("Enter your name", "name", nil)
	mainMenu = ussd.NewMenu("*** CallMe ***").
		With("Send CallMe", ussd.Set("type", "PCM"), enterBNumber, deliver{}).
		With("Send RechargeMe", ussd.Set("type", "PRM"), enterBNumber, deliver{}).
		With("Block", ussd.Visible("!blocked"), setBlock{value: true}).
		With("Unblock", ussd.Visible("blocked"), setBlock{value: false}).
		With("Set Name", ussd.Visible("name_days<10"), enterName, setName{})
	return pcm
}

var (
	mainMenu ussd.Item
)

//pcm implements a ussd.Service
//it is triggered for:
//	code:	*140#
//	regex:	\*140\*[0-9]*#
type pcm struct {
	profileDb *sql.DB
}

func (pcm pcm) Exec(ctx context.Context) (ussd.Session, error) {
	msisdn := ctx.Value(ussd.Msisdn{}).(ussd.Msisdn)
	return ussd.Call(
		ctx,
		pcm.profileDb.Query("SELECT name,value FROM subscriber WHERE msisdn=?", msisdn),
		handleProfile{pcm: pcm})
}

type handleProfile struct {
	pcm pcm
}

func (handleProfile) Exec(ctx context.Context) (next ussd.Item, prompt string, err error) {
	res := ctx.Value(sql.Result{}).(sql.Result)
	switch res.Code {
	case SQLResponse:
		s.Set("a", res.A)
		s.Set("b", res.B)
	case SQLNotFound:
	default:
		//timeout, error in query, db not accessible, unauthorized, ...
		return s, errors.Errorf("failed to read profile")
	}

	if strings.HasPrefix(code, "*140*") && strings.HasSuffix(code, "#") {
		bnumber := code[5 : len(code)-6]
		if len(bnumber) > 0 && bnumber[0] == '0' {
			bnumber = "27" + bnumber[1:]
		}
		if len(bnumber) != 11 { //e.g. 27821234567
			return s, errors.Errorf("invalid request, expecting *140*OtherPhoneNumber# e.g. *140*0821234567#")
		}
		if bnumber == s.Msisdn {
			return s, errors.Errorf("cannot send CallMe to yourself")
		}
		s.Set("bnumber", bnumber)
		return pcm.DeliverCallMe(s)
	}

	return pcm.MainMenu(ctx)
}

func (pcm) MainMenu(ctx context.Context) (ussd.Session, error) {
	return ussd.Menu("*** CallMe ***").
		With("Send CallMe", AskPCMBNumber).
		With("Send RechargeMe", AskPRMBNumber).
		With("Block CallMe", BlockCallMe).
		Prompt(ctx)
}

func (pcm) AskPCMBNumber(ctx context.Context) {
	return Question("Enter Phone Number").
		Prompt(ctx, StorePCMBNumber)
}

func (pcm) AskPRMBNumber(ctx context.Context) {
	return Question("Enter Phone Number").
		Prompt(ctx, StorePRMBNumber)
}

func (pcm) StorePCMBNumber(ctx context.Context) {
	s.Set("bnumber", req.Input)
	pcm.DeliverCallMe(ctx)
}

func (pcm) StorePRMBNumber(ctx context.Context) {
	s.Set("bnumber", req.Input)
	pcm.DeliverCallMe(ctx)
}

func (pcm) Deliver(ctx context.Context) error {
	bnumber, _ := s.Get("bnumber")
	if err := SendSMS(bnumber, "Please Call "+s.Msisdn+" - "+"<advert>"); err != nil {
		return s, errors.Errorf("failed to send")
	}
	return NewFinal("CallMe Delivered to " + bnumber + "-" + "<advert>").Render(s)
}
