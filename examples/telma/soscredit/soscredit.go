package soscredit

import (
	"context"
	"regexp"
	"strings"

	"bitbucket.org/vservices/ms-vservices-ussd/ussd"
	"bitbucket.org/vservices/utils/v4/errors"
)

var router ussd.ItemSvcExec

func Item() ussd.ItemSvcExec {
	if router != nil {
		return router
	}

	forAFriend := ussd.NewMenu("for_a_friend", "")
	fromTelma := ussd.NewMenu("from_telma", "")
	offerFromTelma := ussd.NewMenu("offer_from_telma", "")
	reimburse := ussd.NewMenu("reimburse", "")
	help := ussd.NewMenu("help", "")

	mainMenu := ussd.NewMenu("main_menu", "SOS credit").
		With("SOS_credit_for_a_friend", forAFriend).
		With("SOS_credit_from_TELMA", fromTelma).
		With("SOS_credit_offer_from_TELMA", offerFromTelma).
		With("SOS_credit_reimburse", reimburse).
		With("SOS_credit_help", help)

	router = ussd.NewRouter("soscredit").
		WithCode("*130*107#", ussd.NewFunc("init", ussdInit), getAccountDetails{}, mainMenu)
	return router
}

const msisdnPattern = `[0-9]{9,12}`

var msisdnRegex = regexp.MustCompile("^" + msisdnPattern + "$")

func cleanMsisdnTelma(s string) (string, error) {
	if strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	l := len(s)
	if l < 9 || l > 12 {
		return "", errors.Errorf("not_9_to_12_digits")
	}
	if !msisdnRegex.MatchString(s) {
		return "", errors.Errorf("not_9_to_12_digits")
	}
	if (l == 12 && strings.HasPrefix(s, "261")) ||
		(l == 10 && strings.HasPrefix(s, "0")) ||
		(l == 9) {
		return s[l-9:], nil
	}
	return "", errors.Errorf("not_a_Telma_number")
}

//called before displaying the main menu for sosCredit
//todo: make common for all Telma
func ussdInit(ctx context.Context) error {
	s := ctx.Value(ussd.CtxSession{}).(ussd.Session)

	//get subscriber MSISDN, which will always be international format
	//then create different formats of it in session data for use elsewhere in the service
	msisdnInt := s.Get("msisdn").(string)
	msisdnSub, _ := cleanMsisdnTelma(msisdnInt) //return e.g. "341234567" (always 9 digits), from USSD, so not expected to fail
	s.Set("msisdnSub", msisdnSub)               //e.g. "341234567"
	s.Set("msisdnNat", "0"+msisdnSub)           //e.g. "0341234567"
	s.Set("msisdnInt", msisdnInt)
	return nil
}
