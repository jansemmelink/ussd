package pcm

import (
	"context"
	"fmt"

	"bitbucket.org/vservices/ms-vservices-ussd/ussd"
	"bitbucket.org/vservices/utils/v4/errors"
	"github.com/google/uuid"
)

var pcmRouter ussd.ItemSvcExec

func Item() ussd.ItemSvcExec {
	if pcmRouter != nil {
		return pcmRouter
	}

	//profile service is built-in to make it fast, to access profile as needed directly from SQL connections
	dbConfig := DatabaseConfig{
		Host:     "127.0.0.1",
		Port:     3309,
		Username: "vservices",
		Password: "vservices",
		Database: "vservices",
	}
	if err := Connect(dbConfig); err != nil {
		panic(fmt.Sprintf("failed to connect to db: %+v", err))
	}

	blockMenu := ussd.NewMenu("pcm_block_menu", "-Call Me Messages-").
		With("Unblock Call Me Messages",
			ussd.Set("pcm_blocked", false),
			profileSetItems("pcm_blocked"),
			ussd.NewFinal("pcm_unblocked", "PCM/PRM Messages unblocked."),
		).
		With("Block Call Me Messages",
			ussd.Set("pcm_blocked", true),
			profileSetItems("pcm_blocked"),
			ussd.NewFinal("pcm_unblocked", "PCM/PRM Messages blocked."),
		)

	advertsMenu := ussd.NewMenu("pcm_block_advert", "-Call Me Adverts-").
		With("Unblock Adverts",
			ussd.Set("pcm_adverts", true),
			profileSetItems("pcm_adverts"),
			ussd.NewFinal("pcm_adverts_unblocked", "PCM Adverts unblocked."),
		).
		With("Block Adverts",
			ussd.Set("pcm_adverts", false),
			profileSetItems("pcm_adverts"),
			ussd.NewFinal("pcm_adverts_blocked", "PCM Adverts blocked."),
		)

	mainMenu = ussd.NewMenu("pcm_main_menu", "-Call Me Menu-").
		With("Block/Unblock Call Me Messages", blockMenu).
		With("Send Recharge Me",
			ussd.Set("type", "PRM"),
			ussd.NewPrompt("enter_bnumber_prm", "Enter phone number", "bnumber"), //todo: allow NewXxx() without id to use uuid
			deliver{},
		).
		With("Send Call Me",
			ussd.Set("type", "PCM"),
			ussd.NewPrompt("enter_bnumber_pcm", "Enter phone number", "bnumber"),
			deliver{},
		).
		With("Change Name",
			ussd.NewPrompt("enter_name", "Enter your name:", "name"),
			profileSetItems("pcm_name"),
			ussd.NewFinal("pcm_name_changed", "Your name was changed to <pcm_name>. You may change it again in 1 day."),
		).
		With("Display Name",
			profileGetItems("pcm_name"),
			ussd.NewFinal("display_name", "Your name is <pcm_name>"), //todo: substitute
		).
		With("PCM/PRM Balance",
			profileGetItems("pcm_balance", "prm_balance"),
			ussd.NewFinal("pcm_balances", "You Call Me balance: <bal>\nYour Recharge Me balance: <bal>"), //todo substitute
		).
		With("Disable/Enable Adverts", advertsMenu)

	//todo: can we load profile before routing? Not really required because we can deal with it on demand,
	//but some services may need a pre-action on all routes

	//router is the init item for all pcm ussd requests:
	pcmRouter = ussd.NewRouter("pcm").
		WithCode("*140#", mainMenu).
		WithRegex(`\*140\*([0-9]{10,15})#`, []string{"bnumber"}, deliver{})
	return pcmRouter
} //Item()

var (
	mainMenu ussd.Item
)

type deliver struct{}

func (deliver) ID() string { return "pcm_deliver" }

func (deliver deliver) Exec(ctx context.Context) ([]ussd.Item, error) {
	// bnumber, _ := s.Get("bnumber")
	// if err := SendSMS(bnumber, "Please Call "+s.Msisdn+" - "+"<advert>"); err != nil {
	// 	return s, errors.Errorf("failed to send")
	// }
	// return NewFinal("CallMe Delivered to " + bnumber + "-" + "<advert>").Render(s)
	return nil, errors.Errorf("NYI")
}

func profileGetItems(names ...string) ussd.Item {
	//todo: make sure names are snake_case
	return profileGet{
		id:    uuid.New().String(),
		names: names,
	}
}

type profileGet struct {
	id    string
	names []string //without: load whole profile
}

func (pg profileGet) ID() string { return pg.id }

func (pg profileGet) Exec(ctx context.Context) ([]ussd.Item, error) {
	s := ctx.Value(ussd.CtxSession{}).(ussd.Session)
	msisdn := s.Get("msisdn")
	query := "SELECT name,value FROM subscriber WHERE msisdn=?"
	args := []interface{}{msisdn}
	if len(pg.names) > 0 {
		query += " AND ("
		for i, name := range pg.names {
			if i == 0 {
				query += "name=?"
			} else {
				query += " OR name=?"
			}
			args = append(args, name)
		}
		query += ")"
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query profile(names:%v)", pg.names)
	}
	for rows.Next() {
		var name string
		var value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, errors.Wrapf(err, "failed to parse DB row")
		}
		s.Set(name, value)
		log.Debugf("Profile got msisdn(%s).(%s=\"%s\")", msisdn, name, value)
	}
	return nil, nil
} //profileGet.Exec()

func profileSetItems(names ...string) ussd.Item {
	if len(names) == 0 {
		panic("missing names")
	}
	//todo: make sure names are snake_case
	return profileSet{
		id:    uuid.New().String(),
		names: names,
	}
}

type profileSet struct {
	id    string
	names []string //required
}

func (ps profileSet) ID() string { return ps.id }

func (ps profileSet) Exec(ctx context.Context) (string, ussd.Item, error) {
	s := ctx.Value(ussd.CtxSession{}).(ussd.Session)
	msisdn := s.Get("msisdn")
	for _, name := range ps.names {
		value := fmt.Sprintf("%v", s.Get(name))
		var query string
		var args []interface{}
		if value == "nil" || value == "" {
			query = "DELETE FROM subscriber WHERE msisdn=? AND name=?"
			args = []interface{}{msisdn, name}
		} else {
			query = "INSERT INTO subscriber SET msisdn=?,name=?,value=? ON DUPLICATE KEY value=?"
			args = []interface{}{msisdn, name, value, value}
		}
		if _, err := db.Exec(query, args...); err != nil {
			return "", nil, errors.Wrapf(err, "failed to set msisdn(%s).%s=%s", msisdn, name, value)
		}
		log.Debugf("Profile set msisdn(%s).(%s=\"%s\")", msisdn, name, value)
	}
	return "", nil, nil
} //profileSet.Exec()
