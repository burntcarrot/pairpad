package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

// generateNewState generates a random character that is appended to the client state. Used for testing purposes.
func generateNewState() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	return string(letters[rand.Intn(len(letters))])
}

func main() {
	// Server address.
	addr := "localhost:8080"

	// Set up interrupt channel.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Construct WebSocket URL.
	u := url.URL{Scheme: "ws", Host: addr, Path: "/echo"}
	log.Printf("connecting to %s", u.String())

	// Connect to WebSocket.
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	// Set up done channel.
	done := make(chan struct{})

	// Read username from the user.
	username := "invalid"
	fmt.Print("username: ")
	fmt.Scanf("%s", &username)

	// Initialize user state.
	state := username + " "

	go func() {
		defer close(done)
		for {
			// Read messages from the connection.
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("updated state: %s", message)
		}
	}()

	// Set up a ticker.
	// Interval is set to 4 seconds.
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			// On ticker event, update state.
			state += generateNewState()
			// Send message to server.
			err := c.WriteMessage(websocket.TextMessage, []byte(state))
			if err != nil {
				log.Println("write:", err)
				return
			}
		case <-interrupt:
			log.Println("interrupt")
			// Cleanly close the connection by sending a close message.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
