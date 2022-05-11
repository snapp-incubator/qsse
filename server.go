package qsse

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/go-errors/errors"
	"github.com/lucas-clemente/quic-go"
	"log"
)

// DELIMITER is the delimiter used to separate messages in streams.
const DELIMITER = '\n'

// Server is the main struct for the server.
type Server struct {
	listener     quic.Listener
	EventSources map[string]*EventSource

	Authenticate func(token string) bool
}

// DefaultAuthenticationFunc is the default authentication function. it accepts all clients.
var DefaultAuthenticationFunc = func(token string) bool {
	return true
}

// NewServer creates a new server and listen for connections on the given address.
func NewServer(address string, tlsConfig *tls.Config, topics []string) (*Server, error) {

	listener, err := quic.ListenAddr(address, tlsConfig, nil)
	if err != nil {
		return nil, errors.Errorf("failed to listen at address %s: %s", address, err.Error())
	}

	server := Server{
		listener:     listener,
		Authenticate: DefaultAuthenticationFunc,
		EventSources: make(map[string]*EventSource),
	}

	server.generateEventSources(topics)

	go server.acceptClients()

	return &server, nil
}

// PublishEvent publishes an event to all the subscribers of the given topic.
func (s *Server) PublishEvent(topic string, event []byte) {
	if source, ok := s.EventSources[topic]; ok {
		source.DataChannel <- event
	}
}

// SetAuthenticationFunc replaces the authentication function.
func (s *Server) SetAuthenticationFunc(authenticateFunc func(token string) bool) {
	s.Authenticate = authenticateFunc
}

// acceptClients accepts clients and do the following steps.
// 1. Accept a receivedStream.
// 2. Read client authentication token and topics.
// 3. Authenticate the client.
// 3.1 If the authentication is successful, opens sendStream for each topic and add them to eventSources.
// 3.2 If the authentication is not successful, closes the connection.
func (s *Server) acceptClients() {
	for {
		background := context.Background()
		connection, err := s.listener.Accept(background)
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
			writeData(errEvent, sendStream)
		}
	}
}

// generateEventSources generates eventSources for each topic.
func (s *Server) generateEventSources(topics []string) {
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
