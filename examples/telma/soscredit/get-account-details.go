package soscredit

import (
	"context"
	"fmt"
	"time"

	"bitbucket.org/vservices/utils/v4/errors"
)

//getAccountDetails implements ussd.ItemSvcWait
type getAccountDetails struct {
	id string
}

func (gad getAccountDetails) ID() string {
	return gad.id
}

func (gad getAccountDetails) Request(ctx context.Context) (err error) {
	//get subscriber account details from ucip and store in menu
	//also determine language preference from this
	timeNow := time.Now()
	err := nats.Call(
		"ms-vservices-telma-ucip",
		"getAccountDetails",
		ucip.Request{
			OriginNodeType:      "EXT",
			OriginHostName:      "AdaptITServices",
			OriginTransactionID: fmt.Sprintf("%s%s", timeNow.UnixMilli(), s.Get("msisdnSub")),
			OriginTimeStamp:     timeNow, //"yyyyMMdd'T'HH:mm:ssZ"
			SubscriberNumber:    s.Get("msisdnSub"),
			RequestedOwner:      1,
		},
		ucip.Response{},
	)
	if err != nil {
		return errors.Wrapf(err, "failed to get account details")
	}
	return nil
}

func (gad getAccountDetails) Process(ctx context.Context, value interface{}) (err error) {
	s.Set("accountDetails", res)
	if res.LanguageIDCurrent == "1" {
		s.Set("language", "FR")
	} else {
		s.Set("language", "MG")
	}
	return nil
}
