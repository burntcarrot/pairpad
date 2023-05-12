package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Pallinder/go-randomdata"
	"github.com/burntcarrot/pairpad/client/editor"
	"github.com/burntcarrot/pairpad/commons"
	"github.com/burntcarrot/pairpad/crdt"
	"github.com/sirupsen/logrus"
)

var (
	// Local document containing content.
	doc = crdt.New()

	// Centralized logger.
	logger = logrus.New()

	// termbox-based editor.
	e = editor.NewEditor(editor.EditorConfig{})

	// The name of the file to load from and save to.
	fileName string

	// Parsed flags.
	flags Flags
)

func main() {
	// Parse flags.
	flags = parseFlags()

	s := bufio.NewScanner(os.Stdin)

	// Generate a random username.
	name := randomdata.SillyName()

	// Read username based if login flag is set to true.
	if flags.Login {
		fmt.Print("Enter your name: ")
		s.Scan()
		name = s.Text()
	}

	conn, _, err := createConn(flags)
	if err != nil {
		fmt.Printf("Connection error, exiting: %s\n", err)
		return
	}
	defer conn.Close()

	// Send joining message.
	msg := commons.Message{Username: name, Text: "has joined the session.", Type: commons.JoinMessage}
	_ = conn.WriteJSON(msg)

	logFile, debugLogFile, err := setupLogger(logger)
	if err != nil {
		fmt.Printf("Failed to setup logger, exiting: %s\n", err)
		return
	}
	defer closeLogFiles(logFile, debugLogFile)

	if flags.File != "" {
		if doc, err = crdt.Load(flags.File); err != nil {
			fmt.Printf("failed to load document: %s\n", err)
			return
		}
	}

	uiConfig := UIConfig{
		EditorConfig: editor.EditorConfig{
			ScrollEnabled: flags.Scroll,
		},
	}

	err = initUI(conn, uiConfig)
	if err != nil {
		// If error has the prefix "pairpad", then it was triggered by an event that wasn't an error, for example, exiting the editor.
		// It's a hacky solution since the UI returns an error only.
		if strings.HasPrefix(err.Error(), "pairpad") {
			fmt.Println("exiting session.")
			return
		}

		// This is printed when it's an actual error.
		fmt.Printf("TUI error, exiting: %s\n", err)
		return
	}
}
