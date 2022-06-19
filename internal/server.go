package internal

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/lucas-clemente/quic-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	Metrics       Metrics
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
	s.Metrics.IncPublishEvent()

	matchedTopics := FindTopicsList(s.Topics, topic, s.Logger.Named("topic"))
	for _, matchedTopic := range matchedTopics {
		if source, ok := s.EventSources[matchedTopic]; ok && len(source.Subscribers) > 0 {
			s.Metrics.IncEvent(matchedTopic)
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
		if err != nil {
			s.Logger.Info("failed to accept new client", zap.Error(err))

			continue
		}

		s.Logger.Info("found a new client")

		var client *Subscriber

		client, err = NewSubscriber(connection)
		if err != nil {
			s.Logger.Error("failed to handle new subscriber", zap.Error(err))

			continue
		}

		go s.handleClient(client)
	}
}

// handleClient authenticate client and If the authentication is successful,
// opens sendStream for each topic and add them to eventSources.
func (s *Server) handleClient(client *Subscriber) {
	isValid := s.Authenticator.Authenticate(client.Token)
	if !isValid {
		s.Logger.Warn("client is not valid")

		err := CloseClientConnection(client.connection, CodeNotAuthorized, ErrNotAuthorized)
		if err != nil {
			s.Logger.Error("failed to close connection with client", zap.Error(err))
		}

		return
	}

	s.Logger.Info("client is authenticated")

	sendStream, err := client.connection.OpenUniStream()
	if err != nil {
		s.Logger.Error("failed to open send stream to client", zap.Error(err))

		er := CloseClientConnection(client.connection, CodeUnknown, err)
		if er != nil {
			s.Logger.Error("failed to close connection with client", zap.Error(err))
		}

		return
	}

	s.addClientTopicsToEventSources(client, sendStream)
}

// addClientTopicsToEventSources adds the client's sendStream to the eventSources.
func (s *Server) addClientTopicsToEventSources(client *Subscriber, sendStream quic.SendStream) {
	for _, topic := range client.Topics {
		valid, err := s.isTopicValid(client, sendStream, topic)
		if err != nil {
			s.Logger.Error("failed to send error to client", zap.Error(err))

			break
		}

		if valid {
			s.EventSources[topic].Subscribers = append(s.EventSources[topic].Subscribers, sendStream)
			s.Metrics.IncSubscriber(topic)
		}
	}
}

// isTopicValid check whether topic exists and client is authorized on it or not.
func (s *Server) isTopicValid(client *Subscriber, sendStream quic.SendStream, topic string) (bool, error) {
	if _, ok := s.EventSources[topic]; !ok {
		s.Logger.Warn("topic doesn't exists", zap.String("topic", topic))

		err := SendError(sendStream, NewErr(CodeTopicNotAvailable, map[string]any{"topic": topic}))

		return false, err
	}

	if !s.Authorizer.Authorize(client.Token, topic) {
		s.Logger.Warn("client is not authorized for topic", zap.String("topic", topic))

		err := SendError(sendStream, NewErr(CodeNotAuthorized, map[string]any{"topic": topic}))

		return false, err
	}

	return true, nil
}

// GenerateEventSources generates eventSources for each topic.
func (s *Server) GenerateEventSources(topics []string) {
	for _, topic := range topics {
		if _, ok := s.EventSources[topic]; !ok {
			s.Logger.Info("creating new event source for topic", zap.String("topic", topic))
			s.EventSources[topic] = NewEventSource(topic, make(chan []byte), []quic.SendStream{}, s.Metrics)

			go s.EventSources[topic].TransferEvents(s.Worker)
		}
	}
}

// SendError send input error to client.
func SendError(sendStream quic.SendStream, e *Error) error {
	errBytes, _ := json.Marshal(e) //nolint:errchkjson
	errEvent := NewEvent(ErrorTopic, errBytes)

	return WriteData(errEvent, sendStream)
}

func CloseClientConnection(connection quic.Connection, code int, err error) error {
	appCode := quic.ApplicationErrorCode(code)

	if err = connection.CloseWithError(appCode, err.Error()); err != nil {
		return err
	}

	return nil
}

func (s *Server) MetricHandler() http.Handler {
	return promhttp.Handler()
}
