package internal

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	quic "github.com/quic-go/quic-go"
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
	Finder       Finder

	Authenticator auth.Authenticator
	Authorizer    auth.Authorizer
	Metrics       Metrics

	CleaningInterval time.Duration
}

// DefaultAuthenticationFunc is the default authentication function. it accepts all clients.
func DefaultAuthenticationFunc(_ string) bool {
	return true
}

// DefaultAuthorizationFunc is the default authorization function. it accepts all clients.
func DefaultAuthorizationFunc(_, _ string) bool {
	return true
}

// Publish publishes an event to all the subscribers of the given topic.
func (s *Server) Publish(topic string, event []byte) {
	matchedTopics := s.Finder.FindTopicsList(s.Topics, topic)
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

// handleClient authenticate client and If the authentication is successful,
// opens sendStream for each topic and add them to eventSources.
func (s *Server) handleClient(connection quic.Connection) {
	offer, err := AcceptOffer(connection)
	if err != nil {
		s.Logger.Error("failed to handle new subscriber", zap.Error(err))

		return
	}

	isValid := s.Authenticator.Authenticate(offer.Token)
	if !isValid {
		s.Logger.Warn("client is not valid")

		err := CloseClientConnection(connection, CodeNotAuthorized, ErrNotAuthorized)
		if err != nil {
			s.Logger.Error("failed to close connection with client", zap.Error(err))
		}

		return
	}

	s.Logger.Info("client is authenticated")

	sendStream, err := connection.OpenUniStream()
	if err != nil {
		s.Logger.Error("failed to open send stream to client", zap.Error(err))

		er := CloseClientConnection(connection, CodeUnknown, err)
		if er != nil {
			s.Logger.Error("failed to close connection with client", zap.Error(err))
		}

		return
	}

	subscriber := NewSubscriber(sendStream)

	s.addClientTopicsToEventSources(offer, subscriber)
}

// addClientTopicsToEventSources adds the client's sendStream to the eventSources.
func (s *Server) addClientTopicsToEventSources(offer *Offer, subscriber Subscriber) {
	for _, topic := range offer.Topics {
		valid, err := s.isTopicValid(offer, subscriber.Stream, topic)
		if err != nil {
			s.Logger.Error("failed to send error to client", zap.Error(err))

			break
		}

		if valid {
			s.EventSources[topic].IncomingSubscribers <- subscriber
			s.Metrics.IncSubscriber(topic)
		}
	}
}

// isTopicValid check whether topic exists and client is authorized on it or not.
func (s *Server) isTopicValid(offer *Offer, sendStream quic.SendStream, topic string) (bool, error) {
	if _, ok := s.EventSources[topic]; !ok {
		s.Logger.Warn("topic doesn't exists", zap.String("topic", topic))

		err := SendError(sendStream, NewErr(CodeTopicNotAvailable, map[string]any{"topic": topic}))

		return false, err
	}

	if !s.Authorizer.Authorize(offer.Token, topic) {
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
			s.EventSources[topic] = NewEventSource(
				topic,
				make(chan []byte),
				make([]Subscriber, 0),
				s.Metrics,
				s.CleaningInterval,
			)

			go s.EventSources[topic].DistributeEvents(s.Worker)
			go s.EventSources[topic].CleanCorruptSubscribers()
			go s.EventSources[topic].HandleNewSubscriber()
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
