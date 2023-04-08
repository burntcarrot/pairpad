package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/burntcarrot/pairpad/crdt"
	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type message struct {
	Username  string        `json:"username"`
	Text      string        `json:"text"`
	Type      string        `json:"type"`
	ID        uuid.UUID     `json:"ID"`
	Operation Operation     `json:"operation"`
	Document  crdt.Document `json:"document"`
}

type clientInfo struct {
	Username string `json:"username"`
	SiteID   string `json:"siteID"`
	Conn     *websocket.Conn
}

type Operation struct {
	Type     string `json:"type"`
	Position int    `json:"position"`
	Value    string `json:"value"`
}

// Monotonically increasing site ID.
var siteID = 0

// Mutex for protecting site ID increment operations.
var mu sync.Mutex

// Upgrader instance to upgrade all HTTP connections to a WebSocket.
var upgrader = websocket.Upgrader{}

// Map to store active client connections.
var activeClients = make(map[uuid.UUID]clientInfo)

// Channel for client messages.
var messageChan = make(chan message)

// Channel for document sync messages.
var syncChan = make(chan message)

func main() {
	// Parse flags.
	addr := flag.String("addr", ":8080", "Server's network address")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleConn)

	// Handle document syncing
	go handleSync()

	// Handle incoming messages.
	go handleMsg()

	// Start the server.
	log.Printf("Starting server on %s", *addr)

	server := &http.Server{
		Addr:         *addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      mux,
	}

	err := server.ListenAndServe()
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

	color.Yellow("total active clients: %d\n", len(activeClients))

	// Generate the UUID and the site ID for the client.
	clientID := uuid.New()

	// Carefully increment site ID with mutexes.
	mu.Lock()
	siteID++
	mu.Unlock()

	siteID := strconv.Itoa(siteID)

	// Add the client to the map of active clients.
	c := clientInfo{Conn: conn, SiteID: siteID}
	activeClients[clientID] = c

	color.Magenta("activeClients after SiteID generation: %+v", activeClients)
	color.Yellow("Assigning siteID: %s", c.SiteID)

	// Generate a Site ID message.
	siteIDMsg := message{Type: "SiteID", Text: c.SiteID, ID: clientID}
	if err := conn.WriteJSON(siteIDMsg); err != nil {
		color.Red("ERROR: didn't send siteID message")
	}

	// send a document request to an existing client
	for id, clientInfo := range activeClients {
		if id != clientID {
			msg := message{Type: "docReq", ID: clientID}
			color.Cyan("sending docReq to %s on behalf of %s", id, clientID)
			err = clientInfo.Conn.WriteJSON(&msg)
			if err != nil {
				color.Red("Failed to send docReq: %v\n", err)
				continue
			}
			break
		}
	}

	// read messages from the connection and send to channel to broadcast
	for {
		var msg message

		// Read message from the connection.
		err := conn.ReadJSON(&msg)
		if err != nil {
			color.Red("Closing connection with username: %v\n", activeClients[clientID].Username)
			delete(activeClients, clientID)
			break
		}

		// Set message ID
		msg.ID = clientID

		// Send docSync to handleSync function
		if msg.Type == "docSync" {
			syncChan <- msg
			continue
		}

		// Send message to messageChan for logging and broadcasting
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
		if msg.Type == "join" {
			// Set the username received from the client to the clientInfo present in activeClients.
			clientInfo := activeClients[msg.ID]
			clientInfo.Username = msg.Username
			activeClients[msg.ID] = clientInfo

			color.Green("%s >> %s %s (ID: %s)\n", t, msg.Username, msg.Text, msg.ID)
		} else if msg.Type == "operation" {
			color.Green("operation >> %+v from ID=%s\n", msg.Operation, msg.ID)
		} else {
			color.Green("%s >> %+v\n", t, msg)
		}

		// Broadcast to all active clients.
		for id, clientInfo := range activeClients {
			// Check the UUID to prevent sending messages to their origin.
			if id != msg.ID {
				// Write JSON message.
				color.Magenta("writing message to: %s, msg: %+v\n", id, msg)
				err := clientInfo.Conn.WriteJSON(msg)
				if err != nil {
					color.Red("Error sending message to client: %v\n", err)
					clientInfo.Conn.Close()
					delete(activeClients, id)
				}
			}
		}
	}
}

func handleSync() {
	for {
		// Receive document response.
		syncMsg := <-syncChan
		color.Cyan("got syncMsg, len(document) = %d\n", len(syncMsg.Document.Characters))
		for UUID, clientInfo := range activeClients {
			if UUID != syncMsg.ID {
				color.Cyan("sending syncMsg to %s", syncMsg.ID)
				_ = clientInfo.Conn.WriteJSON(syncMsg)
			}
		}
	}
}
