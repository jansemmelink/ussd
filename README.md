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
