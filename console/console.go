package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"unicode"

	//"bitbucket.org/vservices/ussd/v3"
	"bitbucket.org/vservices/ms-vservices-ussd/ussd"
	"bitbucket.org/vservices/utils/v4/errors"
	"bitbucket.org/vservices/utils/v4/logger"
)

var log = logger.NewLogger()

const imsiPattern = `[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]`

var imsiRegex = regexp.MustCompile("^" + imsiPattern + "$")

func Run(initItem ussd.ItemWithInputHandler) {
	msisdnPtr := flag.String("msisdn", "27821234567", "MSISDN in international format (10..15 digits)")
	imsiPtr := flag.String("imsi", "", "IMSI (default: not defined)")
	maxlPtr := flag.Int("maxl", 182, "Maximum length (valid 50..500)")
	//builtInServicesPtr := flag.Bool("builtin", false, "Include default built in service for demonstration purposes")
	//initItemIdPtr := flag.String("init", "builtInRouter", "Init item id to start all services from")
	flag.Parse()

	if len(*msisdnPtr) < 10 || len(*msisdnPtr) > 15 || (*msisdnPtr)[0] == '0' {
		panic(fmt.Sprintf("--msisdn=%s must be 10..15 digits and not starting with a '0'", *msisdnPtr))
	}
	for _, c := range *msisdnPtr {
		if !unicode.IsDigit(c) {
			panic(fmt.Sprintf("--msisdn=%s must be 10..15 digits and not starting with a '0'", *msisdnPtr))
		}
	}
	if (len(*imsiPtr) != 0 && len(*imsiPtr) != 15) || (len(*imsiPtr) == 15 && !imsiRegex.MatchString(*imsiPtr)) {
		panic(fmt.Sprintf("--imsi=%s must be 15 digits or not specified", *imsiPtr))
	}
	if *maxlPtr < 50 || *maxlPtr > 500 {
		panic(fmt.Sprintf("--maxl=%d is not 50..500", *maxlPtr))
	}

	// //load default services to demonstrate USSD workings
	// if *builtInServicesPtr {
	// 	if err := builtIn(); err != nil {
	// 		panic(fmt.Sprintf("failed to create built-in services: %+v", err))
	// 	}
	// }

	// var initItem ussd.ItemWithInputHandler
	// if item, ok := ussd.ItemByID(*initItemIdPtr); !ok {
	// 	panic(fmt.Sprintf("--init=%s not found", *initItemIdPtr))
	// } else {
	// 	initItem, ok = item.(ussd.ItemWithInputHandler)
	// 	if !ok {
	// 		panic(fmt.Sprintf("--init=%s of type %T does not handle input", *initItemIdPtr, item))
	// 	}
	// }

	//load custom services from file(s)

	//select init service (typical a ussd.Router)

	//create a user input channel used for all console input
	//so we can constantly read the terminal
	userInputChan := make(chan string)
	go func(userInputChan chan string) {
		reader := bufio.NewReader(os.Stdin)
		for {
			input, err := reader.ReadString('\n')
			if err != nil {
				fmt.Fprintf(os.Stderr, "%+v", errors.Wrapf(err,
					"Error reading from stdin"))
				userInputChan <- "exit"
				return
			} // if err
			input = strings.Replace(input, "\n", "", -1)
			userInputChan <- input
		}
	}(userInputChan)

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT) //<ctrl><C>
	go func() {
		<-signalChannel
		userInputChan <- "exit"
	}()

	//main USSD session loop
	//the session ID is just for the console, not the service
	//in the service, there can only be one session per MSISDN at any point in time
	//starting a new session with an MSISDN that has an session will hi-jack that
	//session, as this is not possible in the HLR
	sessionNr := int64(0)
	for {
		//start a new session
		sessionNr++

		//prompt the user for the initial USSD string
		fmt.Fprintf(os.Stdout, "\n")
		fmt.Fprintf(os.Stdout, "\n")
		fmt.Fprintf(os.Stdout, "===== U S S D - S I M U L A T O R =====\n")
		fmt.Fprintf(os.Stdout, "    ( session: %d )    \n", sessionNr)
		fmt.Fprintf(os.Stdout, "---------------------------------------\n")

		ussdDialString := ""
		for len(ussdDialString) == 0 {
			fmt.Fprintf(os.Stdout, "USSD > ")
			ussdDialString = <-userInputChan
		}
		if ussdDialString == "exit" {
			fmt.Fprintf(os.Stdout, "Terminated.\n")
			break
		}
		if ussdDialString[0] != '*' && ussdDialString[0] != '#' {
			fmt.Fprintf(os.Stdout, "  ERROR: USSD must begin with '*' or '#'. Type exit to quit.\n")
			fmt.Fprintf(os.Stdout, "\n")
			continue
		}

		data := map[string]interface{}{
			"maxl": *maxlPtr,
		}
		if len(*imsiPtr) == 15 {
			data["imsi"] = *imsiPtr
		}

		id := "console:" + *msisdnPtr
		resChan := make(chan consoleResponse)

		//USSD started: process all USSD service responses to the user
		ended := false
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			for res := range resChan {
				log.Debugf("res: %+v", res)
				fmt.Fprintf(os.Stdout, "\n")
				fmt.Fprintf(os.Stdout, "%s\n", res.resText)
				fmt.Fprintf(os.Stdout, "-----------------------------(len:%3d)--\n", len(res.resText))

				switch res.resType {
				case ussd.TypeFinal:
					fmt.Fprintf(os.Stdout, "==========[ E N D ]====================\n")

					continue
				case ussd.TypeRedirect:
					fmt.Fprintf(os.Stdout, "==========[ R E D I R E C T ]==========\n")
					continue
				case ussd.TypePrompt:
					log.Debugf("expecting more responses...")
					continue
				default:
					fmt.Fprintf(os.Stdout, "ERROR:\n")
					fmt.Fprintf(os.Stdout, "  UNEXPECTED Type=%s!!!\n", res.resType)
					fmt.Fprintf(os.Stdout, "  Response: {type:%s, message:%s}\n", res.resType, res.resText)
					fmt.Fprintf(os.Stdout, "==========[ E R R O R ]================\n")
					close(resChan)
					continue
				}
			} //for all responses
			log.Debugf("Out of res loop")

			userInputChan <- "" //just to end that loop after we ended the res loop
			ended = true
			wg.Done()
		}()

		ctx := context.Background()
		log.Debugf("Starting USSD (id:%s)...", id)
		if err := ussd.UserStart(ctx, id, data, initItem, ussdDialString, consoleResponder{resChan: resChan}); err != nil {
			fmt.Fprintf(os.Stdout, "  ERROR: USSD failed to start: %+v", err)
			fmt.Fprintf(os.Stdout, "\n")
			continue
		}
		log.Debugf("Started USSD...")

		//input loop
		for {
			//prompting: read input
			//fmt.Fprintf(os.Stdout, "     ? ")
			input := <-userInputChan
			if len(input) == 0 {
				if !ended {
					fmt.Fprintf(os.Stdout, "*** Abort ***\n")
					ussd.UserAbort(context.Background(), id)
				}
				break //continue
			} //if aborted by just hitting <enter>

			//got user input
			if err := ussd.UserContinue(context.Background(), id, nil, input, consoleResponder{resChan: resChan}); err != nil {
				fmt.Fprintf(os.Stdout, "Continue failed: %+v\n", err)
				fmt.Fprintf(os.Stdout, "==========[ E R R O R ]================\n")
				close(resChan)
				break //continue
			}

			log.Debugf("continue success, waiting for next response")
		} //for interactions until resChan closes
		log.Debugf("Out of input loop")

		//wait for session to end
		log.Debugf("waiting for session to end")
		wg.Wait()
		log.Debugf("session ended")

	} //for each USSD session

}

type consoleResponder struct {
	resChan chan consoleResponse
}

func (cr consoleResponder) ID() string { return "console" }

func (cr consoleResponder) Respond(key interface{}, resType ussd.Type, resText string) error {
	log.Debugf("responder.Respond(%v,%s,%s)", key, resType, resText)
	cr.resChan <- consoleResponse{
		resText: resText,
		resType: resType,
	}
	log.Debugf("responder.Responded(%v,%s,%s)", key, resType, resText)
	if resType == ussd.TypeFinal || resType == ussd.TypeRedirect {
		close(cr.resChan)
		cr.resChan = nil
		log.Debugf("%s -> Closed res chan", resType)
	} else {
		log.Debugf("%s -> Not closed chan", resType)
	}
	return nil
}

type consoleResponse struct {
	resText string
	resType ussd.Type
}

// func builtIn() error {

// 	menu123_1 := ussd.NewMenu("123-1-menu", "*** SUB ***")

// 	hello := ussd.NewFinal("hello", "Hello <name> <age>")
// 	askAge := ussd.NewPrompt("ask_age", "Enter your age", "your_age", hello)
// 	askName := ussd.NewPrompt("ask_name", "Enter your name", "your_name", askAge)

// 	menu123 := ussd.NewMenu("123-main", "*** 123 ***").
// 		With("sub menu", menu123_1).
// 		With("two", askName)

// 	menu123_1 = menu123_1.With("back", menu123)

// 	ussd.NewRouter("builtInRouter").
// 		WithPrefix("*123", menu123)

// 	return nil
// }
