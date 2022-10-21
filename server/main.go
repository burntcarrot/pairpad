package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Server address.
var addr = "localhost:8080"

// Temp counter.
var counter = 0

// Connection upgrader.
var upgrader = websocket.Upgrader{}

// Connection list.
var conns []*websocket.Conn

// Echo handler.
func echo(w http.ResponseWriter, r *http.Request) {
	// Upgrade to WebSocket connection.
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	// Add connection to connection list.
	conns = append(conns, c)

	for {
		// Broadcast
		for i, c := range conns {
			mt, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
			}

			// Log for debugging purposes.
			log.Printf("reading from conn %d, got state from: %s", i, message)

			// Increment counter.
			counter += 1

			// Construct a new message to send to the client.
			newMessage := fmt.Sprintf("conns: %d, %d %s", len(conns), counter, string(message))

			// Send message to client.
			err = c.WriteMessage(mt, []byte(newMessage))
			if err != nil {
				log.Println("write:", err)
				break
			}
		}
	}
}

func main() {
	// Register echo handler.
	http.HandleFunc("/echo", echo)

	// Start server.
	log.Printf("starting server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
