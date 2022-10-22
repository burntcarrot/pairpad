package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type message struct {
	Username string `json:"username"`
	Text     string `json:"text"`
	Type     string `json:"type"`
	ID       uuid.UUID
}

// Upgrader instance to upgrade all HTTP connections to a WebSocket.
var upgrader = websocket.Upgrader{}

// Map to store currently active client connections.
var activeClients = make(map[*websocket.Conn]uuid.UUID)

// Channel for client messages.
var messageChan = make(chan message)

func main() {
	// Parse flags.
	addr := flag.String("addr", ":9000", "Server's network address")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleConn)

	// Handle incoming messages.
	go handleMsg()

	// Start the server.
	log.Printf("Starting server on %s", *addr)
	err := http.ListenAndServe(*addr, mux)
	if err != nil {
		log.Fatal("Error starting server, exiting.", err)
	}
}

// handleConn handles incoming HTTP connections by adding the connection to activeClients and reads messages from the connection.
func handleConn(w http.ResponseWriter, r *http.Request) {
	// Upgrade incoming HTTP connections to WebSocket connections
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading connection to websocket: %v", err)
	}
	defer conn.Close()

	// Generate a UUID for the client.
	activeClients[conn] = uuid.New()

	for {
		var msg message

		// Read message from the connection.
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Closing connection with ID: %v", activeClients[conn])
			delete(activeClients, conn)
			break
		}

		// Set message ID
		msg.ID = activeClients[conn]

		// Send message to messageChan.
		messageChan <- msg
	}
}

// handleMsg listens to the messageChan channel and broadcasts messages to other clients.
func handleMsg() {
	for {
		// Get message from messageChan.
		msg := <-messageChan

		// Log each message to stdout.
		t := time.Now().Format(time.ANSIC)
		if msg.Type != "" {
			color.Green("%s >> %s %s\n", t, msg.Username, msg.Text)
		} else {
			color.Green("%s >> %s: %s\n", t, msg.Username, msg.Text)
		}

		// Broadcast to all active clients.
		for client, UUID := range activeClients {
			// Check the UUID to prevent sending messages to their origin.
			if msg.ID != UUID {
				// Write JSON message.
				err := client.WriteJSON(msg)
				if err != nil {
					log.Printf("Error sending message to client: %v", err)
					client.Close()
					delete(activeClients, client)
				}
			}
		}
	}
}
