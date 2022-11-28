package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/burntcarrot/rowix/crdt"
	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type message struct {
	Username  string `json:"username"`
	Text      string `json:"text"`
	Type      string `json:"type"`
	ID        uuid.UUID
	Operation Operation      `json:"operation"`
	Document  *crdt.Document `json:"document"`
}

type Operation struct {
	Type     string `json:"type"`
	Position int    `json:"position"`
	Value    string `json:"value"`
}

// Upgrader instance to upgrade all HTTP connections to a WebSocket.
var upgrader = websocket.Upgrader{}

// Map to store currently active client connections.
var activeClients = make(map[*websocket.Conn]uuid.UUID)

// Channel for client messages.
var messageChan = make(chan message)

func main() {
	// Parse flags.
	addr := flag.String("addr", ":8080", "Server's network address")
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
		color.Red("Error upgrading connection to websocket: %v\n", err)
	}
	defer conn.Close()

	// doc := crdt.New()
	var doc *crdt.Document
	color.Yellow("total active clients: %d\n", len(activeClients))
	if len(activeClients) > 1 {
		// at least 2 clients for requesting a document
		for clientConn, _ := range activeClients {
			// send a docReq message to a client
			msg := message{Type: "docReq"}
			err = clientConn.WriteJSON(&msg)
			if err != nil {
				color.Red("Failed to send docReq: %v\n", err)
			}

			// wait for a client to send a document
			err = clientConn.ReadJSON(&msg)
			if err != nil {
				color.Red("Failed to receive document: %v, msg: %+v\n", err, msg)
			}
			doc = msg.Document
			break
		}

		msg := message{Type: "syncResp", Document: doc}
		err = conn.WriteJSON(&msg)
		if err != nil {
			color.Red("Failed to send syncResp: %v\n", err)
		}
	}

	// Generate a UUID for the client.
	activeClients[conn] = uuid.New()

	for {
		var msg message

		// Read message from the connection.
		err := conn.ReadJSON(&msg)
		if err != nil {
			color.Red("Closing connection with ID: %v\n", activeClients[conn])
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
		if msg.Type == "info" {
			color.Green("%s >> %s %s (ID: %s)\n", t, msg.Username, msg.Text, msg.ID)
		} else if msg.Type == "syncReq" {
			color.Green("%s >> syncReq sent from ID: %s\n", t, msg.ID)
		} else if msg.Type == "operation" {
			color.Green("operation >> %+v from ID=%s\n", msg.Operation, msg.ID)
		} else {
			color.Green("%s >> %+v\n", t, msg)
		}

		// Broadcast to all active clients.
		for client, UUID := range activeClients {
			// Check the UUID to prevent sending messages to their origin.
			if UUID != msg.ID {
				// Write JSON message.
				color.Magenta("writing message to: %s, msg: %+v\n", UUID, msg)
				err := client.WriteJSON(msg)
				if err != nil {
					color.Red("Error sending message to client: %v\n", err)
					client.Close()
					delete(activeClients, client)
				}
			}
		}
	}
}
