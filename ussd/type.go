package ussd

import "fmt"

type Type int

const (
	TypeRedirect Type = iota
	TypePrompt
	TypeFinal
)

func (t Type) String() string {
	switch t {
	case TypeRedirect:
		return "redirect"
	case TypePrompt:
		return "prompt"
	case TypeFinal:
		return "final"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}
