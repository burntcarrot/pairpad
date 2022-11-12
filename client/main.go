package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/burntcarrot/rowix/crdt"
	"github.com/gorilla/websocket"
	"github.com/nsf/termbox-go"
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
	Username  string    `json:"username"`
	Text      string    `json:"text"`
	Type      string    `json:"type"`
	Operation Operation `json:"operation"`
}

type Operation struct {
	Position int    `json:"position"`
	Value    string `json:"value"`
}

// Local document containing content.
var doc crdt.Document

// Centralized logger.
var logger *log.Logger

// termbox-based editor.
var e *Editor

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
	fmt.Print("Enter your name: ")
	s = bufio.NewScanner(os.Stdin)
	s.Scan()
	name = s.Text()

	// Initialize document.
	doc = crdt.New()

	// Get WebSocket connection.
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Printf("Connection error, exiting: %s", err)
		os.Exit(0)
	}
	defer conn.Close()

	// Send joining message.
	msg := message{Username: name, Text: "has joined the chat.", Type: "info"}
	_ = conn.WriteJSON(msg)

	// open file and create if non-existent
	file, err := os.OpenFile("help.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Logger error, exiting: %s", err)
		os.Exit(0)
	}
	defer file.Close()

	logger = log.New(file, "operations:", log.LstdFlags)

	go readMessages(conn, &doc) // Handle incoming messages concurrently.
	// writeMessages(conn, s, name) // Handle outgoing messages concurrently.

	err = UI(conn, &doc)
	if err != nil {
		fmt.Printf("TUI error, exiting: %s", err)
		os.Exit(0)
	}
}

// readMessages handles incoming messages on the WebSocket connection.
func readMessages(conn ConnReader, doc *crdt.Document) {
	for {
		var msg message

		// Read message.
		err := conn.ReadJSON(&msg)
		if err != nil {
			fmt.Printf("Server closed. Exiting...")
			// TODO: error handling?
			os.Exit(0)
		}

		logger.Printf("message received: %+v\n", msg)

		if msg.Type == "operation" {
			logger.Printf("operation received: %+v\n", msg)
			text, _ := doc.Insert(msg.Operation.Position, msg.Operation.Value)
			e.SetText(text)
		}
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
				fmt.Printf("%s %s\n", msg.Username, msg.Text)
			} else {
				fmt.Printf("%s: %s\n", msg.Username, msg.Text)
			}

			// Write message to connection.
			err := conn.WriteJSON(msg)
			if err != nil {
				log.Fatal("Error sending message, exiting")
			}
		}
	}
}

// UI creates a new editor view and runs the main loop.
func UI(conn *websocket.Conn, d *crdt.Document) error {
	err := termbox.Init()
	if err != nil {
		return err
	}
	defer termbox.Close()

	e = NewEditor()
	e.SetSize(termbox.Size())
	e.Draw()

	err = mainLoop(e, conn, d)
	if err != nil {
		return err
	}

	return nil
}

// mainLoop is the main update loop for the UI.
func mainLoop(e *Editor, conn *websocket.Conn, doc *crdt.Document) error {

	// Backstory:
	// termbox.PollEvent() is a blocking call and waits for keyboard events, hence updating the local content at a client requires the client to do an "empty" keypress.
	// This empty keypress can be done using the arrow keys, or any other keys.
	// Once the keypress happens, the local state gets updated.

	// To mitigate this problem, the repeatDraw goroutine is spawned here.
	// repeatDraw sets the editor's content to the local CRDT document's state.
	go repeatDraw(e, doc)

	for {
		// Wait for keyboard event
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyEsc, termbox.KeyCtrlC:
				return errors.New("exiting")
			case termbox.KeyArrowLeft, termbox.KeyCtrlB:
				e.MoveCursor(-1, 0)
				e.Draw()
			case termbox.KeyArrowRight, termbox.KeyCtrlF:
				e.MoveCursor(1, 0)
				e.Draw()
			case termbox.KeyBackspace, termbox.KeyBackspace2: // TODO: support deletion
			case termbox.KeyDelete: // TODO: support deletes
			case termbox.KeyTab: // TODO: add tabs?
			case termbox.KeyEnter:
				// Get position and value.
				pos := e.GetX()
				ch := string(ev.Ch)

				// Modify local state (CRDT) first.
				text, _ := doc.Insert(pos, ch)
				e.SetText(text)

				// Send payload to WebSocket connection.
				msg := message{Type: "operation", Operation: Operation{Position: pos, Value: ch}}
				conn.WriteJSON(msg)

				e.Draw()
			default:
				if ev.Ch != 0 {
					// Get position and value.
					pos := e.GetX()
					ch := string(ev.Ch)

					// Modify local state (CRDT) first.
					text, _ := doc.Insert(pos, ch)
					e.SetText(text)

					// Send payload to WebSocket connection.
					msg := message{Type: "operation", Operation: Operation{Position: pos, Value: ch}}
					conn.WriteJSON(msg)

					e.Draw()
				}
			}
		case termbox.EventResize:
			// Change editor size on resize event.
			e.SetSize(termbox.Size())
		}
	}
}

// repeatDraw updates/syncs the editor state with the document state.
func repeatDraw(e *Editor, doc *crdt.Document) {
	for {
		// Current sync interval is 100ms.
		time.Sleep(100 * time.Millisecond)
		e.SetText(crdt.Content(*doc))
		e.Draw()
	}
}
