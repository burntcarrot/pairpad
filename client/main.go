package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/burntcarrot/rowix/crdt"
	"github.com/fatih/color"
	"github.com/gorilla/websocket"
)

type ConnReader interface {
	ReadJSON(v interface{}) error
}

type ConnWriter interface {
	WriteJSON(v interface{}) error
	Close() error
}

type Scanner interface {
	Scan() bool
	Text() string
}

type message struct {
	Username string `json:"username"`
	Text     string `json:"text"`
	Type     string `json:"type"`
}

func main() {
	var name string
	var s *bufio.Scanner

	// Parse flags.
	server := flag.String("server", "localhost:9000", "Server network address")
	path := flag.String("path", "/", "Server path")
	flag.Parse()

	// Construct WebSocket URL.
	u := url.URL{Scheme: "ws", Host: *server, Path: *path}

	// Read username.
	fmt.Printf("%s", color.YellowString("Enter your Name: "))
	s = bufio.NewScanner(os.Stdin)
	s.Scan()
	name = s.Text()

	var doc = crdt.New()
	crdt.IsCRDT(&doc)
	fmt.Println(doc.Length())

	// Display welcome message.
	color.Green("\nWelcome %s!\n", name)
	color.Green("Connecting to server @ %s\n", *server)
	color.Yellow("Send a message, or type !q to exit.\n")

	// Get WebSocket connection.
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		color.Red("Connection error, exiting: %s", err)
		os.Exit(0)
	}
	defer conn.Close()

	// Send joining message.
	msg := message{Username: name, Text: "has joined the chat.", Type: "info"}
	_ = conn.WriteJSON(msg)

	go readMessages(conn)        // Handle incoming messages concurrently.
	writeMessages(conn, s, name) // Handle outgoing messages concurrently.
}

// readMessages handles incoming messages on the WebSocket connection.
func readMessages(conn ConnReader) {
	for {
		var msg message

		// Read message.
		err := conn.ReadJSON(&msg)
		if err != nil {
			color.Red("Server closed. Exiting...")
			os.Exit(0)
		}

		// Display message.
		color.Magenta("%s: %s\n", msg.Username, msg.Text)
	}
}

// writeMessages scans stdin and sends each scanned line to the server as JSON.
func writeMessages(conn ConnWriter, s Scanner, name string) {
	var msg message
	msg.Username = name

	for {
		fmt.Print("> ")
		if s.Scan() {
			fmt.Printf("\033[A")
			msg.Text = s.Text()

			// Handle quit event.
			if msg.Text == "!q" {
				fmt.Println("Goodbye!")
				_ = conn.WriteJSON(message{Username: name, Text: "has disconnected.", Type: "info"})
				conn.Close()
				os.Exit(0)
			}

			// Display message.
			if msg.Type != "" {
				color.Cyan("%s %s\n", msg.Username, msg.Text)
			} else {
				color.Cyan("%s: %s\n", msg.Username, msg.Text)
			}

			// Write message to connection.
			err := conn.WriteJSON(msg)
			if err != nil {
				log.Fatal("Error sending message, exiting")
			}
		}
	}
}
