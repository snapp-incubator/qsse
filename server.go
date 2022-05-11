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

const DELIMITER = '\n'

type Server struct {
	listener     quic.Listener
	EventSources map[string]*EventSource

	Authenticate func(token string) bool
}

var DefaultAuthenticationFunc = func(token string) bool {
	return true
}

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

func (s *Server) PublishEvent(topic string, event []byte) {
	if source, ok := s.EventSources[topic]; ok {
		source.DataChannel <- event
	}
}

func (s *Server) SetAuthenticationFunc(authenticateFunc func(token string) bool) {
	s.Authenticate = authenticateFunc
}

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
