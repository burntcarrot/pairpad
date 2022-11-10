package crdt

import (
	"errors"
	"fmt"
)

// Document is composed of characters.
type Document struct {
	Characters []Character
}

// Character represents a character in the document.
// As per section 3.1, Data Model in the paper (https://hal.inria.fr/inria-00108523/document)
type Character struct {
	ID         string
	Visible    bool
	Value      string
	IDPrevious string
	IDNext     string
}

var (
	// SiteID is a globally unique variable used with the local clock to generate identifiers for characters in the document.
	SiteID = 0

	// LocalClock is incremented whenever an insert operation takes place. It is used to uniquely identify each character.
	LocalClock = 0

	// CharacterStart is placed at the start.
	CharacterStart = Character{ID: "start", Visible: false, Value: "", IDPrevious: "", IDNext: "end"}

	// CharacterEnd is placed at the end.
	CharacterEnd = Character{ID: "end", Visible: false, Value: "", IDPrevious: "start", IDNext: ""}

	ErrPositionOutOfBounds = errors.New("position out of bounds")
	ErrEmptyWCharacter     = errors.New("empty char ID provided")
	ErrBoundsNotPresent    = errors.New("subsequence bound(s) not present")
)

// New returns a initialized document.
func New() Document {
	return Document{Characters: []Character{CharacterStart, CharacterEnd}}
}

//////////////////////
// Utility functions
//////////////////////

// Content returns the content of the document.
func Content(doc Document) string {
	value := ""
	for _, char := range doc.Characters {
		if char.Visible {
			value += char.Value
		}
	}
	return value
}

// IthVisible returns the ith visible character in the document.
func IthVisible(doc Document, position int) Character {
	count := 0

	for _, char := range doc.Characters {
		if char.Visible {
			if count == position-1 {
				return char
			}
			count++
		}
	}

	return Character{ID: "-1"}
}

// Length returns the length of the document.
func (doc *Document) Length() int {
	return len(doc.Characters)
}

// ElementAt returns the character present in the position.
func (doc *Document) ElementAt(position int) (Character, error) {
	if position < 0 || position >= doc.Length() {
		return Character{}, ErrPositionOutOfBounds
	}

	return doc.Characters[position], nil
}

// Position returns the position of the character.
func (doc *Document) Position(charID string) int {
	for position, char := range doc.Characters {
		if charID == char.ID {
			return position + 1
		}
	}

	return -1
}

func (doc *Document) Left(charID string) string {
	i := doc.Position(charID)
	if i <= 0 {
		return doc.Characters[i].ID
	}
	return doc.Characters[i-1].ID
}

func (doc *Document) Right(charID string) string {
	i := doc.Position(charID)
	if i >= len(doc.Characters)-1 {
		return doc.Characters[i-1].ID
	}
	return doc.Characters[i+1].ID
}

// Contains checks if a character is present in the document.
func (doc *Document) Contains(charID string) bool {
	position := doc.Position(charID)
	return position != -1
}

// Find returns the character at the ID.
func (doc *Document) Find(id string) Character {
	for _, char := range doc.Characters {
		if char.ID == id {
			return char
		}
	}

	return Character{ID: "-1"}
}

// Subseq returns the content between the positions.
func (doc *Document) Subseq(wcharacterStart, wcharacterEnd Character) ([]Character, error) {
	startPosition := doc.Position(wcharacterStart.ID)
	endPosition := doc.Position(wcharacterEnd.ID)

	if startPosition == -1 || endPosition == -1 {
		return doc.Characters, ErrBoundsNotPresent
	}

	if startPosition == endPosition {
		return []Character{}, nil
	}

	return doc.Characters[startPosition : endPosition-1], nil
}

///////////////
// Operations
///////////////

// LocalInsert inserts the character into the document.
func (doc *Document) LocalInsert(char Character, position int) (*Document, error) {
	if position <= 0 || position >= doc.Length() {
		return doc, ErrPositionOutOfBounds
	}

	if char.ID == "" {
		return doc, ErrEmptyWCharacter
	}

	doc.Characters = append(doc.Characters[:position],
		append([]Character{char}, doc.Characters[position:]...)...,
	)

	// Update next and previous pointers.
	doc.Characters[position-1].IDNext = char.ID
	doc.Characters[position+1].IDPrevious = char.ID

	return doc, nil
}

// IntegrateInsert inserts the given Character into the Document
// Characters based off of the previous & next Character
func (doc *Document) IntegrateInsert(char, charPrev, charNext Character) (*Document, error) {
	// Get the subsequence.
	subsequence, _ := doc.Subseq(charPrev, charNext)

	// Get the position of the next character.
	position := doc.Position(charNext.ID)
	position--

	// If no characters are present in the subseqence, insert at current position.
	if len(subsequence) == 0 {
		return doc.LocalInsert(char, position)
	}

	// If one character is present in the subseqence, insert at previous position.
	if len(subsequence) == 1 {
		return doc.LocalInsert(char, position-1)
	}

	// Make a recursive call.
	i := 1
	for i < len(subsequence)-1 && subsequence[i].ID < char.ID {
		i++
	}
	return doc.IntegrateInsert(char, subsequence[i-1], subsequence[i])
}

// GenerateInsert generates a character for a given value.
func (doc *Document) GenerateInsert(position int, value string) (*Document, error) {
	// Increment local clock.
	LocalClock++

	// Get previous and next characters.
	charPrev := IthVisible(*doc, position-1)
	charNext := IthVisible(*doc, position)

	// Use defaults.
	if charPrev.ID == "-1" {
		charPrev = doc.Find("start")
	}
	if charNext.ID == "-1" {
		charNext = doc.Find("end")
	}

	char := Character{
		ID:         fmt.Sprint(SiteID) + fmt.Sprint(LocalClock),
		Visible:    true,
		Value:      value,
		IDPrevious: charPrev.ID,
		IDNext:     charNext.ID,
	}

	return doc.IntegrateInsert(char, charPrev, charNext)
}

// IntegrateDelete finds a character and marks it for deletion.
func (doc *Document) IntegrateDelete(char Character) *Document {
	position := doc.Position(char.ID)
	if position == -1 {
		return doc
	}

	// This is how deletion is done.
	doc.Characters[position-1].Visible = false

	return doc
}

// GenerateDelete generates the character which is to be marked for deletion.
func (doc *Document) GenerateDelete(position int) *Document {
	char := IthVisible(*doc, position)
	return doc.IntegrateDelete(char)
}

////////////////////////////////
// Implement the CRDT interface
////////////////////////////////

func (doc *Document) Insert(position int, value string) (string, error) {
	newDoc, err := doc.GenerateInsert(position, value)
	if err != nil {
		return Content(*doc), err
	}

	return Content(*newDoc), nil
}

func (doc *Document) Delete(position int) string {
	newDoc := doc.GenerateDelete(position)
	return Content(*newDoc)
}
