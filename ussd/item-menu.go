package ussd

import (
	"context"
	"fmt"
	"strconv"
)

//Menu implements ussd.ItemWithInputHandler
type Menu struct {
	id       string
	title    string
	options  []MenuOption
	rendered bool
}

type MenuOption struct {
	caption string
	next    Item
}

func NewMenu(id string, title string) *Menu {
	m := &Menu{
		id:       id,
		title:    title,
		options:  []MenuOption{},
		rendered: false,
	}
	itemByID[id] = m
	return m
}

func (m Menu) ID() string { return m.id }

func (m *Menu) With(caption string, next Item) *Menu {
	m.options = append(m.options, MenuOption{
		caption: caption,
		next:    next,
	})
	return m
}

func (m *Menu) Exec(ctx context.Context) (string, Item, error) {
	if !m.rendered {
		//first time:
		//substitute values into text
		//todo...

		//break into pages
		//todo...

		m.rendered = true
	}

	//see which page to render
	//todo...

	//todo: set in session menu option map -> next item

	menuPage := m.title
	for n, i := range m.options {
		menuPage += fmt.Sprintf("\n%d. %s", n+1, i.caption)
	}

	//prompt user for input showing this page
	return menuPage, m, nil
}

func (m *Menu) HandleInput(ctx context.Context, input string) (string, Item, error) {
	log.Debugf("menu(%s) got input(%s) ...", m.id, input)
	if i64, err := strconv.ParseInt(input, 10, 64); err == nil && i64 >= 1 && int(i64) <= len(m.options) {
		next := m.options[i64-1].next
		if next == nil {
			return "item not yet implemented", nil, nil
		}

		log.Debugf("menu(%s) selected(%s) -> %T", m.id, input)
		return "", next, nil //selected menu item
	}
	log.Debugf("invalid menu selection - display menu again")
	return m.Exec(ctx) //invalid selection - display same menu again
}
