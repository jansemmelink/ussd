package ussd

import "context"

type Prompt struct {
	Text string
	Name string
	Next Item
}

func NewPrompt(text string, name string, next Item) Prompt {
	return Prompt{
		Text: text,
		Name: name,
		Next: next,
	}
}

func (p Prompt) Exec(ctx context.Context) (Item, string, error) {
	return nil, p.Text, nil
}

func (p Prompt) HandleInput(ctx context.Context, input string) (Item, string, error) {
	//todo: optional validator + invalid message
	ctx.Set(p.Name, input)
	return p.Next, "", nil
}
