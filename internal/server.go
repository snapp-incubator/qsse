package internal

import (
	"context"
	"encoding/json"

	"github.com/lucas-clemente/quic-go"
	"github.com/snapp-incubator/qsse/auth"
	"go.uber.org/zap"
)

// DELIMITER is the delimiter used to separate messages in streams.
const DELIMITER = '\n'

// Server is the main struct for the server.
type Server struct {
	Worker       Worker
	Listener     quic.Listener
	EventSources map[string]*EventSource
	Topics       []string
	Logger       *zap.Logger

	Authenticator auth.Authenticator
	Authorizer    auth.Authorizer
}

// DefaultAuthenticationFunc is the default authentication function. it accepts all clients.
func DefaultAuthenticationFunc(token string) bool {
	return true
}

// DefaultAuthorizationFunc is the default authorization function. it accepts all clients.
func DefaultAuthorizationFunc(token, topic string) bool {
	return true
}

// Publish publishes an event to all the subscribers of the given topic.
func (s *Server) Publish(topic string, event []byte) {
	matchedTopics := FindTopicsList(s.Topics, topic, s.Logger.Named("topic"))
	for _, matchedTopic := range matchedTopics {
		if source, ok := s.EventSources[matchedTopic]; ok {
			source.DataChannel <- event
		}
	}
}

// SetAuthenticator replaces the authentication function.
func (s *Server) SetAuthenticator(authenticator auth.Authenticator) {
	s.Authenticator = authenticator
}

// SetAuthenticatorFunc replaces the authentication function.
func (s *Server) SetAuthenticatorFunc(authenticator auth.AuthenticatorFunc) {
	s.Authenticator = authenticator
}

// SetAuthorizer replaces the authentication function.
func (s *Server) SetAuthorizer(authorizer auth.Authorizer) {
	s.Authorizer = authorizer
}

// SetAuthorizerFunc replaces the authentication function.
func (s *Server) SetAuthorizerFunc(authorizer auth.AuthorizerFunc) {
	s.Authorizer = authorizer
}

// AcceptClients accepts clients and do the following steps.
// 1. Accept a receivedStream.
// 2. Read client authentication token and topics.
// 3. Authenticate the client.
// 3.1 If the authentication is successful, opens sendStream for each topic and add them to eventSources.
// 3.2 If the authentication is not successful, closes the connection.
func (s *Server) AcceptClients() {
	for {
		background := context.Background()
		connection, err := s.Listener.Accept(background)
		checkError(err, s.Logger.Named("error"))

		s.Logger.Info("found a new client")

		client := NewSubscriber(connection, s.Logger.Named("error"))
		go s.handleClient(client)
	}
}

// handleClient authenticate client and If the authentication is successful,
// opens sendStream for each topic and add them to eventSources.
func (s *Server) handleClient(client *Subscriber) {
	isValid := s.Authenticator.Authenticate(client.Token)
	if !isValid {
		s.Logger.Warn("client is not authenticated")

		code := quic.ApplicationErrorCode(CodeNotAuthorized)
		err := client.connection.CloseWithError(code, ErrNotAuthorized.Error())
		checkError(err, s.Logger.Named("error"))

		return
	}

	s.Logger.Info("client is authenticated")

	sendStream, err := client.connection.OpenUniStream()
	checkError(err, s.Logger.Named("error"))

	s.addClientTopicsToEventSources(client, sendStream)
}

// addClientTopicsToEventSources adds the client's sendStream to the eventSources.
func (s *Server) addClientTopicsToEventSources(client *Subscriber, sendStream quic.SendStream) {
	for _, topic := range client.Topics {
		if ok := s.Authorizer.Authorize(client.Token, topic); !ok {
			s.Logger.Warn("client is not authorized for %s", zap.String("topic", topic))

			continue
		}

		if _, ok := s.EventSources[topic]; ok {
			s.EventSources[topic].Subscribers = append(s.EventSources[topic].Subscribers, sendStream)
		} else {
			e := NewErr(CodeTopicNotAvailable, map[string]any{
				"topic": topic,
			})
			errBytes, _ := json.Marshal(e) //nolint:errchkjson
			errEvent := NewEvent(ErrorTopic, errBytes)
			err := WriteData(errEvent, sendStream)
			checkError(err, s.Logger.Named("error"))
		}
	}
}

// GenerateEventSources generates eventSources for each topic.
func (s *Server) GenerateEventSources(topics []string) {
	for _, topic := range topics {
		if _, ok := s.EventSources[topic]; !ok {
			s.Logger.Info("creating new event source for topic", zap.String("topic", topic))
			s.EventSources[topic] = NewEventSource(topic, make(chan []byte), []quic.SendStream{})

			go s.EventSources[topic].TransferEvents(s.Worker)
		}
	}
}

func checkError(err error, l *zap.Logger) {
	if err != nil {
		l.Error("error occurred", zap.Error(err))
	}
}
