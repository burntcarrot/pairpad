package editor

import (
	"fmt"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

type Editor struct {
	Text      []rune
	Cursor    int
	Width     int
	Height    int
	ShowMsg   bool
	StatusMsg string
}

func NewEditor() *Editor {
	return &Editor{}
}

func (e *Editor) GetText() []rune {
	return e.Text
}

func (e *Editor) SetText(text string) {
	e.Text = []rune(text)
}

func (e *Editor) GetX() int {
	x, _ := e.calcCursorXY(e.Cursor)
	return x
}

func (e *Editor) SetX(x int) {
	e.Cursor = x
}

func (e *Editor) GetY() int {
	_, y := e.calcCursorXY(e.Cursor)
	return y
}

func (e *Editor) GetWidth() int {
	return e.Width
}

func (e *Editor) GetHeight() int {
	return e.Height
}

func (e *Editor) SetSize(w, h int) {
	e.Width = w
	e.Height = h
}

// AddRune adds a rune to the editor's state and updates position.
func (e *Editor) AddRune(r rune) {
	if e.Cursor == 0 {
		e.Text = append([]rune{r}, e.Text...)
	} else if e.Cursor < len(e.Text) {
		e.Text = append(e.Text[:e.Cursor], e.Text[e.Cursor-1:]...)
		e.Text[e.Cursor] = r
	} else {
		e.Text = append(e.Text[:e.Cursor], r)
	}
	e.Cursor++
}

// Draw updates the UI by setting cells with the editor's content.
func (e *Editor) Draw() {
	_ = termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	cx, cy := e.calcCursorXY(e.Cursor)
	termbox.SetCursor(cx-1, cy-1)

	x, y := 0, 0
	for i := 0; i < len(e.Text); i++ {
		if e.Text[i] == rune('\n') {
			x = 0
			y++
		} else {
			if x < e.Width {
				// Set cell content.
				termbox.SetCell(x, y, e.Text[i], termbox.ColorDefault, termbox.ColorDefault)
			}

			// Update x by rune's Width.
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

func (e *Editor) SetStatusBar() {
	e.ShowMsg = true

	for i, r := range []rune(e.StatusMsg) {
		termbox.SetCell(i, e.Height-1, r, termbox.ColorDefault, termbox.ColorDefault)
	}

	_ = time.AfterFunc(5*time.Second, func() {
		e.ShowMsg = false
	})
}

// showPositions shows the positions with other details.
func (e *Editor) showPositions() {
	x, y := e.calcCursorXY(e.Cursor)

	// Construct message for debugging.
	str := fmt.Sprintf("x=%d, y=%d, cursor=%d, len(text)=%d", x, y, e.Cursor, len(e.Text))

	for i, r := range []rune(str) {
		termbox.SetCell(i, e.Height-1, r, termbox.ColorDefault, termbox.ColorDefault)
	}
}

// MoveCursor updates the Cursor position.
func (e *Editor) MoveCursor(x, y int) {
	if len(e.Text) == 0 {
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

	// Reset to bounds.
	if newCursor > len(e.Text) {
		newCursor = len(e.Text)
	}

	if newCursor < 0 {
		newCursor = 0
	}

	e.Cursor = newCursor
}

// For the functions calcCursorUp and calcCursorDown, newline characters are found by iterating
// backward and forward from the current Cursor position. These characters are taken as the "start"
// and "end" of the current line. The "offset" from the start of the current line to the Cursor
// is calculated and used to determine the final Cursor position on the target line, based on whether the
// offset is greater than the length of the target line. "pos" is used as a placeholder variable for
// the Cursor.

// calcCursorUp calculates the intended Cursor position after moving the Cursor up one line.
func (e *Editor) calcCursorUp() int {
	pos := e.Cursor
	offset := 0

	// If the initial cursor is out of the bounds of the Text or already on a newline, move it.
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

	// If the Cursor is already on the first line, move to the beginning of the Text.
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

	// Calculate the distance from the start of the current line to the Cursor.
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

	// If the initial Cursor is out of the bounds of the Text or already on a newline, move it.
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

	// This handles the case where the Cursor is on the first line. This is necessary because the start
	// of the first line is not a newline character, unlike the other lines in the Text.
	if start == 0 && e.Text[start] != '\n' {
		offset++
	}

	// Find the end of the current line.
	for end < len(e.Text) && e.Text[end] != '\n' {
		end++
	}

	// This handles the case where the Cursor is on a newline. end has to be incremented, otherwise
	// start == end.
	if e.Text[pos] == '\n' && e.Cursor != 0 {
		end++
	}

	// If the Cursor is already on the last line, move to the end of the Text.
	if end == len(e.Text) {
		return len(e.Text)
	}

	// Find the end of the next line.
	nextEnd := end + 1
	for nextEnd < len(e.Text) && e.Text[nextEnd] != '\n' {
		nextEnd++
	}

	// Calculate the distance from the start of the current line to the Cursor.
	offset += pos - start
	if offset < nextEnd-end {
		return end + offset
	} else {
		return nextEnd
	}
}

// calcCursorXY calculates Cursor position from the index obtained from the content.
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
