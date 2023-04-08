package main

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/burntcarrot/pairpad/crdt"
	"github.com/gorilla/websocket"
	"github.com/nsf/termbox-go"
	"github.com/sirupsen/logrus"
)

// handleTermboxEvent handles key input by updating the local CRDT document and sending a message over the WebSocket connection.
func handleTermboxEvent(ev termbox.Event, conn *websocket.Conn) error {
	if ev.Type == termbox.EventKey {
		switch ev.Key {
		case termbox.KeyEsc, termbox.KeyCtrlC:
			return errors.New("pairpad: exiting")
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
		logger.Infof("LOCAL DELETE: cursor position %v\n", e.Cursor)

		if e.Cursor-1 < 0 {
			e.Cursor = 0
		}

		text := doc.Delete(e.Cursor)
		e.SetText(text)

		msg = message{Type: "operation", Operation: Operation{Type: "delete", Position: e.Cursor}}
		e.MoveCursor(-1, 0)
	}

	err := conn.WriteJSON(msg)
	if err != nil {
		e.StatusMsg = "lost connection!"
		e.SetStatusBar()
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
	switch msg.Type {
	case "docSync":
		logger.Infof("DOCSYNC RECEIVED, updating local doc %+v\n", msg.Document)

		doc = msg.Document

	case "docReq":
		logger.Infof("DOCREQ RECEIVED, sending local document to %v\n", msg.ID)

		docMsg := message{Type: "docSync", Document: doc, ID: msg.ID}
		_ = conn.WriteJSON(&docMsg)

	case "SiteID":
		siteID, err := strconv.Atoi(msg.Text)
		if err != nil {
			logger.Errorf("failed to set siteID, err: %v\n", err)
		}

		crdt.SiteID = siteID
		logger.Infof("SITE ID %v, INTENDED SITE ID: %v", crdt.SiteID, siteID)

	case "join":
		e.StatusMsg = fmt.Sprintf("%s has joined the session!", msg.Username)
		e.SetStatusBar()

	default:
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
