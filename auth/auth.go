package auth

// Autheticator authenticate clients on connection without paying attention
// to their subscription and etc.
type Autheticator interface {
	Authenticate(token string) bool
}

type AutheticatorFunc func(token string) bool

func (a AutheticatorFunc) Authenticate(token string) bool {
	return a(token)
}

// Authorizer authorize clients when they want to subscribe on a specific topic.
type Authorizer interface {
	Authorize(token, topic string) bool
}

type AuthorizerFunc func(token, topic string) bool

func (a AuthorizerFunc) Authorize(token, topic string) bool {
	return a(token, topic)
}
