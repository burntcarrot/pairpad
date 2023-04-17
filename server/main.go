package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/burntcarrot/pairpad/commons"
	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type clientInfo struct {
	Username string `json:"username"`
	SiteID   string `json:"siteID"`
	Conn     *websocket.Conn
}

var (
	// Monotonically increasing site ID.
	siteID = 0

	// Mutex for protecting site ID increment operations.
	mu sync.Mutex

	// Upgrader instance to upgrade all HTTP connections to a WebSocket.
	upgrader = websocket.Upgrader{}

	// Map to store active client connections.
	activeClients = make(map[uuid.UUID]clientInfo)

	// Channel for client messages.
	messageChan = make(chan commons.Message)

	// Channel for document sync messages.
	syncChan = make(chan commons.Message)
)

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
	// Generate the UUID and the site ID for the client.
	clientID := uuid.New()

	// Upgrade incoming HTTP connections to WebSocket connections
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		color.Red("Error upgrading connection to websocket: %v\n", err)
		closeConn(clientID)
		return
	}
	defer conn.Close()

	// Carefully increment and assign site ID with mutexes.
	mu.Lock()
	siteID++

	// Add the client to the map of active clients.
	c := clientInfo{Conn: conn, SiteID: strconv.Itoa(siteID)}
	activeClients[clientID] = c
	mu.Unlock()

	color.Yellow("New client joining. Total active clients: %d\n", len(activeClients))

	color.Magenta("activeClients after SiteID generation: %+v", activeClients)
	color.Yellow("Assigning siteID: %s", c.SiteID)

	// Generate a Site ID message.
	siteIDMsg := commons.Message{Type: commons.SiteIDMessage, Text: c.SiteID, ID: clientID}
	if err := conn.WriteJSON(siteIDMsg); err != nil {
		color.Red("ERROR: didn't send siteID message")
		closeConn(clientID)
		return
	}

	// send a document request to an existing client
	for id, clientInfo := range activeClients {
		if id != clientID {
			msg := commons.Message{Type: commons.DocReqMessage, ID: clientID}
			color.Cyan("sending docReq to %s on behalf of %s", id, clientID)
			err = clientInfo.Conn.WriteJSON(&msg)
			if err != nil {
				color.Red("Failed to send docReq: %v\n", err)
				continue
			}
			break
		}
	}

	updateUsers()

	// read messages from the connection and send to channel to broadcast
	for {
		var msg commons.Message

		// Read message from the connection.
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				color.Red("Failed to read message from client %s: %v", activeClients[clientID].Username, err)
			}
			closeConn(clientID)
			return
		}

		// Set message ID
		msg.ID = clientID

		// Send docSync to handleSync function
		if msg.Type == commons.DocSyncMessage {
			syncChan <- msg
			continue
		}

		// Send message to messageChan for logging and broadcasting
		messageChan <- msg
	}
}

// closeConn cleanly closes a client connection.
func closeConn(clientID uuid.UUID) {
	if err := activeClients[clientID].Conn.Close(); err != nil {
		color.Red("Error closing connection: %v\n", err)
	}
	color.Red("Closing connection with username: %v\n", activeClients[clientID].Username)
	delete(activeClients, clientID)
	updateUsers()
}

// handleMsg listens to the messageChan channel and broadcasts messages to other clients.
func handleMsg() {
	for {
		// Get message from messageChan.
		msg := <-messageChan

		// Log each message to stdout.
		t := time.Now().Format(time.ANSIC)
		if msg.Type == commons.JoinMessage {
			// Set the username received from the client to the clientInfo present in activeClients.
			clientInfo := activeClients[msg.ID]
			clientInfo.Username = msg.Username
			activeClients[msg.ID] = clientInfo

			color.Green("%s >> %s %s (ID: %s)\n", t, msg.Username, msg.Text, msg.ID)
			updateUsers()
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
					closeConn(id)
				}
			}
		}
	}
}

func handleSync() {
	for {
		syncMsg := <-syncChan
		switch syncMsg.Type {
		case commons.DocSyncMessage:
			// Receive document response.
			color.Cyan("got syncMsg, len(document) = %d\n", len(syncMsg.Document.Characters))
			for UUID, clientInfo := range activeClients {
				if UUID != syncMsg.ID {
					color.Cyan("sending syncMsg to %s", syncMsg.ID)
					if err := clientInfo.Conn.WriteJSON(syncMsg); err != nil {
						color.Red("failed to send syncMsg to %s", UUID)
					}
				}
			}
		case commons.UsersMessage:
			for UUID, clientInfo := range activeClients {
				if err := clientInfo.Conn.WriteJSON(syncMsg); err != nil {
					color.Red("failed to send userMsg to %s", UUID)
				}
			}
		}
	}
}

func updateUsers() {
	var users string
	for _, ci := range activeClients {
		users += ci.Username + ","
	}

	syncChan <- commons.Message{Text: users, Type: commons.UsersMessage}
	// return commons.Message{Text: users, Type: commons.UsersMessage}
}
