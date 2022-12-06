package main

import (
	"fmt"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

type Editor struct {
	text   []rune
	cursor int
	width  int
	height int
}

func NewEditor() *Editor {
	return &Editor{}
}

func (e *Editor) GetText() []rune {
	return e.text
}

func (e *Editor) SetText(text string) {
	e.text = []rune(text)
}

func (e *Editor) GetX() int {
	x, _ := e.calcCursorXY(e.cursor)
	return x
}

func (e *Editor) SetX(x int) {
	e.cursor = x
}

func (e *Editor) GetY() int {
	_, y := e.calcCursorXY(e.cursor)
	return y
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

// AddRune adds a rune to the editor's state and updates position.
func (e *Editor) AddRune(r rune) {
	if e.cursor == 0 {
		e.text = append([]rune{r}, e.text...)
	} else if e.cursor < len(e.text) {
		e.text = append(e.text[:e.cursor], e.text[e.cursor-1:]...)
		e.text[e.cursor] = r
	} else {
		e.text = append(e.text[:e.cursor], r)
	}
	e.cursor++
}

// Draw updates the UI by setting cells with the editor's content.
func (e *Editor) Draw() {
	_ = termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	cx, cy := e.calcCursorXY(e.cursor)
	termbox.SetCursor(cx-1, cy-1)

	x, y := 1, 1
	for i := 0; i < len(e.text); i++ {
		if e.text[i] == rune('\n') {
			x = 1
			y++
		} else {
			if x < e.width {
				// Set cell content.
				termbox.SetCell(x, y, e.text[i], termbox.ColorDefault, termbox.ColorDefault)
			}

			// Update x by rune's width.
			x = x + runewidth.RuneWidth(e.text[i])
		}
	}

	// Show position details (for debugging).
	e.showPositions()

	// Flush back buffer!
	termbox.Flush()
}

// showPositions shows the positions with other details.
func (e *Editor) showPositions() {
	x, y := e.calcCursorXY(e.cursor)

	// Construct message for debugging.
	str := fmt.Sprintf("x=%d, y=%d, cursor=%d, len(text)=%d", x, y, e.cursor, len(e.text))

	for i, r := range []rune(str) {
		termbox.SetCell(i, e.height-1, r, termbox.ColorDefault, termbox.ColorDefault)
	}
}

// MoveCursor updates the cursor position.
func (e *Editor) MoveCursor(x, _ int) {
	newCursor := e.cursor + x

	if newCursor < 0 {
		newCursor = 0
	}
	if newCursor > len(e.text) {
		newCursor = len(e.text)
	}
	e.cursor = newCursor
}

// calcCursorXY calculates cursor position from the index obtained from the content.
func (e *Editor) calcCursorXY(index int) (int, int) {
	x := 1
	y := 1

	if index < 0 {
		return x, y
	}

	if index > len(e.text) {
		index = len(e.text)
	}

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
