package editor

import (
	"fmt"
	"sync"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

type EditorConfig struct {
	ScrollEnabled bool
}

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

	// ColOff is the number of columns between the start of a line and the left of the editor window
	ColOff int

	// RowOff is the number of rows between the beginning of the text and the top of the editor window
	RowOff int

	// ShowMsg acts like a switch for the status bar.
	ShowMsg bool

	// StatusMsg holds the text to be displayed in the status bar.
	StatusMsg string

	// StatusChan is used to send and receive status messages.
	StatusChan chan string

	// StatusMu protects against concurrent reads and writes to status bar info.
	StatusMu sync.Mutex

	// Users holds the names of all users connected to the server, displayed in the status bar.
	Users []string

	// ScrollEnabled determines whether or not the user can scroll past the initial editor
	// window. It is set by the EditorConfig.
	ScrollEnabled bool

	// IsConnected shows whether the editor is currently connected to the server.
	IsConnected bool

	// DrawChan is used to send and receive signals to update the terminal display.
	DrawChan chan int

	// mu prevents concurrent reads and writes to the editor state.
	mu sync.RWMutex
}

var userColors = []termbox.Attribute{
	termbox.ColorGreen,
	termbox.ColorYellow,
	termbox.ColorBlue,
	termbox.ColorMagenta,
	termbox.ColorCyan,
	termbox.ColorLightYellow,
	termbox.ColorLightMagenta,
	termbox.ColorLightGreen,
	termbox.ColorLightRed,
	termbox.ColorRed,
}

// NewEditor returns a new instance of the editor.
func NewEditor(conf EditorConfig) *Editor {
	return &Editor{
		ScrollEnabled: conf.ScrollEnabled,
		StatusChan:    make(chan string, 100),
		DrawChan:      make(chan int, 10000),
	}
}

// GetText returns the editor's content.
func (e *Editor) GetText() []rune {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.Text
}

// SetText sets the given string as the editor's content.
func (e *Editor) SetText(text string) {
	e.mu.Lock()
	e.Text = []rune(text)
	e.mu.Unlock()
}

// GetX returns the X-axis component of the current cursor position.
func (e *Editor) GetX() int {
	x, _ := e.calcXY(e.Cursor)
	return x
}

// SetX sets the X-axis component of the current cursor position to the specified X position.
func (e *Editor) SetX(x int) {
	e.Cursor = x
}

// GetY returns the Y-axis component of the current cursor position.
func (e *Editor) GetY() int {
	_, y := e.calcXY(e.Cursor)
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

// GetRowOff returns the vertical offset of the editor window from the start of the text.
func (e *Editor) GetRowOff() int {
	return e.RowOff
}

// GetColOff returns the horizontal offset of the editor window from the start of a line.
func (e *Editor) GetColOff() int {
	return e.ColOff
}

// IncRowOff increments the vertical offset of the editor window from the start of the
// text by inc.
func (e *Editor) IncRowOff(inc int) {
	e.RowOff += inc
}

// IncColOff increments the horizontal offset of the editor window from the start of a
// line by inc.
func (e *Editor) IncColOff(inc int) {
	e.ColOff += inc
}

// SendDraw sends a draw signal to the drawLoop. Use this function to
// ensure concurrency safety for rendering the editor.
func (e *Editor) SendDraw() {
	e.DrawChan <- 1
}

// Draw updates the UI by setting cells with the editor's content.
func (e *Editor) Draw() {
	_ = termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	e.mu.RLock()
	cursor := e.Cursor
	e.mu.RUnlock()

	cx, cy := e.calcXY(cursor)

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
	yEnd := yStart + e.GetHeight() - 1 // -1 accounts for the status bar

	// find the starting ending column of the termbox window.
	xStart := e.GetColOff()

	x, y := 0, 0
	for i := 0; i < len(e.Text) && y < yEnd; i++ {
		if e.Text[i] == rune('\n') {
			x = 0
			y++
		} else {
			// Set cell content. setX and setY account for the window offset.
			setY := y - yStart
			setX := x - xStart
			termbox.SetCell(setX, setY, e.Text[i], termbox.ColorDefault, termbox.ColorDefault)

			// Update x by rune's width.
			x = x + runewidth.RuneWidth(e.Text[i])
		}
	}

	e.DrawStatusBar()

	// Flush back buffer!
	termbox.Flush()
}

// DrawStatusBar shows all status and debug information on the bottom line of the editor.
func (e *Editor) DrawStatusBar() {
	e.StatusMu.Lock()
	showMsg := e.ShowMsg
	e.StatusMu.Unlock()
	if showMsg {
		e.DrawStatusMsg()
	} else {
		e.DrawInfoBar()
	}

	// Render connection indicator
	if e.IsConnected {
		termbox.SetBg(e.Width-1, e.Height-1, termbox.ColorGreen)
	} else {
		termbox.SetBg(e.Width-1, e.Height-1, termbox.ColorRed)
	}
}

// DrawStatusMsg draws the editor's status message at the bottom of the
// termbox window.
func (e *Editor) DrawStatusMsg() {
	e.StatusMu.Lock()
	statusMsg := e.StatusMsg
	e.StatusMu.Unlock()
	for i, r := range []rune(statusMsg) {
		termbox.SetCell(i, e.Height-1, r, termbox.ColorDefault, termbox.ColorDefault)
	}
}

// DrawInfoBar draws the editor's debug information and the names of the
// active users in the editing session at the bottom of the termbox window.
func (e *Editor) DrawInfoBar() {
	e.StatusMu.Lock()
	users := e.Users
	e.StatusMu.Unlock()

	e.mu.RLock()
	length := len(e.Text)
	e.mu.RUnlock()

	x := 0
	for i, user := range users {
		for _, r := range user {
			colorIdx := i % len(userColors)
			termbox.SetCell(x, e.Height-1, r, userColors[colorIdx], termbox.ColorDefault)
			x++
		}
		termbox.SetCell(x, e.Height-1, ' ', termbox.ColorDefault, termbox.ColorDefault)
		x++
	}

	e.mu.RLock()
	cursor := e.Cursor
	e.mu.RUnlock()

	cx, cy := e.calcXY(cursor)
	debugInfo := fmt.Sprintf(" x=%d, y=%d, cursor=%d, len(text)=%d", cx, cy, e.Cursor, length)

	for _, r := range debugInfo {
		termbox.SetCell(x, e.Height-1, r, termbox.ColorDefault, termbox.ColorDefault)
		x++
	}
}

// MoveCursor updates the cursor position horizontally by a given x increment, and
// vertically by one line in the direction indicated by y. The positive directions are
// right and down, respectively.
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

	if e.ScrollEnabled {
		cx, cy := e.calcXY(newCursor)

		// move the window to adjust for the cursor
		rowStart := e.GetRowOff()
		rowEnd := e.GetRowOff() + e.GetHeight() - 1

		if cy <= rowStart { // scroll up
			e.IncRowOff(cy - rowStart - 1)
		}

		if cy > rowEnd { // scroll down
			e.IncRowOff(cy - rowEnd)
		}

		colStart := e.GetColOff()
		colEnd := e.GetColOff() + e.GetWidth()

		if cx <= colStart { // scroll left
			e.IncColOff(cx - (colStart + 1))
		}

		if cx > colEnd { // scroll right
			e.IncColOff(cx - colEnd)
		}
	}

	// Reset to bounds.
	if newCursor > len(e.Text) {
		newCursor = len(e.Text)
	}

	if newCursor < 0 {
		newCursor = 0
	}
	e.mu.Lock()
	e.Cursor = newCursor
	e.mu.Unlock()
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

// calcCursorDown calculates and returns the intended cursor position after moving the cursor down one line.
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

// calcXY returns the x and y coordinates of the cell at the given
// index in the text.
func (e *Editor) calcXY(index int) (int, int) {
	x := 1
	y := 1

	if index < 0 {
		return x, y
	}

	e.mu.RLock()
	length := len(e.Text)
	e.mu.RUnlock()

	if index > length {
		index = length
	}

	for i := 0; i < index; i++ {
		e.mu.RLock()
		r := e.Text[i]
		e.mu.RUnlock()
		if r == rune('\n') {
			x = 1
			y++
		} else {
			x = x + runewidth.RuneWidth(r)
		}
	}
	return x, y
}
