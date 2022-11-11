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
	"github.com/mattn/go-runewidth"
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
var logger *log.Logger

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
	// doc.Insert(1, "a")

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

	// err = tui.UI(conn, &doc)

	go readMessages(conn, &doc) // Handle incoming messages concurrently.
	// writeMessages(conn, s, name) // Handle outgoing messages concurrently.

	err = UI(conn, &doc)
	if err != nil {
		fmt.Printf("TUI error, exiting: %s", err)
		//	fmt.Printf("\ndocument: %+v\n\n", doc)
		os.Exit(0)
	}
}

// readMessages handles incoming messages on the WebSocket connection.
func readMessages(conn ConnReader, doc *crdt.Document) {
	// logger.Println("read message goroutine is called!")
	fmt.Println("read message goroutine is called!")
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

		// Display message.
		// fmt.Printf("%s: %s\n", msg.Username, msg.Text)
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

type Editor struct {
	text   []rune
	x      int
	y      int
	width  int
	height int
}

func NewEditor() *Editor {
	return &Editor{
		x: 1,
		y: 1,
	}
}

func (e *Editor) GetText() []rune {
	return e.text
}

func (e *Editor) SetText(text string) {
	e.text = []rune(text)
}

func (e *Editor) GetX() int {
	return e.x
}

func (e *Editor) GetY() int {
	return e.y
}

func (e *Editor) GetWidth() int {
	return e.width
}

func (e *Editor) GetHeight() int {
	return e.height
}

func (e *Editor) SetSize(w, h int) {
	e.width = w
	e.height = h
}

func (e *Editor) AddRune(r rune) {
	cursor := e.calcCursor()
	if cursor == 0 {
		e.text = append([]rune{r}, e.text...)
	} else if cursor < len(e.text) {
		e.text = append(e.text[:cursor], e.text[cursor-1:]...)
		e.text[cursor] = r
	} else {
		e.text = append(e.text[:cursor], r)
	}
	if r == rune('\n') {
		e.x = 1
		e.y += 1
	} else {
		e.x += runewidth.RuneWidth(r)
	}
}

func (e *Editor) DeletePrevRune() {
}

func (e *Editor) Draw() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	termbox.SetCursor(e.x-1, e.y-1)
	x := 0
	y := 0
	for i := 0; i < len(e.text); i++ {
		if e.text[i] == rune('\n') {
			x = 0
			y++
		} else {
			if x < e.width {
				termbox.SetCell(x, y, e.text[i], termbox.ColorDefault, termbox.ColorDefault)
			}
			x = x + runewidth.RuneWidth(e.text[i])
		}
	}
	e.debugDraw()
	termbox.Flush()
}

func (e *Editor) debugDraw() {
	x, y := e.calcCursorXY(e.calcCursor())
	str := fmt.Sprintf("x=%d, y=%d, cursor=%d, len(text)=%d, x,y=%d,%d", e.x, e.y, e.calcCursor(), len(e.text), x, y)
	for i, r := range []rune(str) {
		termbox.SetCell(i, e.height-1, r, termbox.ColorDefault, termbox.ColorDefault)
	}
}

func (e *Editor) MoveCursor(x, y int) {
	c := e.calcCursor()

	if x > 0 {
		if c+x <= len(e.text) {
			e.x, e.y = e.calcCursorXY(c + x)
		}
	} else {
		if 0 <= c+x {
			if e.text[c+x] == rune('\n') {
				e.x, e.y = e.calcCursorXY(c + x - 1)
			} else {
				e.x, e.y = e.calcCursorXY(c + x)
			}
		}
	}
}

// CalcCursor calc index of []rune from e.x and e.y.
func (e *Editor) calcCursor() int {
	ri := 0
	y := 1
	x := 0

	for y < e.y {
		for _, r := range e.text {
			ri++
			if r == '\n' {
				y++
				break
			}
		}
	}

	for _, r := range e.text[ri:] {
		if x >= e.x-runewidth.RuneWidth(r) {
			break
		}
		x += runewidth.RuneWidth(r)
		ri++
	}

	return ri
}

// calcCursorXY calc x and y from index of []rume
func (e *Editor) calcCursorXY(index int) (int, int) {
	x := 1
	y := 1
	for i := 0; i < index; i++ {
		if e.text[i] == rune('\n') {
			x = 1
			y++
		} else {
			x = x + runewidth.RuneWidth(e.text[i])
		}
	}
	return x, y
}

var e *Editor

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

func mainLoop(e *Editor, conn *websocket.Conn, doc *crdt.Document) error {
	go repeatDraw(e, doc)
	for {
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
			case termbox.KeyBackspace, termbox.KeyBackspace2:
			case termbox.KeyDelete, termbox.KeyCtrlD:
			case termbox.KeyTab:
			case termbox.KeySpace:
			case termbox.KeyCtrlK:
			case termbox.KeyHome, termbox.KeyCtrlA:
			case termbox.KeyEnd, termbox.KeyCtrlE:
			case termbox.KeyEnter:
				e.AddRune(rune('\n'))

				pos := e.GetX()
				ch := string(ev.Ch)
				text, _ := doc.Insert(pos, ch)
				e.SetText(text)
				msg := message{Type: "operation", Operation: Operation{Position: pos, Value: ch}}
				conn.WriteJSON(msg)

				e.Draw()
			default:
				if ev.Ch != 0 {
					// e.AddRune(ev.Ch)

					pos := e.GetX()
					ch := string(ev.Ch)
					// _, _ = doc.Insert(pos, ch)
					text, _ := doc.Insert(pos, ch)
					e.SetText(text)
					msg := message{Type: "operation", Operation: Operation{Position: pos, Value: ch}}
					conn.WriteJSON(msg)

					e.Draw()
				}
			}
		case termbox.EventResize:
			e.SetSize(termbox.Size())
		}
	}
}

func repeatDraw(e *Editor, doc *crdt.Document) {
	for {
		time.Sleep(100 * time.Millisecond)
		e.SetText(crdt.Content(*doc))
		e.Draw()
	}

}
