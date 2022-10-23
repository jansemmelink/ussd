package ms

const TimestampFormat = "2006-01-02 15:04:05.000"

type Message struct {
	Header   MessageHeader `json:"header"`
	Request  interface{}   `json:"request,omitempty"`
	Response interface{}   `json:"response,omitempty"`
}

type MessageHeader struct {
	Timestamp    string               `json:"timestamp"`
	TTL          int                  `json:"ttl,omitempty"`
	ReplyAddress string               `json:"reply_address,omitempty"`
	EchoRequest  bool                 `json:"echo_request"`
	Result       *MessageHeaderResult `json:"result,omitempty"`
	Provider     *ServiceAddress      `json:"provider,omitempty"`
	Consumer     *ServiceAddress      `json:"consumer,omitempty"`
}

type ServiceAddress struct {
	Name string `json:"name,omitempty"`
	Tid  string `json:"tid,omitempty"`
	Sid  string `json:"sid,omitempty"`
}

type MessageHeaderResult struct {
	Code        int    `json:"code"`
	Description string `json:"description,omitempty"`
	Details     string `json:"details,omitempty"`
}
