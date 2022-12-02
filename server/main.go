package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/burntcarrot/rowix/crdt"
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

// var siteIDs = make(map[string]uuid.UUID)

var clientInfos = make(map[uuid.UUID]clientInfo)

// Channel for client messages.
var messageChan = make(chan message)

var docChan = make(chan message)

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

	color.Yellow("total active clients: %d\n", len(activeClients))

	// Generate a UUID for the client and add client connection to the map.
	newID := uuid.New()
	activeClients[conn] = newID

	// generate a unique siteID and assign send it to the client
	siteID := strconv.Itoa(len(clientInfos))
	clientInfos[newID] = clientInfo{SiteID: siteID}
	color.Magenta("clientInfos after SiteID generation: %+v", clientInfos)
	color.Yellow("Assigning siteID: %+v", clientInfos[newID])
	siteIDMsg := message{Type: "SiteID", Text: clientInfos[newID].SiteID, ID: newID}
	if err := conn.WriteJSON(siteIDMsg); err != nil {
		color.Red("ERROR: didn't send siteID message")
	}

	// send a document request to an existing client
	for clientConn, id := range activeClients {
		if id != newID {
			msg := message{Type: "docReq", ID: newID}
			color.Cyan("sending docReq to %s on behalf of %s", id, newID)
			err = clientConn.WriteJSON(&msg)
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
			color.Red("Closing connection with ID: %v\n", activeClients[conn])
			delete(activeClients, conn)
			break
		}

		// Set message ID
		msg.ID = activeClients[conn]

		// Send docResp to handleSync function
		if msg.Type == "docResp" {
			docChan <- msg
			continue
		} else {
			// Set message ID
			msg.ID = activeClients[conn]
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
		if msg.Type == "info" {
			// color.Green("%s >> %s %s (ID: %s)\n", t, clientInfos[msg.ID], msg.Text, msg.ID)
			color.Green("%s >> %s %s (ID: %s)\n", t, msg.Username, msg.Text, msg.ID)
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

func handleSync() {
	for {
		docRespMsg := <-docChan
		color.Cyan("got docRespMsg with ID %s", docRespMsg.ID)
		for client, UUID := range activeClients {
			color.Yellow("client: %s", UUID)
			if UUID == docRespMsg.ID {
				color.Cyan("sending docResp to %s", docRespMsg.ID)
				client.WriteJSON(docRespMsg)
			}
		}

	}
}
