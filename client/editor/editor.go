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

// For the functions calcCursorUp and calcCursorDown, variables like cls (current line start),
// pls (previous line start), and nle (next line end) hold the index of the '\n' characters that separate
// each line. These variables are used to calculate the length of the line to move to. If these variables
// remain at -1, it's assumed that the start or end of the document was found instead of a '\n' character.
// The offset variable is used to calculate the number of characters between the start of the current line
// and the cursor, which should be kept constant as you move between lines.

// calcCursorUp calculates the new position of the cursor after moving one line up.
// func (e *Editor) calcCursorUp() int {
// 	file, err := os.OpenFile("cursor.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
// 	if err != nil {
// 		fmt.Printf("Logger error, exiting: %s", err)
// 	}
// 	defer func() {
// 		err := file.Close()
// 		if err != nil {
// 			log.Fatalln(err)
// 		}
// 	}()
// 	logger := log.New(file, "---", log.LstdFlags)

// 	pos := e.Cursor
// 	logger.Printf("MOVING UP: initial position: %v\n", pos)
// 	// reset cursor if out of bounds
// 	if pos >= len(e.Text) {
// 		pos = len(e.Text) - 1
// 		// if e.Text[pos] == '\n' && e.Text[pos-1] == '\n' { // this covers the case where the previous line is blank
// 		// 	logger.Printf("prev char is new line, setting position to %v\n", pos)
// 		// 	return pos
// 		// }
// 	}

// 	// if cursor is currently on newline, "move" it
// 	if e.Text[pos] == '\n' {
// 		logger.Printf("On newline: moving one space back\n")
// 		pos--
// 		if pos < 1 {
// 			logger.Printf("set position to 0")
// 			return 0
// 		}
// 		// if e.Text[pos] == '\n' && e.Text[pos-1] == '\n' { // this covers the case where the previous line is blank
// 		// 	logger.Printf("prev char is new line, setting position to %v\n", pos)
// 		// 	return pos
// 		// }
// 	}

// 	cls := -1
// 	// find the start of the line the cursor is currently on
// 	for i := pos; i >= 0; i-- {
// 		if e.Text[i] == '\n' {
// 			cls = i
// 			break
// 		}
// 	}
// 	// if pos == len(e.Text) {
// 	// 	logger.Printf("at end of text, setting cursor to %v\n", cls)
// 	// 	return cls
// 	// }
// 	logger.Printf("cls is set to %v\n", cls)
// 	var offset int
// 	// set the cursor offset from the start of the current line
// 	if cls < 0 {
// 		offset = e.Cursor + 1
// 	} else {
// 		offset = e.Cursor - cls
// 	}
// 	logger.Printf("offset is set to %v\n", offset)
// 	pls := -1
// 	// find the start of the previous line
// 	if cls > 0 { // no need to find previous line start if current line start doesn't exist
// 		for i := cls - 1; i >= 0; i-- {
// 			if e.Text[i] == '\n' {
// 				pls = i
// 				break
// 			}
// 		}
// 	}
// 	// if start of previous line isn't found, assume previous line is first of the document and set cursor to end
// 	if pls < 0 {
// 		pls = 0
// 		offset--
// 	}
// 	logger.Printf("pls is set to %v\n", pls)

// 	if cls < 0 {
// 		return 0
// 	} else if cls-pls < offset { // if previous line is shorter than the offset, set cursor to start of current line
// 		logger.Printf("pos is cls (%v)\n", cls)
// 		return cls
// 	} else if pos == len(e.Text)-1 {
// 		return cls + offset
// 	} else { // default
// 		logger.Printf("pos is pls (%v) + offset (%v)\n", pls, offset)
// 		return pls + offset
// 	}
// }

// chatGPT
func (e *Editor) calcCursorUp() int {
	file, err := os.OpenFile("cursor.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Printf("Logger error, exiting: %s", err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Fatalln(err)
		}
	}()
	logger := log.New(file, "---", log.LstdFlags)

	logger.Println("MOVING UP")
	logger.Printf("Cursor at %v", e.Cursor)

	pos := e.Cursor
	offset := 0

	if pos == len(e.Text) || e.Text[pos] == '\n' {
		logger.Println("cursor out of bounds or on newline, moving back one")
		offset++
		pos--
	}

	start := pos
	end := pos

	// Find the end of the current line
	for end < len(e.Text) && e.Text[end] != '\n' {
		end++
	}

	// Find the start of the current line
	for start > 0 && e.Text[start] != '\n' {
		start--
	}
	// start++

	logger.Printf("Found start of current line at %v and end of current line at %v\n", start, end)

	// Check if the cursor is at the first line
	if start == 0 {
		return 0
	}

	// Find the start of the previous line
	prevStart := start - 1
	for prevStart >= 0 && e.Text[prevStart] != '\n' {
		prevStart--
	}

	logger.Printf("Found start of previous line at %v \n", prevStart)

	// Calculate the cursor position in the previous line
	offset += pos - start
	logger.Printf("offset set at %v", offset)
	logger.Printf("length of line is %v", end-start)
	if offset <= start-prevStart {
		logger.Printf("offset is less than the length of the previous line, placing cursor at %v", prevStart+offset+1)
		return prevStart + offset
	} else {
		logger.Printf("offset is greater than the length of the previous line, placing cursor at %v", prevStart+1)
		return start
	}
}

// func (e *Editor) calcCursorUp() int {
// 	pos := e.Cursor

// 	if e.Text[pos] == '\n' {
// 		pos--
// 	}

// 	cls, cle := -1, -1
// 	for i := pos; i > 0; i-- {
// 		if e.Text[i] == '\n' {

// 			cls = i

// 		}

// 	}

// 	pls := -1
// 	for i := cls - 1; i > 0; i-- {
// 		if e.Text[i] == '\n' {
// 			pls = i
// 		}
// 	}
// 	offset := 0
// 	if cls >= 0 {
// 		offset = e.Cursor - cls
// 	}

// 	// if cls < 0 {
// 	// 	return 0
// 	// } else if cls-pls < offset { // if previous line is shorter than the offset, set cursor to start of current line
// 	// 	return cls
// 	// } else { // default
// 	// 	return pls + offset
// 	// }
// }

// calcCursorDown calculates the new position of the cursor after moving one line down.
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
