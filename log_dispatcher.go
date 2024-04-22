package main

import (
	"io"
	"log"
	models "log_dispatcher/models"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/websocket"
)

var port string = ":8080"                  // listening port
var post_endpoint string = "/log"          // API endpoint for post
var feed_endpoint string = "/feed"         // API endpoint for feed page
var websocket_endpoint string = "/feed_ws" // API endpoint for ws feed
var logfile string = "test.log"            // Logfile name
var log_channel chan models.LogInfo = make(chan models.LogInfo)
var all_connections map[*websocket.Conn]bool = make(map[*websocket.Conn]bool)

// The function to handle serving the static feed page.
func feed_page_handler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/feed.html")
}

// The post function which accepts arbitrary JSON data and writes that data
// plus a timestamp to a file.
func post_handler(w http.ResponseWriter, r *http.Request) {
	// Check for and disallow methods other than POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read JSON request body
	body, post_err := io.ReadAll(r.Body)
	if post_err != nil {
		http.Error(w, "Failed to read request body",
			http.StatusInternalServerError)
		return
	}

	log_line := models.LogInfo{
		JsonLog:   string(body),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Respond -- request received and read
	w.WriteHeader(http.StatusOK)

	// Get file handler -- create file if it doesn't exist
	file, file_err := os.OpenFile(
		logfile,
		os.O_WRONLY|os.O_APPEND|os.O_CREATE,
		0644,
	)
	if file_err != nil {
		log.Panic(file_err)
	}
	defer file.Close()

	// Write timestamp & request body, one per line
	file.Write([]byte(log_line.String()))

	// Send the log entry to the channel for distribution.
	log_channel <- log_line
}

// The function to handle sending log messages from the post endpoint back out
// to a monitoring feed.
func websocket_handler(ws *websocket.Conn) {
	all_connections[ws] = true // Keep a list of active connections
	for {
		// Whenever a new log line comes into the distribution channel, send
		// a message out to anybody viewing the feed page.
		log_line := <-log_channel
		for connection := range all_connections {
			_, err := connection.Write([]byte(log_line.String()))
			if err != nil {
				// If there's an error with any of the connections, remove it.
				connection.Close()
				delete(all_connections, connection)
			}
		}
	}
}

func main() {
	http.HandleFunc(post_endpoint, post_handler)
	http.HandleFunc(feed_endpoint, feed_page_handler)
	http.Handle(websocket_endpoint, websocket.Handler(websocket_handler))
	log.Fatal(http.ListenAndServe(port, nil))
}
