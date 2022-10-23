package rest

import (
	"net/http"

	"bitbucket.org/vservices/ms-vservices-ussd/ms"
	"bitbucket.org/vservices/utils/v4/errors"
	"bitbucket.org/vservices/utils/v4/logger"
)

var log = logger.NewLogger()

type handler struct {
	config Config
}

func (h handler) Run(s ms.Service) error {
	if err := http.ListenAndServe(h.config.Address, h); err != nil {
		return errors.Wrapf(err, "failed to serve on %s", h.config.Address)
	}
	return nil
}

func (h handler) ServeHTTP(httpRes http.ResponseWriter, httpReq *http.Request) {
	log.Debugf("HTTP %s %s", httpReq.Method, httpReq.URL.Path)
}
