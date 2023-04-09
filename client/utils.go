package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/burntcarrot/pairpad/crdt"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/writer"
)

// Flags represents the command-line flags that are passed to pairpad's client.
type Flags struct {
	Server string
	Secure bool
	Login  bool
	File   string
	Debug  bool
}

// parseFlags parses command-line flags.
func parseFlags() Flags {
	serverAddr := flag.String("server", "localhost:8080", "The network address of the server")
	useSecureConn := flag.Bool("secure", false, "Enable a secure WebSocket connection (wss://)")
	enableDebug := flag.Bool("debug", false, "Enable debugging mode to show more verbose logs")
	enableLogin := flag.Bool("login", false, "Enable the login prompt for the server")
	file := flag.String("file", "", "The file to load the pairpad content from")

	flag.Parse()

	return Flags{
		Server: *serverAddr,
		Secure: *useSecureConn,
		Debug:  *enableDebug,
		Login:  *enableLogin,
		File:   *file,
	}
}

// createConn creates a WebSocket connection.
func createConn(flags Flags) (*websocket.Conn, *http.Response, error) {
	var u url.URL
	if flags.Secure {
		u = url.URL{Scheme: "wss", Host: flags.Server, Path: "/"}
	} else {
		u = url.URL{Scheme: "ws", Host: flags.Server, Path: "/"}
	}

	// Get WebSocket connection.
	dialer := websocket.Dialer{
		HandshakeTimeout: 2 * time.Minute,
	}

	return dialer.Dial(u.String(), nil)
}

// ensureDirExists ensures that a directory exists, and if it isn't present, it tries to create a new one.
func ensureDirExists(path string) (bool, error) {
	// Check if the directory exists
	if _, err := os.Stat(path); err == nil {
		return true, nil
	}

	// Create the directory
	err := os.Mkdir(path, 0700)
	if err != nil {
		return false, err
	}

	return true, nil
}

// setupLogger initializes the client's logger (logrus).
func setupLogger(logger *logrus.Logger) (*os.File, *os.File, error) {
	// define log file paths, based on the home directory.
	logPath := "pairpad.log"
	debugLogPath := "pairpad-debug.log"

	// Get the home directory.
	homeDirExists := true
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDirExists = false
	}

	pairpadDir := filepath.Join(homeDir, ".pairpad")

	dirExists, err := ensureDirExists(pairpadDir)
	if err != nil {
		return nil, nil, err
	}

	// Get log paths based on the home directory.
	if dirExists && homeDirExists {
		logPath = filepath.Join(pairpadDir, "pairpad.log")
		debugLogPath = filepath.Join(pairpadDir, "pairpad-debug.log")
	}

	// Open the log file and create if it does not exist.
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) // skipcq: GSC-G302
	if err != nil {
		fmt.Printf("Logger error, exiting: %s", err)
		return nil, nil, err
	}

	// Create a separate log file for verbose logs.
	debugLogFile, err := os.OpenFile(debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) // skipcq: GSC-G302
	if err != nil {
		fmt.Printf("Logger error, exiting: %s", err)
		return nil, nil, err
	}

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

	return logFile, debugLogFile, nil
}

// closeLogFiles closes the log files created by the client.
// closeLogFiles is meant to be used for defer calls.
func closeLogFiles(logFile, debugLogFile *os.File) {
	if err := logFile.Close(); err != nil {
		fmt.Printf("Failed to close log file: %s", err)
		return
	}

	if err := debugLogFile.Close(); err != nil {
		fmt.Printf("Failed to close debug log file: %s", err)
		return
	}
}

// printDoc "prints" the document state to the logs.
func printDoc(doc crdt.Document) {
	if flags.Debug {
		logger.Infof("---DOCUMENT STATE---")
		for i, c := range doc.Characters {
			logger.Infof("index: %v  value: %s  ID: %v  IDPrev: %v  IDNext: %v  ", i, c.Value, c.ID, c.IDPrevious, c.IDNext)
		}
	}
}
