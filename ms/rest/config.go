package rest

import (
	"bitbucket.org/vservices/ms-vservices-ussd/ms"
	"bitbucket.org/vservices/utils/v4/errors"
)

type Config struct {
	Address string `json:"address" doc:"HTTP server address (default ':8080')"`
}

func (c *Config) Validate() error {
	if c.Address == "" {
		c.Address = ":8080"
	}
	return nil
}

func (c Config) New() (ms.Handler, error) {
	if err := c.Validate(); err != nil {
		return nil, errors.Wrapf(err, "invalid nats config")
	}
	return nil, errors.Errorf("NYI")
}

// func (r rest) Subscribe(subject string, broadcast bool, callback HandlerFunc) error {
// 	return errors.Errorf("NYI")
// }

// func (r rest) Send(header map[string]string, subject string, data []byte) error {
// 	return errors.Errorf("NYI")
// }
