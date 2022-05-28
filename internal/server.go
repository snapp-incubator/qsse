package internal

import (
	"context"
	"encoding/json"
	"log"

	"github.com/lucas-clemente/quic-go" //nolint:typecheck
	"github.com/snapp-incubator/qsse/auth"
)

// DELIMITER is the delimiter used to separate messages in streams.
const DELIMITER = '\n'

// Server is the main struct for the server.
type Server struct {
	Worker       Worker
	Listener     quic.Listener //nolint:typecheck
	EventSources map[string]*EventSource
	Topics       []string

	Autheticator auth.Autheticator
	Authorizer   auth.Authorizer
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
	matchedTopics := FindTopicsList(s.Topics, topic)
	for _, matchedTopic := range matchedTopics {
		if source, ok := s.EventSources[matchedTopic]; ok {
			source.DataChannel <- event
		}
	}
}

// SetAuthenticator replaces the authentication function.
func (s *Server) SetAuthenticator(authenticator auth.Autheticator) {
	s.Autheticator = authenticator
}

// SetAuthenticatorFunc replaces the authentication function.
func (s *Server) SetAuthenticatorFunc(authenticator auth.AutheticatorFunc) {
	s.Autheticator = authenticator
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
	for {
		background := context.Background()
		connection, err := s.Listener.Accept(background)
		checkError(err)
		log.Println("found a new client")

		client := NewSubscriber(connection)
		go s.handleClient(client)
	}
}

// handleClient authenticate client and If the authentication is successful,
// opens sendStream for each topic and add them to eventSources.
func (s *Server) handleClient(client *Subscriber) {
	isValid := s.Autheticator.Authenticate(client.Token)
	if !isValid {
		log.Println("client is not authenticated")

		err := client.connection.CloseWithError(quic.ApplicationErrorCode(CodeNotAuthorized), ErrNotAuthorized.Error()) //nolint:typecheck
		checkError(err)

		return
	}

	log.Println("client is authenticated")

	sendStream, err := client.connection.OpenUniStream()
	checkError(err)

	s.addClientTopicsToEventSources(client, sendStream)
}

// addClientTopicsToEventSources adds the client's sendStream to the eventSources.
func (s *Server) addClientTopicsToEventSources(client *Subscriber, sendStream quic.SendStream) { //nolint:typecheck
	for _, topic := range client.Topics {
		if ok := s.Authorizer.Authorize(client.Token, topic); !ok {
			log.Printf("client is not authorized for %s", topic)

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
			checkError(err)
		}
	}
}

// GenerateEventSources generates eventSources for each topic.
func (s *Server) GenerateEventSources(topics []string) {
	for _, topic := range topics {
		if _, ok := s.EventSources[topic]; !ok {
			log.Printf("creating new event source for topic %s", topic)
			s.EventSources[topic] = NewEventSource(topic, make(chan []byte), []quic.SendStream{}) //nolint:typecheck

			go s.EventSources[topic].TransferEvents(s.Worker)
		}
	}
}

func checkError(err error) {
	if err != nil {
		log.Println(err.Error())
	}
}
