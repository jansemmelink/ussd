package ussd

import (
	"context"
	"fmt"
)

//Menu implements ussd.Item
type Menu struct {
	Title    string
	Options  []MenuOption
	Rendered bool
}

type MenuOption struct {
	Caption string
	Next    Item
}

func NewMenu(title string) Menu {
	return Menu{
		Title:    title,
		Options:  []MenuOption{},
		Rendered: false,
	}
}

func (m Menu) With(caption string, next Item) Menu {
	m.Options = append(m.Options, MenuOption{
		Caption: caption,
		Next:    next,
	})
	return m
}

func (m Menu) Exec(ctx context.Context) (Item, string, error) {
	if !m.Rendered {
		//first time:
		//substitute values into text
		//todo...

		//break into pages
		//todo...

	}

	//see which page to render
	//todo...

	menuPage := m.Title
	for n, i := range m.Options {
		menuPage += fmt.Sprintf("\n%d. %s", n+1, i.Caption)
	}

	//prompt user for input showing this page
	return nil, menuPage, nil
}
