package main

import (
	"fmt"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

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

func (e *Editor) SetX(x int) {
	e.x = x
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

// AddRune adds a rune to the editor's state and updates position.
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

// Draw updates the UI by setting cells with the editor's content.
func (e *Editor) Draw() {
	_ = termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	termbox.SetCursor(e.x-1, e.y-1)
	x := 0
	y := 0

	for i := 0; i < len(e.text); i++ {
		if e.text[i] == rune('\n') {
			x = 0
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
	x, y := e.calcCursorXY(e.calcCursor())

	// Construct message for debugging.
	str := fmt.Sprintf("x=%d, y=%d, cursor=%d, len(text)=%d, x,y=%d,%d", e.x, e.y, e.calcCursor(), len(e.text), x, y)

	for i, r := range []rune(str) {
		termbox.SetCell(i, e.height-1, r, termbox.ColorDefault, termbox.ColorDefault)
	}
}

// MoveCursor updates the cursor position.
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

// CalcCursor calculates the cursor position.
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

// calcCursorXY calculates cursor position from the index obtained from the content.
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
