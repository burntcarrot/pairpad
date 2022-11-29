package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/burntcarrot/rowix/crdt"
	"github.com/gorilla/websocket"
	"github.com/nsf/termbox-go"
)

type message struct {
	Username  string         `json:"username"`
	Text      string         `json:"text"`
	Type      string         `json:"type"`
	Operation Operation      `json:"operation"`
	Document  *crdt.Document `json:"document"`
}

type Operation struct {
	Type     string `json:"type"`
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
	server := flag.String("server", "localhost:8080", "Server network address")
	path := flag.String("path", "/", "Server path")
	secure := flag.Bool("wss", false, "Use wss by default")
	flag.Parse()

	// Construct WebSocket URL.
	var u url.URL
	if *secure {
		u = url.URL{Scheme: "wss", Host: *server, Path: *path}
	} else {
		u = url.URL{Scheme: "ws", Host: *server, Path: *path}
	}

	// Read username.
	fmt.Print("Enter your name: ")
	s = bufio.NewScanner(os.Stdin)
	s.Scan()
	name = s.Text()

	// Initialize document.
	doc = crdt.New()

	// Get WebSocket connection.
	dialer := websocket.Dialer{
		HandshakeTimeout: 2 * time.Minute,
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Printf("Connection error, exiting: %s\n", err)
		os.Exit(0)
	}
	defer conn.Close()

	// Send joining message.
	msg := message{Username: name, Text: "has joined the session.", Type: "info"}
	_ = conn.WriteJSON(msg)

	// syncMsg := message{Type: "syncReq"}
	// _ = conn.WriteJSON(syncMsg)

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
		if strings.HasPrefix(err.Error(), "rowix") {
			fmt.Println("exiting session.")
			os.Exit(0)
		}
		fmt.Printf("TUI error, exiting: %s\n", err)
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

		case msg := <-msgChan:
			handleMsg(msg, doc, conn)
		}
	}
}

// handleTermboxEvent handles key input by updating the local CRDT document and sending a message over the WebSocket connection.
func handleTermboxEvent(ev termbox.Event, conn *websocket.Conn) error {
	switch ev.Type {
	case termbox.EventKey:
		switch ev.Key {
		case termbox.KeyEsc, termbox.KeyCtrlC:
			return errors.New("rowix: exiting")
		case termbox.KeyArrowLeft, termbox.KeyCtrlB:
			e.MoveCursor(-1, 0)
		case termbox.KeyArrowRight, termbox.KeyCtrlF:
			e.MoveCursor(1, 0)
		case termbox.KeyHome:
			e.SetX(0)
		case termbox.KeyEnd:
			e.SetX(len(e.text))
		case termbox.KeyBackspace, termbox.KeyBackspace2:
			performOperation(OperationDelete, ev, conn)
		case termbox.KeyDelete:
			performOperation(OperationDelete, ev, conn)
		case termbox.KeyTab: // TODO: add tabs?
		case termbox.KeyEnter:
			logger.Println("enter value:", ev.Ch)
			ev.Ch = '\n'
			performOperation(OperationInsert, ev, conn)
		case termbox.KeySpace:
			logger.Println("space value:", ev.Ch)
			ev.Ch = ' '
			performOperation(OperationInsert, ev, conn)
		default:
			if ev.Ch != 0 {
				performOperation(OperationInsert, ev, conn)
			}
		}
	}
	e.Draw()
	return nil
}

const (
	OperationInsert = iota
	OperationDelete
)

func performOperation(opType int, ev termbox.Event, conn *websocket.Conn) {
	// Get position and value.
	ch := string(ev.Ch)

	var msg message

	// Modify local state (CRDT) first.
	switch opType {
	case OperationInsert:
		r := []rune(ch)
		e.AddRune(r[0])

		text, _ := doc.Insert(e.cursor, ch)
		e.SetText(text)
		// logger.Println(crdt.Content(doc))
		msg = message{Type: "operation", Operation: Operation{Type: "insert", Position: e.cursor, Value: ch}}
	case OperationDelete:
		if e.cursor-1 <= 0 {
			e.cursor = 1
		}
		text := doc.Delete(e.cursor)
		e.SetText(text)
		msg = message{Type: "operation", Operation: Operation{Type: "delete", Position: e.cursor}}
		e.MoveCursor(-1, 0)
	}

	_ = conn.WriteJSON(msg)
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
func handleMsg(msg message, doc *crdt.Document, conn *websocket.Conn) {
	if msg.Type == "syncResp" {
		*doc = *msg.Document
		logger.Printf("%+v\n", msg.Document)
	} else if msg.Type == "docReq" {
		docMsg := message{Type: "docResp", Document: doc}
		conn.WriteJSON(&docMsg)
	} else {
		switch msg.Operation.Type {
		case "insert":
			_, _ = doc.Insert(msg.Operation.Position, msg.Operation.Value)
		case "delete":
			_ = doc.Delete(msg.Operation.Position)
		}
	}

	e.SetText(crdt.Content(*doc))
	e.Draw()
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
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("websocket error: %v", err)
				}
				break
			}

			logger.Printf("message received: %+v\n", msg)

			// send message through channel
			messageChan <- msg

		}
	}()
	return messageChan
}
