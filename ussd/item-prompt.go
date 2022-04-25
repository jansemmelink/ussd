package ussd

import "context"

//Prompt implements ussd.ItemWithInputHandler
type Prompt struct {
	id         string
	text       string
	name       string
	validators []InputValidator
	next       Item
}

type InputValidator interface {
	Validate(input string) error
}

func NewPrompt(id string, text string, name string, next Item) *Prompt {
	p := &Prompt{
		id:         id,
		text:       text,
		name:       name,
		validators: nil,
		next:       next,
	}
	itemByID[id] = p
	return p
}

func (p Prompt) ID() string {
	return p.id
}

func (p *Prompt) Exec(ctx context.Context) (string, Item, error) {
	return p.text, p, nil
}

func (p *Prompt) HandleInput(ctx context.Context, input string) (string, Item, error) {
	s := ctx.Value(CtxSession{}).(Session)
	for _, v := range p.validators {
		if err := v.Validate(input); err != nil {
			return err.Error(), p, nil
		}
	}
	//todo: optional validator + invalid message
	s.Set(p.name, input)
	return "", p.next, nil
}
