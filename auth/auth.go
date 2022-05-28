package auth

// Authenticator authenticate clients on connection without paying attention
// to their subscription etc.
type Authenticator interface {
	Authenticate(token string) bool
}

type AuthenticatorFunc func(token string) bool

func (a AuthenticatorFunc) Authenticate(token string) bool {
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
