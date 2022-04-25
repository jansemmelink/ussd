package ussd

//Responder sends a cont/final response to the user
//different responders can be user, based on how you got the user input
type Responder interface {
	ID() string
	Respond(key interface{}, resType Type, resText string) error
}
