package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

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

	// open logging file  and create if non-existent
	file, err := os.OpenFile("help.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Logger error, exiting: %s", err)
		os.Exit(0)
	}
	defer file.Close()

	logger = log.New(file, "operations:", log.LstdFlags)

	err = UI(conn, &doc)
	if err != nil {
		fmt.Printf("TUI error, exiting: %s", err)
		os.Exit(0)
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
	termboxChan := getTermboxChan()
	msgChan := getMsgChan(conn)

	// event select
	for {
		select {
		case termboxEvent := <-termboxChan:
			err := handleTermboxEvent(termboxEvent, conn)
			if err != nil {
				return err
			}
			e.SetText(crdt.Content(*doc))
			e.Draw()
		case msg := <-msgChan:
			handleMsg(msg, doc)
			e.SetText(crdt.Content(*doc))
			e.Draw()
		}
	}
}

// handleTermboxEvent handles key input by updating the local CRDT document and sending
// a message over the websocket connection.
func handleTermboxEvent(ev termbox.Event, conn *websocket.Conn) error {
	switch ev.Type {
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
	}
	return nil
}

// getTermboxChan returns a channel of termbox Events repeatedly waiting on user input.
func getTermboxChan() chan termbox.Event {
	termboxChan := make(chan termbox.Event)
	go func() {
		for {
			termboxChan <- termbox.PollEvent()
		}
	}()
	return termboxChan
}

// handleMsg updates the CRDT document with the contents of the message.
func handleMsg(msg message, doc *crdt.Document) {
	_, _ = doc.Insert(msg.Operation.Position, msg.Operation.Value)
}

// getMsgChan returns a message channel that repeatedly reads from a websocket connection.
func getMsgChan(conn *websocket.Conn) chan message {
	messageChan := make(chan message)
	go func() {
		for {
			var msg message

			// Read message.
			err := conn.ReadJSON(&msg)
			if err != nil {
				fmt.Printf("Server closed. Exiting...")
				fmt.Printf("connection error!: %s\n", err)
				// TODO: error handling?
				os.Exit(0)
			}

			logger.Printf("message received: %+v\n", msg)

			// send message through channel
			messageChan <- msg
			// if msg.Type == "operation" {
			// 	logger.Printf("operation received: %+v\n", msg)
			// 	text, _ := doc.Insert(msg.Operation.Position, msg.Operation.Value)
			// 	e.SetText(text)
			// }

		}
	}()
	return messageChan
}
