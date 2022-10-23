package ms

type Handler interface {
	Run(s Service) error
	//	Subscribe(subject string, broadcast bool, callback HandlerFunc) error
	//	Send(header map[string]string, subject string, data []byte) error
}

//HandlerFunc is function prototype for queue subscription handler
type HandlerFunc func(data []byte, replyAddress string)
