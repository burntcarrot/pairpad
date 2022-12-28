package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/burntcarrot/rowix/client/editor"
	"github.com/burntcarrot/rowix/crdt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/nsf/termbox-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/writer"
)

type message struct {
	Username  string        `json:"username"`
	Text      string        `json:"text"`
	Type      string        `json:"type"`
	ID        uuid.UUID     `json:"ID"`
	Operation Operation     `json:"operation"`
	Document  crdt.Document `json:"document"`
}

type Operation struct {
	Type     string `json:"type"`
	Position int    `json:"position"`
	Value    string `json:"value"`
}

// Local document containing content.
var doc crdt.Document

// Centralized logger.
var logger *logrus.Logger

// WebSocket connection.
var conn *websocket.Conn

// termbox-based editor.
var e *editor.Editor

// The name of the file to load from and save to.
var fileName string

func main() {
	// Parse flags.
	server := flag.String("server", "localhost:8080", "Server network address")
	path := flag.String("path", "/", "Server path")
	secure := flag.Bool("wss", false, "Enable a secure WebSocket connection")
	login := flag.Bool("login", false, "Enable the login prompt")
	file := flag.Bool("file", false, "Choose a file to load")
	flag.Parse()

	// Construct WebSocket URL.
	var u url.URL
	if *secure {
		u = url.URL{Scheme: "wss", Host: *server, Path: *path}
	} else {
		u = url.URL{Scheme: "ws", Host: *server, Path: *path}
	}

	var name string
	var s *bufio.Scanner

	// Read username based if login flag is set to true, otherwise generate a random name.
	if *login {
		fmt.Print("Enter your name: ")
		s = bufio.NewScanner(os.Stdin)
		s.Scan()
		name = s.Text()
	} else {
		name = randomdata.SillyName()
	}

	// Get WebSocket connection.
	dialer := websocket.Dialer{
		HandshakeTimeout: 2 * time.Minute,
	}

	var err error

	conn, _, err = dialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Printf("Connection error, exiting: %s\n", err)
		os.Exit(0)
	}
	defer conn.Close()

	// Send joining message.
	msg := message{Username: name, Text: "has joined the session.", Type: "join"}
	_ = conn.WriteJSON(msg)

	// define log file paths, based on the home directory.
	var logPath, debugLogPath string

	// Get the home directory.
	homeDirExists := true
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDirExists = false
	}

	// Get log paths based on the home directory.
	rowixDir := filepath.Join(homeDir, ".rowix")
	if rowixDirExists(rowixDir) && homeDirExists {
		logPath = filepath.Join(rowixDir, "rowix.log")
		debugLogPath = filepath.Join(rowixDir, "rowix-debug.log")
	} else {
		logPath = "rowix.log"
		debugLogPath = "rowix-debug.log"
	}

	// open the log file and create if it does not exist
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) // skipcq: GSC-G302
	if err != nil {
		fmt.Printf("Logger error, exiting: %s", err)
		return
	}
	defer func() {
		err := logFile.Close()
		if err != nil {
			log.Fatalln(err)
		}
	}()

	// create a separate log file for verbose logs
	debugLogFile, err := os.OpenFile(debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) // skipcq: GSC-G302
	if err != nil {
		fmt.Printf("Logger error, exiting: %s", err)
		return
	}
	defer func() {
		err := debugLogFile.Close()
		if err != nil {
			log.Fatalln(err)
		}
	}()

	logger = logrus.New()
	logger.SetOutput(io.Discard)
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.AddHook(&writer.Hook{
		Writer: logFile,
		LogLevels: []logrus.Level{
			logrus.WarnLevel,
			logrus.ErrorLevel,
			logrus.FatalLevel,
			logrus.PanicLevel,
		},
	})
	logger.AddHook(&writer.Hook{
		Writer: debugLogFile,
		LogLevels: []logrus.Level{
			logrus.TraceLevel,
			logrus.DebugLevel,
			logrus.InfoLevel,
		},
	})

	// Initialize document.
	doc = crdt.New()
	// load the file
	if *file {
		fmt.Print("Enter the name of a file to load and save to: ")
		s = bufio.NewScanner(os.Stdin)
		s.Scan()
		fileName = s.Text()
		if doc, err = crdt.Load(fileName); err != nil {
			fmt.Printf("failed to load document: %s\n", err)
		}
	}

	err = UI(conn)
	if err != nil {
		if strings.HasPrefix(err.Error(), "rowix") {
			fmt.Println("exiting session.")
			return
		}
		fmt.Printf("TUI error, exiting: %s\n", err)
		return
	}

	if err := logFile.Close(); err != nil {
		fmt.Printf("Failed to close log file: %s", err)
		return
	}

	if err := debugLogFile.Close(); err != nil {
		fmt.Printf("Failed to close debug log file: %s", err)
		return
	}

	if err := conn.Close(); err != nil {
		fmt.Printf("Failed to close websocket connection: %s", err)
		return
	}
}

// UI creates a new editor view and runs the main loop.
func UI(conn *websocket.Conn) error {
	err := termbox.Init()
	if err != nil {
		return err
	}
	defer termbox.Close()

	e = editor.NewEditor()
	e.SetSize(termbox.Size())
	e.Draw()

	err = mainLoop(conn)
	if err != nil {
		return err
	}

	return nil
}

// mainLoop is the main update loop for the UI.
func mainLoop(conn *websocket.Conn) error {
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
			handleMsg(msg, conn)
		}
	}
}

// handleTermboxEvent handles key input by updating the local CRDT document and sending a message over the WebSocket connection.
func handleTermboxEvent(ev termbox.Event, conn *websocket.Conn) error {
	if ev.Type == termbox.EventKey {
		switch ev.Key {
		case termbox.KeyEsc, termbox.KeyCtrlC:
			return errors.New("rowix: exiting")
		case termbox.KeyCtrlS:
			if fileName != "" {
				err := crdt.Save(fileName, &doc)
				if err != nil {
					e.StatusMsg = "Failed to save to " + fileName
					logrus.Errorf("failed to save to file %s", fileName)
					e.SetStatusBar()
					return err
				}
				e.StatusMsg = "Saved document to " + fileName
				e.SetStatusBar()
			} else {
				e.StatusMsg = "No file to save to!"
				e.SetStatusBar()
			}
		case termbox.KeyCtrlL:
			if fileName != "" {
				logger.Log(logrus.InfoLevel, "LOADING DOCUMENT")
				newDoc, err := crdt.Load(fileName)
				e.StatusMsg = "Loading " + fileName
				e.SetStatusBar()
				if err != nil {
					e.StatusMsg = "Failed to load " + fileName
					logrus.Errorf("failed to load file %s", fileName)
					e.SetStatusBar()
					return err
				}
				doc = newDoc
				e.SetX(0)
				e.SetText(crdt.Content(doc))

				logger.Log(logrus.InfoLevel, "SENDING DOCUMENT")
				docMsg := message{Type: "docSync", Document: doc}
				_ = conn.WriteJSON(&docMsg)
			} else {
				e.StatusMsg = "No file to load!"
				e.SetStatusBar()
			}
		case termbox.KeyArrowLeft, termbox.KeyCtrlB:
			e.MoveCursor(-1, 0)
		case termbox.KeyArrowRight, termbox.KeyCtrlF:
			e.MoveCursor(1, 0)
		case termbox.KeyArrowUp, termbox.KeyCtrlP:
			e.MoveCursor(0, -1)
		case termbox.KeyArrowDown, termbox.KeyCtrlN:
			e.MoveCursor(0, 1)
		case termbox.KeyHome:
			e.SetX(0)
		case termbox.KeyEnd:
			e.SetX(len(e.Text))
		case termbox.KeyBackspace, termbox.KeyBackspace2:
			performOperation(OperationDelete, ev, conn)
		case termbox.KeyDelete:
			performOperation(OperationDelete, ev, conn)
		case termbox.KeyTab:
			for i := 0; i < 4; i++ {
				ev.Ch = ' '
				performOperation(OperationInsert, ev, conn)
			}
		case termbox.KeyEnter:
			ev.Ch = '\n'
			performOperation(OperationInsert, ev, conn)
		case termbox.KeySpace:
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

// performOperation performs a CRDT insert or delete operation on the local document and sends a message over the WebSocket connection
func performOperation(opType int, ev termbox.Event, conn *websocket.Conn) {
	// Get position and value.
	ch := string(ev.Ch)

	var msg message

	// Modify local state (CRDT) first.
	switch opType {
	case OperationInsert:
		logger.Infof("LOCAL INSERT: %s at cursor position %v\n", ch, e.Cursor)
		r := []rune(ch)
		e.AddRune(r[0])
		text, err := doc.Insert(e.Cursor, ch)
		if err != nil {
			e.SetText(text)
			logger.Errorf("CRDT error: %v\n", err)
		}
		e.SetText(text)
		msg = message{Type: "operation", Operation: Operation{Type: "insert", Position: e.Cursor, Value: ch}}
	case OperationDelete:
		logger.Infof("LOCAL DELETE:  cursor position %v\n", e.Cursor)
		if e.Cursor-1 <= 0 {
			e.Cursor = 1
		}
		text := doc.Delete(e.Cursor)
		e.SetText(text)
		msg = message{Type: "operation", Operation: Operation{Type: "delete", Position: e.Cursor}}
		e.MoveCursor(-1, 0)
	}

	// Print document state to logs.
	printDoc(doc)

	err := conn.WriteJSON(msg)
	if err != nil {
		e.StatusMsg = "lost connection!"
		e.SetStatusBar()
	}
}

// printDoc "prints" the document state to the logs.
func printDoc(doc crdt.Document) {
	logger.Infof("---DOCUMENT STATE---")
	for i, c := range doc.Characters {
		logger.Infof("index: %v  value: %s  ID: %v  IDPrev: %v  IDNext: %v  ", i, c.Value, c.ID, c.IDPrevious, c.IDNext)
	}
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
func handleMsg(msg message, conn *websocket.Conn) {
	if msg.Type == "docSync" { // update local document
		logger.Infof("DOCSYNC RECEIVED, updating local doc%+v\n", msg.Document)
		doc = msg.Document
	} else if msg.Type == "docReq" { // send local document as docResp message
		logger.Infof("DOCREQ RECEIVED, sending local document to %v\n", msg.ID)
		docMsg := message{Type: "docSync", Document: doc, ID: msg.ID}
		_ = conn.WriteJSON(&docMsg)
	} else if msg.Type == "SiteID" {
		siteID, err := strconv.Atoi(msg.Text)
		if err != nil {
			logger.Errorf("failed to set siteID, err: %v\n", err)
		}
		crdt.SiteID = siteID
		logger.Infof("SITE ID %v, INTENDED SITE ID: %v", crdt.SiteID, siteID)
	} else if msg.Type == "join" {
		e.StatusMsg = fmt.Sprintf("%s has joined the session!", msg.Username)
		e.SetStatusBar()
	} else {
		switch msg.Operation.Type {
		case "insert":
			_, err := doc.Insert(msg.Operation.Position, msg.Operation.Value)
			if err != nil {
				logger.Errorf("failed to insert, err: %v\n", err)
			}
			logger.Infof("REMOTE INSERT: %s at position %v\n", msg.Operation.Value, msg.Operation.Position)
		case "delete":
			_ = doc.Delete(msg.Operation.Position)
			logger.Infof("REMOTE DELETE: position %v\n", msg.Operation.Position)
		}
	}
	printDoc(doc)
	e.SetText(crdt.Content(doc))
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
					logger.Errorf("websocket error: %v", err)
				}
				break
			}

			logger.Infof("message received: %+v\n", msg)

			// send message through channel
			messageChan <- msg

		}
	}()
	return messageChan
}

func rowixDirExists(rowixDir string) bool {
	if _, err := os.Stat(rowixDir); err == nil {
		return true
	} else if errors.Is(err, os.ErrNotExist) {
		err = os.Mkdir(rowixDir, 0744) // skipcq: GSC-G302
		if err != nil {
			return false
		} else {
			// skipcq: GSC-G302
			if err = os.Chmod(rowixDir, 0744); err != nil {
				return false
			} else {
				return true
			}
		}
	} else {
		return false
	}
}
