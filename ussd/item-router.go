package ussd

import (
	"context"
	"fmt"
	"regexp"

	"bitbucket.org/vservices/utils/v4/errors"
)

func NewRouter(id string) *Router {
	r := &Router{
		id:       id,
		byCode:   map[string][]Item{},
		byPrefix: map[string][]Item{},
		byRegex:  []regexRoute{},
	}
	itemByID[id] = r
	log.Debugf("defined item: %T(%s)", r, id)
	return r
}

//Router implements ItemWithInputHandler
type Router struct {
	id       string
	byCode   map[string][]Item
	byPrefix map[string][]Item //todo: check longest first - not reliable as it is here!
	byRegex  []regexRoute
}

func (r *Router) WithCode(code string, nextItems ...Item) *Router {
	r.byCode[code] = nextItems
	return r
}

func (r *Router) WithPrefix(prefix string, nextItem ...Item) *Router {
	r.byPrefix[prefix] = nextItem
	return r
}

func (r *Router) WithRegex(pattern string, regexNames []string, item Item) *Router {
	regex, err := regexp.Compile("^" + pattern + "$")
	if err != nil {
		panic(fmt.Sprintf("invalid regex pattern: %s: %+v", pattern, err))
	}
	if regex.NumSubexp() != len(regexNames) {
		panic(fmt.Sprintf("regex(%s) has %d subexpressions but you specified %d names(%v)", pattern, regex.NumSubexp(), len(regexNames), regexNames))
	}
	r.byRegex = append(r.byRegex, regexRoute{regex: regex, names: regexNames, item: item})
	return r
}

type regexRoute struct {
	regex *regexp.Regexp
	names []string
	item  Item
}

func (r Router) ID() string {
	return r.id
} //Router.ID()

func (r Router) Exec(ctx context.Context) ([]Item, error) {
	s := ctx.Value(CtxSession{}).(Session)
	input := s.Get("init_request").(string)

	//routing: select a service based on the USSD code
	//start by looking up the exact code match, which uses a map hash
	//and will be the quickest match
	if items, ok := r.byCode[input]; ok {
		return items, nil //found exact match
	} else {
		//run through prefix matches, e.g. *123* and *123# both go to xyz
		for prefix, items := range r.byPrefix {
			if len(input) >= len(prefix) && input[0:len(prefix)] == prefix {
				return items, nil //match prefix
			}
		}
	}
	for _, route := range r.byRegex {
		if route.regex.MatchString(input) {
			if len(route.names) > 0 {
				subMatches := route.regex.FindStringSubmatchIndex(input)
				log.Debugf("matched(%s) -> %+v", input, subMatches)
			}
			return []Item{route.item}, nil
		}
	}
	return nil, errors.Errorf("unknown USSD code")
} //Router.Exec()
