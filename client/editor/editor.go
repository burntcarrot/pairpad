package editor

import (
	"fmt"
	"log"
	"os"
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
	file, err := os.OpenFile("cursor.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Printf("Logger error, exiting: %s", err)
		return
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Fatalln(err)
		}
	}()
	logger := log.New(file, "---", log.LstdFlags)

	newCursor := e.Cursor + x

	// move cursor down y cells
	if y > 0 {
		logger.Printf("DOWN ARROW PRESSED")
		n1 := -1 // index of next newline character
		n2 := -1 // index of the newline character after n1

		// store the position of the next two newline characters
		for j := e.Cursor; j < len(e.Text); j++ {
			if e.Text[j] == '\n' {
				if n1 > 0 { // if the next newline character has already been found
					n2 = j
					break
				}
				n1 = j
			}
		}
		if n1 < 0 && n2 < 0 {
			logger.Printf("couldn't find next newline, setting e.Cursor to end of text\n")
			e.Cursor = len(e.Text)
			return
		}
		if n2 < 0 {
			n2 = len(e.Text)
		}

		newCursor = n2
		logger.Printf("the next two new line characters are at %v and %v\n.", n1, n2)
		for k := 0; k < n1-e.Cursor; k++ {
			if newCursor == n1 {
				break
			}
			newCursor--
		}
		logger.Printf("After calc, newCursor is at: %v", newCursor)
	}

	// move cursor up y cells
	if y < 0 {
		logger.Printf("UP ARROW PRESSED")
		n1 := -1 // index of previous newline character
		n2 := -1 // index of the newline character before n1
		// store the position of the previous two newline characters
		for j := e.Cursor; j > 0; j-- {
			if e.Text[j] == '\n' {
				if n1 > 0 { // if the previous newline character has already been found
					n2 = j
					logger.Printf("this code is getting reached! n2 = %v\n", n2)
					break
				}
				n1 = j
			}
		}
		if n1 < 0 && n2 < 0 {
			logger.Printf("couldn't find previous newline, setting e.Cursor to 0\n")
			e.Cursor = 0
			return
		}
		if n2 < 0 {
			n2 = 0
		}
		newCursor = n2
		logger.Printf("the previous two new line characters are at %v and %v\n.", n1, n2)
		for k := 0; k < e.Cursor-n1; k++ {
			if newCursor == n1 {
				break
			}
			newCursor++
		}
		logger.Printf("After calc, newCursor is at: %v", newCursor)
	}

	if newCursor < 0 {
		newCursor = 0
	}
	if newCursor > len(e.Text) {
		newCursor = len(e.Text)
	}

	e.Cursor = newCursor

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

// func (e *Editor) calcCursor(x, y int) int {
// 	ri := 0
// 	yi := 1
// 	xi := 1

// 	for yi < y {
// 		for _, r := range e.Text {
// 			ri++
// 			if r == '\n' {
// 				yi++
// 				break
// 			}
// 		}
// 		if ri > len(e.Text) {
// 			ri = len(e.Text)
// 		}

// 		for _, r := range e.Text[ri:] {
// 			if xi >= x-runewidth.RuneWidth(r) {
// 				break
// 			}
// 			xi += runewidth.RuneWidth(r)
// 			ri++
// 		}
// 	}
// 	return ri
// }
