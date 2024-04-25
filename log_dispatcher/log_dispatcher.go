package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"golang.org/x/net/websocket"
)

type dispatchServer struct {
	serveMux        http.ServeMux
	subscribersLock sync.Mutex
	subscribers     map[*subscriber]bool
	messageBuffer   int
	logfile         string
	producer        sarama.AsyncProducer
}

type subscriber struct {
	messages chan []byte
}

func newDispatchServer() *dispatchServer {
	producer, err := sarama.NewAsyncProducer([]string{"kafka:9092"}, nil)
	if err != nil {
		panic(err)
	}
	ds := &dispatchServer{
		messageBuffer: 16,
		subscribers:   make(map[*subscriber]bool),
		logfile:       "service.log",
		producer:      producer,
	}
	ds.serveMux.Handle("/", http.FileServer(http.Dir("./static")))
	ds.serveMux.HandleFunc("/publish", ds.publishHandler)
	ds.serveMux.Handle("/subscribe", websocket.Handler(ds.subscribeHandler))
	return ds
}

// Handles requests to the web server's /publish endpoint.
func (ds *dispatchServer) publishHandler(w http.ResponseWriter,
	r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed)
		return
	}
	body := http.MaxBytesReader(w, r.Body, 8192)
	logJson, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge),
			http.StatusRequestEntityTooLarge)
		return
	}

	timestamp := []byte(time.Now().Format(time.RFC822Z) + " ")
	msg := append(timestamp, logJson...)
	msg = append(msg, "\n"...)

	ds.publish(msg)

	w.WriteHeader(http.StatusAccepted)
}

// Publishes the JSON log to all the various receivers.
func (ds *dispatchServer) publish(msg []byte) {
	ds.subscribersLock.Lock()
	defer ds.subscribersLock.Unlock()

	// File receiver
	go func() {
		file, err := os.OpenFile(ds.logfile,
			os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Print(err)
		}
		file.Write(msg)
	}()

	ds.producer.Input() <- &sarama.ProducerMessage{Topic: "logs", Value: sarama.StringEncoder(msg)}

	// Websocket subscribers
	for s := range ds.subscribers {
		s.messages <- msg
	}
}

// Handles requests to the web server's /subscribe endpoint. Upgrades incoming
// connections to websockets and creates listeners.
func (ds *dispatchServer) subscribeHandler(ws *websocket.Conn) {
	err := ds.subscribe(ws)
	if err == io.EOF {
		return
	} else if err != nil {
		return
	}
}

// Creates a listener for each new websocket request.
func (ds *dispatchServer) subscribe(ws *websocket.Conn) error {
	s := &subscriber{
		messages: make(chan []byte, ds.messageBuffer),
	}
	ds.addSubscriber(s)
	defer ds.deleteSubscriber(s)

	done := make(chan bool, 1)

	// Reader
	go func() {
		// Discard messages received over websocket except close notification
		var msg []byte
		for {
			_, err := ws.Read(msg)
			if err == io.EOF {
				done <- true // Signal the writer the websocket is closed
				return
			}
		}
	}()

	// Writer
	for {
		select {
		case msg := <-s.messages:
			_, err := ws.Write(msg)
			if err != nil {
				return err
			}
		case <-done:
			return io.EOF
		}
	}
}

// Adds new subscribers to the server's list.
func (ds *dispatchServer) addSubscriber(s *subscriber) {
	ds.subscribersLock.Lock()
	ds.subscribers[s] = true
	ds.subscribersLock.Unlock()
}

// Removes closed subscribers from the server's list.
func (ds *dispatchServer) deleteSubscriber(s *subscriber) {
	ds.subscribersLock.Lock()
	delete(ds.subscribers, s)
	ds.subscribersLock.Unlock()
}

func (ds *dispatchServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ds.serveMux.ServeHTTP(w, r)
}
