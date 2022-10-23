package soscredit

import (
	"context"

	"bitbucket.org/vservices/ms-vservices-ussd/ussd"
	"bitbucket.org/vservices/utils/v4/errors"
)

type getUserDetails struct {
	id        string
	operation string
}

func (gud getUserDetails) ID() string { return gud.id }

func (gud getUserDetails) Request(ctx context.Context) error {
	s := ctx.Value(ussd.CtxSession{}).(ussd.Session)
	err := nats.Publish(ctx,
		"ms-vservices-soap",
		"tsSCTGetUserDetails",
		map[string]interface{}{
			"msisdn":           s.Get("msisdnInt"),
			"langauge":         s.GetOrDefault("language", "1"),
			"operation_Source": operation,
			"soapEndpoint":     configs["endpoints.properties"].getProperty("service.tsSCTService", "http://tahaq1:8040/services/tsSCTService"),
		},
		// timeout,
		// "GetUserDetails",
	)
	if err != nil {
		return errors.Wrapf(err, "failed to get user details")
	}
	return nil
}

func (gud getUserDetails) Process(ctx context.Context, value interface{}) error {
	s := ctx.Value(ussd.CtxSession{}).(ussd.Session)
	s.Set("userDetails("+gud.operation+")", res)
	return nil
}
