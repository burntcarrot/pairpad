package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"log"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

// Editor represents the editor's skeleton.
// The editor is composed of two components:
// 1. an editable text area; which acts as the primary interactive area.
// 2. a status bar; which displays messages on different events, for example, when an user joins a session, etc.
type Editor struct {
	// Text contains the editor's content.
	Text []rune

	// Cursor represents the cursor position of the editor.
	Cursor int

	// Width represents the terminal's width in characters.
	Width int

	// Height represents the terminal's width in characters.
	Height int

	//ColOff is the number of columns between the start of a line and the left of the editor window
	ColOff int

	// RowOff is the number of rows between the beginning of the text and the top of the editor window
	RowOff int

	// ShowMsg acts like a switch for the status bar.
	ShowMsg bool

	// StatusMsg represents the text displayed in the status bar.
	StatusMsg string

	Logger *log.Logger
}

var logFile *os.File

// NewEditor returns a new instance of the editor.
func NewEditor() *Editor {
	logName := "scroll.log"
	dirName, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if filepath.Base(dirName) == "editor" {
		logName = "test.log"
	}
	logFile, err := os.OpenFile(logName, os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	logger := log.New(logFile, "--", log.Default().Flags())

	return &Editor{Logger: logger}
}

// GetText returns the editor's content.
func (e *Editor) GetText() []rune {
	return e.Text
}

// SetText sets the given string as the editor's content.
func (e *Editor) SetText(text string) {
	e.Text = []rune(text)
}

// GetX returns the X-axis component of the current cursor position.
func (e *Editor) GetX() int {
	x, _ := e.calcCursorXY(e.Cursor)
	return x
}

// SetX sets the X-axis component of the current cursor position to the specified X position.
func (e *Editor) SetX(x int) {
	e.Cursor = x
}

// GetY returns the Y-axis component of the current cursor position.
func (e *Editor) GetY() int {
	_, y := e.calcCursorXY(e.Cursor)
	return y
}

// GetWidth returns the editor's width (in characters).
func (e *Editor) GetWidth() int {
	return e.Width
}

// GetWidth returns the editor's height (in characters).
func (e *Editor) GetHeight() int {
	return e.Height
}

// SetSize sets the editor size to the specific width and height.
func (e *Editor) SetSize(w, h int) {
	e.Width = w
	e.Height = h
}

func (e *Editor) SetRowOff(y int) {
	e.RowOff = y
}

func (e *Editor) GetRowOff() int {
	return e.RowOff
}

func (e *Editor) GetColOff() int {
	return e.ColOff
}

// AddRune adds a rune to the editor's existing content and updates the cursor position.
func (e *Editor) AddRune(r rune) {
	if e.Cursor == 0 {
		e.Text = append([]rune{r}, e.Text...)
	} else if e.Cursor < len(e.Text) {
		e.Text = append(e.Text[:e.Cursor], e.Text[e.Cursor-1:]...)
		e.Text[e.Cursor] = r
	} else {
		e.Text = append(e.Text[:e.Cursor], r)
	}
	e.MoveCursor(1, 0)
}

// Draw updates the UI by setting cells with the editor's content.
// func (e *Editor) Draw() {
// 	_ = termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
// 	cx, cy := e.calcCursorXY(e.Cursor)
// 	termbox.SetCursor(cx-1, cy-1)

// 	x, y := 0, 0
// 	for i := 0; i < len(e.Text); i++ {
// 		if e.Text[i] == rune('\n') {
// 			x = 0
// 			y++
// 		} else {
// 			if x < e.Width {
// 				// Set cell content.
// 				termbox.SetCell(x, y, e.Text[i], termbox.ColorDefault, termbox.ColorDefault)
// 			}

// 			// Update x by rune's width.
// 			x = x + runewidth.RuneWidth(e.Text[i])
// 		}
// 	}

// 	if e.ShowMsg {
// 		e.SetStatusBar()
// 	} else {
// 		e.showPositions()
// 	}

// 	// Flush back buffer!
// 	termbox.Flush()
// }

// There's a RowOff that defines the start of the viewport.
func (e *Editor) Draw() {
	e.Logger.Print("\n\n DRAWING")

	_ = termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	cx, cy := e.calcCursorXY(e.Cursor)

	// draw cursor x position relative to row offset
	if cx-e.GetColOff() > 0 {
		cx -= e.GetColOff()
	}

	// draw cursor y position relative to row offset
	if cy-e.GetRowOff() > 0 {
		cy -= e.GetRowOff()
	}

	termbox.SetCursor(cx-1, cy-1)
	// find the starting and ending row of the termbox window.
	yStart := e.GetRowOff()
	yEnd := yStart + e.GetHeight() - 1 // -1 to deal with status bar

	// find the starting and ending column of the termbox window.
	xStart := e.GetColOff()

	e.Logger.Print("\n" + string(e.GetText()))
	e.Logger.Print("rowoff: ", e.GetRowOff())

	x, y := 0, 0
	for i := 0; i < len(e.Text) && y < yEnd; i++ {
		e.Logger.Print("i: ", i, " rune: ", string(e.Text[i]))
		if e.Text[i] == rune('\n') {
			x = 0
			y++
		} else {
			if y >= yStart && x >= xStart {
				// Set cell content.
				setY := y - yStart
				setX := x - xStart
				e.Logger.Printf("setting rune %s at x: %d, y: %d", string(e.Text[i]), setX, setY)
				termbox.SetCell(setX, setY, e.Text[i], termbox.ColorDefault, termbox.ColorDefault)
			}

			// Update x by rune's width.
			x = x + runewidth.RuneWidth(e.Text[i])
		}
	}

	if e.ShowMsg {
		e.SetStatusBar()
	} else {
		e.showPositions()
	}

	// Flush back buffer!
	termbox.Flush()
}

// SetStatusBar sets the message (e.StatusMsg) in the editor's status bar.
// The message disappears automatically after 5 seconds, in order to simulate the "popup" effect.
func (e *Editor) SetStatusBar() {
	e.ShowMsg = true

	for i, r := range []rune(e.StatusMsg) {
		termbox.SetCell(i, e.Height-1, r, termbox.ColorDefault, termbox.ColorDefault)
	}

	// Toggle showMsg to false to hide the message.
	_ = time.AfterFunc(5*time.Second, func() {
		e.ShowMsg = false
	})
}

// showPositions shows the cursor positions with other details.
func (e *Editor) showPositions() {
	x, y := e.calcCursorXY(e.Cursor)

	// Construct message for debugging.
	str := fmt.Sprintf("x=%d, y=%d, cursor=%d, len(text)=%d, y offset=%d, height=%d", x, y, e.Cursor, len(e.Text), e.RowOff, e.Height)

	for i, r := range []rune(str) {
		termbox.SetCell(i, e.Height-1, r, termbox.ColorDefault, termbox.ColorDefault)
	}
}

// // scroll moves the editor view.
// func (e *Editor) Scroll(y int) {
// 	// find the y coord of the end of the text
// 	_, textHeight := e.calcCursorXY(len(e.Text) - 1)
// 	switch {
// 	case y > 0: // scroll down
// 		// only scroll down if there's more text to see.
// 		if e.RowOff+e.Height < textHeight+3 {
// 			e.RowOff += y
// 		}

// 		// make sure the cursor doesn't get left behind
// 		_, cy := e.calcCursorXY(e.Cursor)

// 		// if the start of the window frame is after the cursor y position, move it til after
// 		if e.RowOff > cy {
// 			e.Logger.Print("")
// 			e.setCursorY(e.RowOff + 1)
// 		}
// 	case y < 0: // scroll up
// 		e.RowOff += y

// 		// can't scroll negatively
// 		if e.RowOff < 0 {
// 			e.RowOff = 0
// 		}
// 	}

// }

// MoveCursor updates the cursor position horizontally by a given x increment, and
// vertically by one line in the direction indicated by y. Positive y moves the cursor
// down, and negative y moves the cursor up.
// This is used by the UI layer, where it updates the cursor position on keypresses.
func (e *Editor) MoveCursor(x, y int) {
	if len(e.Text) == 0 && e.Cursor == 0 {
		return
	}
	// Move cursor horizontally.
	newCursor := e.Cursor + x

	// Move cursor vertically.
	if y > 0 {
		newCursor = e.calcCursorDown()
	}

	if y < 0 {
		newCursor = e.calcCursorUp()
	}

	cx, cy := e.calcCursorXY(newCursor)

	// if the cursor is too low, move the window down
	if cy > e.Height+e.RowOff-1 {
		e.RowOff++
	}

	// if the cursor is too high, move the window up
	if cy <= e.RowOff {
		e.RowOff--
	}

	// if the cursor is too far right, move the window right
	if cx > e.ColOff+e.Width {
		e.ColOff++
	}

	// if the cursor is too far left, move the window left
	if cx <= e.ColOff {
		e.ColOff--
	}

	// Reset to bounds.
	if newCursor > len(e.Text) {
		newCursor = len(e.Text)
	}

	if newCursor < 0 {
		newCursor = 0
	}

	e.Cursor = newCursor
}

// For the functions calcCursorUp and calcCursorDown, newline characters are found by iterating backward and forward from the current cursor position.
// These characters are taken as the "start" and "end" of the current line.
// The "offset" from the start of the current line to the cursor is calculated and used to determine the final cursor position on the target line, based on whether the offset is greater than the length of the target line.
// "pos" is used as a placeholder variable for the cursor.

// calcCursorUp calculates and returns the intended cursor position after moving the cursor up one line.
func (e *Editor) calcCursorUp() int {
	pos := e.Cursor
	offset := 0

	// If the initial cursor is out of the bounds of the text or already on a newline, move it.
	if pos == len(e.Text) || e.Text[pos] == '\n' {
		offset++
		pos--
	}

	if pos < 0 {
		pos = 0
	}

	start, end := pos, pos

	// Find the start of the current line.
	for start > 0 && e.Text[start] != '\n' {
		start--
	}

	// If the cursor is already on the first line, move to the beginning of the Text.
	if start == 0 {
		return 0
	}

	// Find the end of the current line.
	for end < len(e.Text) && e.Text[end] != '\n' {
		end++
	}

	// Find the start of the previous line.
	prevStart := start - 1
	for prevStart >= 0 && e.Text[prevStart] != '\n' {
		prevStart--
	}

	// Calculate the distance from the start of the current line to the cursor.
	offset += pos - start
	if offset <= start-prevStart {
		return prevStart + offset
	} else {
		return start
	}
}

func (e *Editor) calcCursorDown() int {
	pos := e.Cursor
	offset := 0

	// If the initial cursor position is out of the bounds or already on a newline, move it.
	if pos == len(e.Text) || e.Text[pos] == '\n' {
		offset++
		pos--
	}

	if pos < 0 {
		pos = 0
	}

	start, end := pos, pos

	// Find the start of the current line.
	for start > 0 && e.Text[start] != '\n' {
		start--
	}

	// This handles the case where the cursor is on the first line. This is necessary because the start of the first line is not a newline character, unlike the other lines in the text.
	if start == 0 && e.Text[start] != '\n' {
		offset++
	}

	// Find the end of the current line.
	for end < len(e.Text) && e.Text[end] != '\n' {
		end++
	}

	// This handles the case where the cursor is on a newline. end has to be incremented, otherwise start == end.
	if e.Text[pos] == '\n' && e.Cursor != 0 {
		end++
	}

	// If the Cursor is already on the last line, move to the end of the text.
	if end == len(e.Text) {
		return len(e.Text)
	}

	// Find the end of the next line.
	nextEnd := end + 1
	for nextEnd < len(e.Text) && e.Text[nextEnd] != '\n' {
		nextEnd++
	}

	// Calculate the distance from the start of the current line to the cursor.
	offset += pos - start
	if offset < nextEnd-end {
		return end + offset
	} else {
		return nextEnd
	}
}

// calcCursorXY returns the x and y coordinates of the given cursor index.
func (e *Editor) calcCursorXY(index int) (int, int) {
	x := 1
	y := 1

	if index < 0 {
		return x, y
	}

	if index > len(e.Text) {
		index = len(e.Text)
	}

	for i := 0; i < index; i++ {
		if e.Text[i] == rune('\n') {
			x = 1
			y++
		} else {
			x = x + runewidth.RuneWidth(e.Text[i])
		}
	}
	return x, y
}

// func (e *Editor) setCursorY(y int) {
// 	i := 0
// 	for cy := 0; cy <= y && i < len(e.Text); i++ {
// 		if e.Text[i] == '\n' {
// 			cy++
// 		}
// 	}

// 	e.Cursor = i
// }
