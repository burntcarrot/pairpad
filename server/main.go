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

// Clients is used to store, reference, and update information about all connected clients.
type Clients struct {
	// list stores information about active clients.
	list map[uuid.UUID]*client

	// clientsMu protects against concurrent read/write access to the activeClients map.
	mu sync.RWMutex

	// deleteRequests indicates which clients to delete from the list of clients.
	deleteRequests chan deleteRequest

	// readRequests indicates which clients to retrieve from the list of clients.
	readRequests chan readRequest

	// addRequests is used to send clients to add to the list of clients.
	addRequests chan *client

	// nameUpdateRequests is used to update a client with their username.
	nameUpdateRequests chan nameUpdate
}

// a client holds the information of a connected client.
type client struct {
	Conn     *websocket.Conn
	SiteID   string `json:"siteID"`
	id       uuid.UUID
	writeMu  sync.Mutex
	Username string `json:"username"`
}

var (
	// Monotonically increasing site ID, unique to each client.
	siteID = 0

	// Mutex for protecting site ID increment operations.
	mu sync.Mutex

	// Upgrader instance to upgrade all HTTP connections to a WebSocket.
	upgrader = websocket.Upgrader{}

	// Channel for client messages.
	messageChan = make(chan commons.Message)

	// Channel for document sync messages.
	syncChan = make(chan commons.Message)

	// Holds information about all clients.
	clients = NewClients()
)

func main() {
	addr := flag.String("addr", ":8080", "Server's network address")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleConn)

	// Handle state of client information.
	go clients.handle()

	// Handle incoming messages.
	go handleMsg()

	// Handle document syncing
	go handleSync()

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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		color.Red("Error upgrading connection to websocket: %v\n", err)
		conn.Close()
		return
	}
	defer conn.Close()

	clientID := uuid.New()

	// Carefully increment and assign site ID with mutexes.
	mu.Lock()
	siteID++

	client := &client{
		Conn:    conn,
		SiteID:  strconv.Itoa(siteID),
		id:      clientID,
		writeMu: sync.Mutex{},
	}
	mu.Unlock()

	clients.add(client)

	siteIDMsg := commons.Message{Type: commons.SiteIDMessage, Text: client.SiteID, ID: clientID}
	clients.broadcastOne(siteIDMsg, clientID)

	docReq := commons.Message{Type: commons.DocReqMessage, ID: clientID}
	clients.broadcastOneExcept(docReq, clientID)

	clients.sendUsernames()

	// Read messages from the connection and send to channel to broadcast
	for {
		var msg commons.Message
		if err := client.read(&msg); err != nil {
			color.Red("Failed to read message. closing client connection with %s. Error: %s", client.Username, err)
			return
		}

		// Send docSync to handleSync function. DocSync message IDs refer to
		// their destination. This channel send should happen before reassigning the
		// msg.ID
		if msg.Type == commons.DocSyncMessage {
			syncChan <- msg
			continue
		}

		// Set message ID as the ID of the sending client. Most message IDs refer to
		// their origin.
		msg.ID = clientID

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
		if msg.Type == commons.JoinMessage {
			clients.updateName(msg.ID, msg.Username)
			color.Green("%s >> %s %s (ID: %s)\n", t, msg.Username, msg.Text, msg.ID)
			clients.sendUsernames()
		} else if msg.Type == "operation" {
			color.Green("operation >> %+v from ID=%s\n", msg.Operation, msg.ID)
		} else {
			color.Green("%s >> unknown message type:  %v\n", t, msg)
			clients.sendUsernames()
			continue
		}

		clients.broadcastAllExcept(msg, msg.ID)
	}
}

// handleSync reads from the syncChan and sends the message to the appropriate user(s).
func handleSync() {
	for {
		syncMsg := <-syncChan
		switch syncMsg.Type {
		case commons.DocSyncMessage:
			clients.broadcastOne(syncMsg, syncMsg.ID)
		case commons.UsersMessage:
			color.Blue("usernames: %s", syncMsg.Text)
			clients.broadcastAll(syncMsg)
		}
	}
}

// handle acts as a monitor for a Clients type. handle attempts to ensure concurrency safety
// for accessing the Clients struct.
func (c *Clients) handle() {
	for {
		select {
		case req := <-c.deleteRequests:
			c.close(req.id)
			req.done <- 1
			close(req.done)
		case req := <-c.readRequests:
			if req.readAll {
				for _, client := range c.list {
					req.resp <- client
				}
				close(req.resp)
			} else {
				req.resp <- c.list[req.id]
				close(req.resp)
			}
		case client := <-c.addRequests:
			c.mu.Lock()
			c.list[client.id] = client
			c.mu.Unlock()
		case n := <-c.nameUpdateRequests:
			c.list[n.id].Username = n.newName
		}
	}
}

// A deleteRequest is used to delete clients from the list of clients.
type deleteRequest struct {
	// id is the ID of the client to be deleted.
	id uuid.UUID

	// done is used to signal that a delete request has been fulfilled.
	done chan int
}

// A readRequest is used to help callers retrieve information about clients.
type readRequest struct {
	// readAll indicates whether the caller want's to receive all clients.
	readAll bool

	// id is the id of the client to be retrieved from the list of clients. id is the
	//  zero value of uuid.UUID if readAll is true.
	id uuid.UUID

	// resp is the channel from which requesters can read the response.
	resp chan *client
}

// NewClients returns a new instance of a Clients struct.
func NewClients() *Clients {
	return &Clients{
		list:               make(map[uuid.UUID]*client),
		mu:                 sync.RWMutex{},
		deleteRequests:     make(chan deleteRequest),
		readRequests:       make(chan readRequest),
		addRequests:        make(chan *client),
		nameUpdateRequests: make(chan nameUpdate),
	}
}

// getAll requests all active clients from the clients list and returns a channel containing
// all clients.
func (c *Clients) getAll() chan *client {
	c.mu.RLock()
	resp := make(chan *client, len(c.list))
	c.mu.RUnlock()
	c.readRequests <- readRequest{readAll: true, resp: resp}
	return resp
}

// get requests a client with the given id, and returns a channel containing the client. If
// the client doesn't exist, the channel will be empty
func (c *Clients) get(id uuid.UUID) chan *client {
	resp := make(chan *client)

	c.readRequests <- readRequest{readAll: false, id: id, resp: resp}
	return resp
}

// add adds a client to the list of clients.
func (c *Clients) add(client *client) {
	c.addRequests <- client
}

// A nameUpdate is used as a message to update the name of a client.
type nameUpdate struct {
	id      uuid.UUID
	newName string
}

// updateName updates the name field of a client with the given id.
func (c *Clients) updateName(id uuid.UUID, newName string) {
	c.nameUpdateRequests <- nameUpdate{id, newName}
}

// delete deletes a client from the list of active clients.
func (c *Clients) delete(id uuid.UUID) {
	req := deleteRequest{id, make(chan int)}
	c.deleteRequests <- req
	<-req.done
	c.sendUsernames()
}

// broadcastAll sends a message to all active clients.
func (c *Clients) broadcastAll(msg commons.Message) {
	color.Blue("sending message to all users. Text: %s", msg.Text)
	for client := range c.getAll() {
		if err := client.send(msg); err != nil {
			color.Red("ERROR: %s", err)
			c.delete(client.id)
		}
	}
}

// broadcastAllExcept sends a message to all clients except for the one whose ID
// matches except.
func (c *Clients) broadcastAllExcept(msg commons.Message, except uuid.UUID) {
	for client := range c.getAll() {
		if client.id == except {
			continue
		}
		if err := client.send(msg); err != nil {
			color.Red("ERROR: %s", err)
			c.delete(client.id)
		}
	}
}

// broadcastOne sends a message to a single client with the ID matching dst.
func (c *Clients) broadcastOne(msg commons.Message, dst uuid.UUID) {
	client := <-c.get(dst)
	if err := client.send(msg); err != nil {
		color.Red("ERROR: %s", err)
		c.delete(client.id)
	}
}

// broadcastOneExcept sends a message to any one client whose ID does not match except.
func (c *Clients) broadcastOneExcept(msg commons.Message, except uuid.UUID) {
	for client := range c.getAll() {
		if client.id == except {
			continue
		}
		if err := client.send(msg); err != nil {
			color.Red("ERROR: %s", err)
			c.delete(client.id)
			continue
		}
		break
	}
}

// close closes a WebSocket connection and removes it from the list of clients in a
// concurrency safe manner.
func (c *Clients) close(id uuid.UUID) {
	c.mu.RLock()
	client, ok := c.list[id]
	if ok {
		if err := client.Conn.Close(); err != nil {
			color.Red("Error closing connection: %s\n", err)
		}
	} else {
		color.Red("Couldn't close connection: client not in list")
		return
	}
	color.Red("Removing %v from client list.\n", c.list[id].Username)
	c.mu.RUnlock()

	c.mu.Lock()
	delete(c.list, id)
	c.mu.Unlock()

}

// read reads a message over the client Conn, and stores the result in msg.
func (c *client) read(msg *commons.Message) error {
	err := c.Conn.ReadJSON(msg)
	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			color.Red("Failed to read message from client %s: %v", c.Username, err)
		}
		color.Red("client %v disconnected", c.Username)
		clients.delete(c.id)
		return err
	}
	return nil
}

// send sends a message over the client Conn while protecting from
// concurrent writes.
func (c *client) send(v interface{}) error {
	c.writeMu.Lock()
	err := c.Conn.WriteJSON(v)
	c.writeMu.Unlock()
	return err
}

// sendUsernames sends a message containing the names of all active clients
// to the syncChan, to be broadcast to all clients and displayed in their editor.
func (c *Clients) sendUsernames() {
	var users string
	for client := range c.getAll() {
		users += client.Username + ","
	}

	syncChan <- commons.Message{Text: users, Type: commons.UsersMessage}
}
