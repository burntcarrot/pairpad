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
	// move cursor horizontally
	newCursor := e.Cursor + x

	// move cursor vertically
	if y > 0 {
		newCursor = e.calcCursorDown()
	}

	if y < 0 {
		newCursor = e.calcCursorUp()
	}

	if newCursor > len(e.Text) {
		newCursor = len(e.Text)
	}

	if newCursor < 0 {
		newCursor = 0
	}

	e.Cursor = newCursor
}

func (e *Editor) calcCursorUp() int {
	pos := e.Cursor
	// reset cursor if out of bounds
	if pos > len(e.Text)-1 {
		pos = len(e.Text) - 1
	}

	// if cursor is currently on newline, "move" it
	if e.Text[pos] == '\n' {
		pos--
	}

	cls := -1
	// find the start of the line the cursor is currently on
	for i := pos; i > 0; i-- {
		if e.Text[i] == '\n' {
			cls = i
			break
		}
	}
	var offset int
	// set the cursor offset from the start of the current line
	if cls < 0 {
		offset = e.Cursor + 1
	} else {
		offset = e.Cursor - cls
	}

	pls := -1
	// find the start of the previous line
	if cls > 0 { // no need to find previous line start if current line start doesn't exist
		for i := cls - 1; i > 0; i-- {
			if e.Text[i] == '\n' {
				pls = i
				break
			}
		}
	}
	// if start of previous line isn't found, assume previous line is first of the document and set cursor to end
	if pls < 0 {
		pls = 0
		offset--
	}

	if cls < 0 {
		return 0
	} else if cls-pls < offset { // if previous line is shorter than the offset, set cursor to start of current line
		return cls
	} else { // default
		return pls + offset
	}
}

func (e *Editor) calcCursorDown() int {
	pos := e.Cursor
	// reset cursor if out of bounds
	if pos > len(e.Text)-1 {
		pos = len(e.Text) - 1
	}

	// if cursor is currently on newline, "move" it
	if e.Text[pos] == '\n' {
		pos--
	}

	cls := -1
	// find the start of the line the cursor is currently on
	for i := pos; i > 0; i-- {
		if e.Text[i] == '\n' {
			cls = i
			break
		}
	}
	var offset int
	// set the cursor offset from the start of the current line
	if cls < 0 {
		offset = e.Cursor + 1
	} else {
		offset = e.Cursor - cls
	}

	cle, nle := -1, -1
	// find the end of the current line
	for i := cls + 1; i < len(e.Text); i++ {
		if e.Text[i] == '\n' {
			cle = i
			break
		}
	}
	// find the end of the next line
	if cle > 0 { // no need to find next line end if the end of the current line doesn't exist
		for i := cle + 1; i < len(e.Text); i++ {
			if e.Text[i] == '\n' {
				nle = i
				break
			}
		}
	}
	// if end of next line isn't found, assume next line is last of the document and set cursor to end
	if nle < 0 {
		nle = len(e.Text)
	}

	if cle < 0 {
		return len(e.Text)
	} else if nle-cle < offset { // if next line is shorter than the offset, set cursor to end of next line
		return nle
	} else { // default
		return cle + offset
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
