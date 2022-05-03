package ussd

import "context"

func NewFinal(id string, text string) *Final {
	f := &Final{
		id:   id,
		text: text,
	}
	itemByID[id] = f
	return f
}

//Final implements ussd.Item
type Final struct {
	id   string
	text string
}

func (f Final) ID() string { return f.id }

func (f Final) Render(ctx context.Context) string {
	return f.text
}
