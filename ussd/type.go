package ussd

import "fmt"

type ResponseType int

const (
	ResponseTypeRedirect ResponseType = iota
	ResponseTypePrompt
	ResponseTypeFinal
)

func (t ResponseType) String() string {
	switch t {
	case ResponseTypeRedirect:
		return "redirect"
	case ResponseTypePrompt:
		return "prompt"
	case ResponseTypeFinal:
		return "final"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

type Response struct {
	Type ResponseType `json:"type"`
	Text string       `json:"text"`
}
