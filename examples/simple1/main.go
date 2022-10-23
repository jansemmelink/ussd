package main

import (
	"fmt"
	"time"

	"bitbucket.org/vservices/ms-vservices-ussd/ms"
	"bitbucket.org/vservices/ms-vservices-ussd/ms/nats"
	datatype "bitbucket.org/vservices/utils/v4/type"
)

func main() {
	nc := nats.Config{
		Domain:             "ussd",
		Url:                "nats://localhost:4222",
		Secure:             false,
		InsecureSkipVerify: true,
		Username:           "",
		//Password: "",
		MaxReconnects: 10,
		ReconnectWait: datatype.Duration(time.Second * 5),
	}
	commsHandler, err := nc.New()
	if err != nil {
		panic(fmt.Sprintf("cannot create comms handler: %+v", err))
	}

	s := ms.NewService(commsHandler)
	// 	With("a", a).
	// 	With("b", b)

	if err := commsHandler.Run(); err != nil {
		panic(err)
	}
}
