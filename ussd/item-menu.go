package ussd

import (
	"context"
	"fmt"
	"strconv"

	"bitbucket.org/vservices/utils/v4/errors"
)

//Menu implements ussd.ItemWithInputHandler
type Menu struct {
	id       string
	title    string
	options  []MenuOption
	rendered bool
}

type MenuOption struct {
	caption   string
	nextItems []Item
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

func (m *Menu) With(caption string, nextItems ...Item) *Menu {
	if len(nextItems) > 0 { //if menu item is implemented, nextItems may not be nil
		for i := 0; i < len(nextItems); i++ {
			if nextItems[i] == nil {
				panic(fmt.Sprintf("menu(%s).With(%s).next[%d]==nil", m.id, caption, i))
			}
		}
	}
	m.options = append(m.options, MenuOption{
		caption:   caption,
		nextItems: nextItems, //will be executed in series until the last one, expecting text="" and next="" from others
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
		nextItems := m.options[i64-1].nextItems
		if len(nextItems) == 0 {
			return "menu item not yet implemented", nil, nil
		}

		//execute all leading items except the last one, expecting text="" and next=nil for all of them
		//this allows you to set variables etc before jumping into the selected next ussd item
		for i := 0; i < len(nextItems)-1; i++ {
			log.Debugf("menu(%s).item[%d].next[%d(len:%d)].id(%s).%T.Exec()...", m.id, i64, i, len(nextItems), nextItems[i].ID(), nextItems[i])
			itemText, itemNext, itemErr := nextItems[i].Exec(ctx)
			if itemErr != nil {
				return "", nil, errors.Wrapf(itemErr, "menu(%s).item[%d].next[%d].id(%s).Exec failed", m.id, i64-1, i, nextItems[i].ID())
			}
			if itemText != "" {
				return "", nil, errors.Errorf("menu(%s).item[%d].next[%d].id(%s).Exec() returned text=\"%s\"", m.id, i64-1, i, nextItems[i].ID(), itemText)
			}
			if itemNext != nil {
				return "", nil, errors.Errorf("menu(%s).item[%d].next[%d].id(%s).Exec() returned next.id(%s)=%T", m.id, i64-1, i, nextItems[i].ID(), itemNext.ID(), itemNext)
			}
		}

		//return the last nextItem
		log.Debugf("1")
		next := nextItems[len(nextItems)-1]
		log.Debugf("menu(%s) selected(%s) -> %T", m.id, input, next)
		return "", next, nil //selected menu item
	}
	log.Debugf("invalid menu selection - display menu again")
	return m.Exec(ctx) //invalid selection - display same menu again
}
