# Done #
- console is working with basic router, menu, prompt and final
- rest-sessions is working
- rest-ussd is working and using rest-sessions (no menu items except exit defined...)
- nats-ussd is working with in-memory session
- nats-ussd also uses rest-sessions and can run multiple instances and continue in another instance
- updated example/pcm 
- updated item type interfaces and updated lots of code in ussd and pcm to work like that
- PCM in ussd-nats seems to work except deliver is not implemented and ItemSvcWait not yet used.

# Next #
- do long service call with an ItemSvcWait and see if call response can be handled by other instance
        and should also be able to restart a single nats-ussd while call is being processed and have it continue.
- seems to handle user continue in any instance - need to verify again after item type changes
- create examples in code and files to show how deasy/difficult custom scripting/programming would be
- need to update console and rest version of service to demo

- test with multiple instances of rest-ussd (need to specify addr:port on command line)
- make nats USSD service
    consider existing framework or rather close go-routines while waiting and let any instance reply

    make long call with res on generic topic that all nats ussd instances listen on
    send reply to original service waiting for a reply
    test with ms-client
- make ms client for USSD to call existing ms-vservices-xxx services through NATS and test with delay

- make ext calls that takes time to complete and show that res can be handled by other instance
        rest-ussd will wait for reply (good) but ussd will not keep open
        let resp come to generic ussd res topic on NATS then any rest-ussd can process
- load items from file
- mix items from packages and files
- implement more types of items and generic statemets/switches etc.
- try simple web UI withinput form or simple react app
- implement few examples to see how possible it is
- try rest API to manage ussd definitions? i.e. run-time menu definition and editing safed to file - over multiple instances...
- consider how NGF need to change to use this
- make ussd.ReqType and ussd.ResType separate because not completely the same

# USSD Processing #
USSD request is either:

- BEGIN <ussd code> to start a new USSD session
- CONTINUE <user input> to process user input
- RELEASE to end a session

In all cases, the request is handled and a response is sent, but no resources are kept open except the session data that is stored centrally so that any instance of this service can handle the session.

The response is either:

- CONTINUE <text> to display text to user and wait for input (CONTINUE request) or abort (RELEASE request).
- RELEASE <text> to display final response that ends the session and no input allowed.
- REDIRECT <ussd code> to end the current session then HLR starts a new session similar to BEGIN. User not involved in this.

Processing of USSD request is divided into a series of the following:

- Translate (On BEGIN only, translate code to another code)
- Route (On BEGIN only, from code, select service to call)
- Menu (shows a list of items to choose from)
- Prompt (ask a question)
- Assignment (set session value using an expression)
- IF/Switch (evalualte an expression then choose next)
- Service Call (calls external micro-service)
- HTTP (call external HTTP service)
- SQL (executes SQL query on external database)
- Cache GET/SET (gets/sets cache values with expiry outside the session)
- Script (execute a script)
- Select Language (select language used for text translations)

# TODO #

- Unit Testing
- Generate Documentation + annotate
- custom audits/metrics/logs
- trace trigger on MSISDN or pattern
