package nats

import (
	"crypto/tls"
	"net/url"

	"bitbucket.org/vservices/ms-vservices-ussd/comms"
	"bitbucket.org/vservices/utils/v4/errors"
	"bitbucket.org/vservices/utils/v4/logger"
	datatype "bitbucket.org/vservices/utils/v4/type"
	"github.com/nats-io/nats.go"
)

type Config struct {
	Name               string            `json:"name"`
	Url                string            `json:"url"`
	NormalSubscription bool              `json:"normal_subscription"`
	Timeout            datatype.Duration `json:"timeout"`
	MaxReconnects      int               `json:"max_reconnects"`
	ReconnectWait      datatype.Duration `json:"reconnect_wait"`
	ReconnectJitter    datatype.Duration `json:"reconnect_jitter"`
	ReconnectJitterTls datatype.Duration `json:"reconnect_jitter_tls"`
	DontRandomize      bool              `json:"dont_randomize"`
	Username           string            `json:"username"`
	Password           datatype.EncStr   `json:"password"`
	Token              string            `json:"token"`
	Secure             bool              `json:"secure"`
	InsecureSkipVerify bool              `json:"insecure_skip_verify"`
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.Errorf("nil.Validate()")
	}
	if len(c.Name) <= 0 {
		return errors.Errorf("missing name")
	}
	if len(c.Url) <= 0 {
		c.Url = nats.DefaultURL
	}
	if pu, err := url.ParseRequestURI(c.Url); err != nil {
		return errors.Wrapf(err, "invalid url:\"%s\"")
	} else {
		if pu.Scheme != "nats" {
			return errors.Errorf("url:\"%s\" must have scheme \"nats://...\", not \"%s://...\"", c.Url, pu.Scheme)
		}
	}
	if c.MaxReconnects == 0 {
		return errors.Errorf("invalid max_reconnects:%d", c.MaxReconnects)
	}
	if c.ReconnectWait <= 0 {
		return errors.Errorf("invalid reconnect_wait:\"%s\"", c.ReconnectWait)
	}
	return nil
} //Config.Validate()

func (c *Config) New() (comms.Handler, error) {
	if err := c.Validate(); err != nil {
		return nil, errors.Wrapf(err, "invalid nats config")
	}
	var options []nats.Option
	options = append(options, nats.Name(c.Name))
	options = append(options, nats.Timeout(c.Timeout.Duration()))
	options = append(options, nats.MaxReconnects(c.MaxReconnects))
	options = append(options, nats.ReconnectWait(c.ReconnectWait.Duration()))
	options = append(options, nats.ReconnectJitter(c.ReconnectJitter.Duration(), c.ReconnectJitterTls.Duration()))
	options = append(options, nats.ReconnectHandler(func(conn *nats.Conn) {
		logger.Errorf("Reconnecting %+v\n", conn)
	}))
	if c.DontRandomize {
		options = append(options, nats.DontRandomize())
	}
	options = append(options, nats.UserInfo(c.Username, c.Password.StringPlain()))
	options = append(options, nats.Token(c.Token))
	if c.Secure {
		options = append(options, nats.Secure(&tls.Config{InsecureSkipVerify: c.InsecureSkipVerify}))
	}
	h := &handler{
		config:        *c,
		conn:          nil,
		subscriptions: make(map[string]*nats.Subscription),
		// defaultReplyQ:      fmt.Sprintf("%s:reply", natsConfig.Name),
		replyChannels:      make(map[string]chan *nats.Msg, 100),
		replySubjectPrefix: nats.NewInbox() + ".",
	}
	var err error
	h.conn, err = nats.Connect(c.Url, options...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to NATS")
	}
	h.headersSupported = h.conn.HeadersSupported()
	h.replySubscription, err = h.conn.Subscribe(
		h.replySubjectPrefix+"*",
		h.handleReply)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to subscribe to reply subject")
	}
	return h, nil
} //Config.New()
