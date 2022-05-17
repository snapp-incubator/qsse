package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/lucas-clemente/quic-go"
	"log"
)

// DELIMITER is the delimiter used to separate messages in streams.
const DELIMITER = '\n'

// Server is the main struct for the server.
type Server struct {
	Listener     quic.Listener
	EventSources map[string]*EventSource

	Authenticate func(token string) bool
}

// DefaultAuthenticationFunc is the default authentication function. it accepts all clients.
var DefaultAuthenticationFunc = func(token string) bool {
	return true
}

// Publish publishes an event to all the subscribers of the given topic.
func (s *Server) Publish(topic string, event []byte) {
	if source, ok := s.EventSources[topic]; ok {
		source.DataChannel <- event
	}
}

// SetAuthentication replaces the authentication function.
func (s *Server) SetAuthentication(authenticateFunc func(token string) bool) {
	s.Authenticate = authenticateFunc
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
	isValid := s.Authenticate(client.Token)
	if !isValid {
		log.Println("client is not authenticated")
		client.connection.CloseWithError(quic.ApplicationErrorCode(CodeNotAuthorized), ErrNotAuthorized.Error())

		return
	}

	log.Println("client is authenticated")

	sendStream, err := client.connection.OpenUniStream()
	checkError(err)

	s.addClientTopicsToEventSources(client, sendStream)

}

// addClientTopicsToEventSources adds the client's sendStream to the eventSources.
func (s *Server) addClientTopicsToEventSources(client *Subscriber, sendStream quic.SendStream) {
	for _, topic := range client.Topics {
		if _, ok := s.EventSources[topic]; ok {
			s.EventSources[topic].Subscribers = append(s.EventSources[topic].Subscribers, sendStream)
		} else {
			errBytes, _ := json.Marshal(ErrTopicNotAvailable(topic))
			errEvent := NewEvent(ErrorTopic, errBytes)
			WriteData(errEvent, sendStream)
		}
	}
}

// GenerateEventSources generates eventSources for each topic.
func (s *Server) GenerateEventSources(topics []string) {
	for _, topic := range topics {
		if _, ok := s.EventSources[topic]; !ok {
			log.Printf("creating new event source for topic %s", topic)
			s.EventSources[topic] = NewEventSource(topic, make(chan []byte), *new([]quic.SendStream))

			go s.EventSources[topic].transferEvents()
		}
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err.Error())
	}
}