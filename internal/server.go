package internal

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"runtime"

	"github.com/lucas-clemente/quic-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/snapp-incubator/qsse/auth"
)

// DELIMITER is the delimiter used to separate messages in streams.
const DELIMITER = '\n'

// Server is the main struct for the server.
type Server struct {
	Worker       Worker
	Listener     quic.Listener
	EventSources map[string]*EventSource
	Topics       []string

	Authenticator auth.Authenticator
	Authorizer    auth.Authorizer
	Metrics       Metrics
}

// DefaultAuthenticationFunc is the default authentication function. it accepts all clients.
func DefaultAuthenticationFunc(token string) bool {
	return true
}

// DefaultAuthenticationFunc is the default authorization function. it accepts all clients.
func DefaultAuthorizationFunc(token, topic string) bool {
	return true
}

// Publish publishes an event to all the subscribers of the given topic.
func (s *Server) Publish(topic string, event []byte) {
	s.Metrics.IncPublishEvent()

	matchedTopics := FindTopicsList(s.Topics, topic)
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

// SetAuthenticatorFunc replaces the authentication function.
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
	runtime.LockOSThread()

	for {
		background := context.Background()

		connection, err := s.Listener.Accept(background)
		if err != nil {
			log.Printf("failed to accept new client: %+v\n", err)

			continue
		}

		log.Println("found a new client")

		go s.handleClient(connection)
	}
}

// handleClient authenticate client and If the authentication is successful,
// opens sendStream for each topic and add them to eventSources.
func (s *Server) handleClient(connection quic.Connection) {
	offer, err := AcceptOffer(connection)
	if err != nil {
		log.Printf("failed to handle new subscriber: %+v\n", err)

		return
	}

	isValid := s.Authenticator.Authenticate(offer.Token)
	if !isValid {
		log.Println("client is not valid")

		CloseClientConnection(connection, CodeNotAuthorized, ErrNotAuthorized)

		return
	}

	log.Println("client is authenticated")

	sendStream, err := connection.OpenUniStream()
	if err != nil {
		log.Printf("failed to open send stream to client: %+v\n", err)
		CloseClientConnection(connection, CodeUnknown, err)

		return
	}

	s.addClientTopicsToEventSources(offer, sendStream)
}

// addClientTopicsToEventSources adds the client's sendStream to the eventSources.
func (s *Server) addClientTopicsToEventSources(offer *Offer, sendStream quic.SendStream) {
	for _, topic := range offer.Topics {
		valid, err := s.isTopicValid(offer, sendStream, topic)
		if err != nil {
			log.Printf("failed to send error to client: %+v\n", err)

			break
		}

		if valid {
			s.EventSources[topic].Subscribers = append(s.EventSources[topic].Subscribers, sendStream)
			s.Metrics.IncSubscriber(topic)
		}
	}
}

// isTopicValid check whether topic exists and client is authorized on it or not.
func (s *Server) isTopicValid(offer *Offer, sendStream quic.SendStream, topic string) (bool, error) {
	if _, ok := s.EventSources[topic]; !ok {
		log.Printf("topic doesn't exists: %s\n", topic)

		err := SendError(sendStream, NewErr(CodeTopicNotAvailable, map[string]any{"topic": topic}))

		return false, err
	}

	if !s.Authorizer.Authorize(offer.Token, topic) {
		log.Printf("client is not authorized for topic: %s", topic)

		err := SendError(sendStream, NewErr(CodeNotAuthorized, map[string]any{"topic": topic}))

		return false, err
	}

	return true, nil
}

// GenerateEventSources generates eventSources for each topic.
func (s *Server) GenerateEventSources(topics []string) {
	for _, topic := range topics {
		if _, ok := s.EventSources[topic]; !ok {
			log.Printf("creating new event source for topic %s", topic)
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

func CloseClientConnection(connection quic.Connection, code int, err error) {
	appCode := quic.ApplicationErrorCode(code)

	if err = connection.CloseWithError(appCode, err.Error()); err != nil {
		log.Printf("failed to close connection with client: %+v\n", err)
	}
}

func (s *Server) MetricHandler() http.Handler {
	return promhttp.Handler()
}
