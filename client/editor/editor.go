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
		cls, cle, nle := -1, -1, -1
		offset := 1 // current offset of cursor from beginning of line

		//TODO: check if the below is necessary for avoiding bugs
		if newCursor > len(e.Text)-1 {
			newCursor = len(e.Text) - 1
		}

		// if cursor is currently on newline, 'move' it
		// if e.Text[newCursor] == '\n' {
		// 	newCursor--
		// }

		// find offset from start of line and set cls to start of line
		for i := newCursor; i > 0; i-- {
			if e.Text[i] == '\n' {
				cls = i
				break
			}
		}
		// if start of current line isn't set, assume current line is the first line of the document,
		// and manually set the offset
		if cls < 0 {
			offset = e.Cursor + 1
		} else {
			offset = e.Cursor - cls
		}
		logger.Printf("offset: %v", offset)
		// cle is used to find length of current line (cle - cls)
		for i := cls + 1; i < len(e.Text); i++ {
			if e.Text[i] == '\n' {
				cle = i
				break
			}
		}

		// nle is used to find length of next line (nle - cle)
		if cle > 0 { // if the end of the current line isn't set, no need to find nle
			for i := cle + 1; i < len(e.Text); i++ {
				if e.Text[i] == '\n' {
					nle = i
					break
				}
			}
		}
		// if end of next line isn't set, assume next line is last of the document
		if nle < 0 {
			nle = len(e.Text)
		}
		logger.Printf("Current line starts at %v. Current line ends at %v. Next line ends at %v\n", cls, cle, nle)
		if cle < 0 {
			newCursor = len(e.Text)
		} else if nle-cle < offset { // if next line is shorter than the offset
			newCursor = nle
		} else {
			newCursor = cle + offset
		}
	}

	// move cursor up y cells
	if y < 0 {
		logger.Printf("UP ARROW PRESSED")
		pls, cls, cle := -1, -1, -1 // prev line start, curr line start, curr line end
		offset := 1                 // current offset of cursor from beginning of line

		// below might be necessary
		if newCursor > len(e.Text)-1 {
			logger.Printf("cursor out of bounds! cursor reset at %v", len(e.Text))
			newCursor = len(e.Text) - 1
		}

		// if cursor is currently on newline, 'move' it
		if e.Text[newCursor] == '\n' {
			logger.Printf("cursor on new line, moving to prev char")
			newCursor--
		}

		//  set cls to start of line
		for i := newCursor; i > 0; i-- {
			if e.Text[i] == '\n' {
				logger.Println("current line start found at ", i)
				cls = i
				break
			}
		}
		// if start of current line isn't set, assume current line is the first line of the document,
		// and manually set the offset
		if cls < 0 {
			logger.Printf("on first line")
			offset = e.Cursor + 1
		} else {
			offset = e.Cursor - cls
		}

		logger.Printf("offset: %v", offset)

		// cle is used to find length of current line (cle - cls)
		for i := cls + 1; i < len(e.Text); i++ {
			if e.Text[i] == '\n' {
				logger.Println("current line end at ", i)
				cle = i
				break
			}
		}

		// pls is used to find length of previous line (cle - pls)
		if cls > 0 { // only need to find pls if cls is set
			for i := cls - 1; i > 0; i-- {
				if e.Text[i] == '\n' {
					logger.Println("next line start at ", i)
					pls = i
					break
				}
			}
		}
		// if start of previous line isn't set, assume previous line is start of document
		if pls < 0 {
			pls = 0
			offset--
		}
		logger.Printf("Current line starts at %v. Current line ends at %v. Previous line starts at %v\n", cls, cle, pls)

		if cls < 0 {
			newCursor = 0
		} else if cls-pls < offset { // if next line is shorter than the offset
			logger.Printf("previous line is shorter, moving to end of previous line")
			newCursor = cls
		} else {
			newCursor = pls + offset
		}
	}

	if newCursor > len(e.Text) {
		newCursor = len(e.Text)
	}
	if newCursor < 0 {
		newCursor = 0
	}
	logger.Println("cursor moved to ", newCursor)
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
