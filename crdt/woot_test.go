package crdt

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDocument(t *testing.T) {
	doc := New()

	// A new document must have at least 2 characters (start and end).
	got := doc.Length()
	want := 2

	if got != want {
		t.Errorf("got != want; got = %v, expected = %v\n", got, want)
	}
}

// TestInsert verifies Insert's functionality.
func TestInsert(t *testing.T) {
	doc := New()

	position := 1
	value := "a"

	// Perform insertion.
	content, err := doc.Insert(position, value)
	if err != nil {
		t.Errorf("error: %v\n", err)
	}

	// Generate document for equality assertion.
	wantDoc := &Document{
		Characters: []Character{
			{ID: "start", Visible: false, Value: "", IDPrevious: "", IDNext: "end"},
			{ID: "1", Visible: true, Value: "a", IDPrevious: "start", IDNext: "end"},
			{ID: "end", Visible: false, Value: "", IDPrevious: "1", IDNext: ""},
		},
	}

	got := content
	want := Content(*wantDoc)

	// Since content is a string, it could be compared directly.
	if got != want {
		t.Errorf("got != want; got = %v, expected = %v\n", got, want)
	}
}

// TestIntegrateInsert_SamePosition checks what happens if a value is inserted at the same position.
func TestIntegrateInsert_SamePosition(t *testing.T) {
	// Generate a test document.
	doc := &Document{
		Characters: []Character{
			{ID: "start", Visible: false, Value: "", IDPrevious: "", IDNext: "1"},
			{ID: "1", Visible: false, Value: "e", IDPrevious: "start", IDNext: "2"},
			{ID: "2", Visible: false, Value: "n", IDPrevious: "1", IDNext: "end"},
			{ID: "end", Visible: false, Value: "", IDPrevious: "2", IDNext: ""},
		},
	}

	// Insert a new character at the start. (IDPrevious = start)
	newChar := Character{ID: "3", Visible: false, Value: "b", IDPrevious: "start", IDNext: "1"}

	charPrev := Character{ID: "start", Visible: false, Value: "", IDPrevious: "", IDNext: "1"}
	charNext := Character{ID: "1", Visible: false, Value: "e", IDPrevious: "start", IDNext: "2"}

	// Perform insertion.
	content, err := doc.IntegrateInsert(newChar, charPrev, charNext)
	if err != nil {
		t.Errorf("error: %v\n", err)
	}

	// This should be the final representation of the document.
	wantDoc := &Document{
		Characters: []Character{
			{ID: "start", Visible: false, Value: "", IDPrevious: "", IDNext: "3"},
			{ID: "3", Visible: false, Value: "b", IDPrevious: "start", IDNext: "1"},
			{ID: "1", Visible: false, Value: "e", IDPrevious: "3", IDNext: "2"},
			{ID: "2", Visible: false, Value: "n", IDPrevious: "1", IDNext: "end"},
			{ID: "end", Visible: false, Value: "", IDPrevious: "2", IDNext: ""},
		},
	}

	got := content
	want := wantDoc

	// Do equality check using go-cmp, and display human-readable diff.
	if !cmp.Equal(got, want) {
		t.Errorf("got != want; diff = %v\n", cmp.Diff(got, want))
	}
}

// TestIntegrateInsert_SamePosition checks what happens if a value is inserted at the same position.
func TestIntegrateInsert_BetweenTwoPositions(t *testing.T) {
	// Generate a test document.
	doc := &Document{
		Characters: []Character{
			{ID: "start", Visible: false, Value: "", IDPrevious: "", IDNext: "1"},
			{ID: "1", Visible: false, Value: "c", IDPrevious: "start", IDNext: "2"},
			{ID: "2", Visible: false, Value: "t", IDPrevious: "1", IDNext: "end"},
			{ID: "end", Visible: false, Value: "", IDPrevious: "2", IDNext: ""},
		},
	}

	// Insert a new character between <"1", "c"> and <"2", "t">.
	newChar := Character{ID: "3", Visible: false, Value: "a", IDPrevious: "1", IDNext: "2"}

	charPrev := Character{ID: "1", Visible: false, Value: "c", IDPrevious: "start", IDNext: "2"}
	charNext := Character{ID: "2", Visible: false, Value: "t", IDPrevious: "1", IDNext: "end"}

	// Perform insertion.
	content, err := doc.IntegrateInsert(newChar, charPrev, charNext)
	if err != nil {
		t.Errorf("error: %v\n", err)
	}

	// This should be the final representation of the document.
	wantDoc := &Document{
		Characters: []Character{
			{ID: "start", Visible: false, Value: "", IDPrevious: "", IDNext: "1"},
			{ID: "1", Visible: false, Value: "c", IDPrevious: "start", IDNext: "3"},
			{ID: "3", Visible: false, Value: "a", IDPrevious: "1", IDNext: "2"},
			{ID: "2", Visible: false, Value: "t", IDPrevious: "3", IDNext: "end"},
			{ID: "end", Visible: false, Value: "", IDPrevious: "2", IDNext: ""},
		},
	}

	got := content
	want := wantDoc

	// Do equality check using go-cmp, and display human-readable diff.
	if !cmp.Equal(got, want) {
		t.Errorf("got != want; diff = %v\n", cmp.Diff(got, want))
	}
}

func TestLoad(t *testing.T) {
	// create test doc
	doc := &Document{
		Characters: []Character{
			{ID: "start", Visible: false, Value: "", IDPrevious: "", IDNext: "1"},
			{ID: "1", Visible: true, Value: "c", IDPrevious: "start", IDNext: "3"},
			{ID: "3", Visible: true, Value: "a", IDPrevious: "1", IDNext: "2"},
			{ID: "2", Visible: true, Value: "t", IDPrevious: "3", IDNext: "4"},
			{ID: "4", Visible: true, Value: "\n", IDPrevious: "2", IDNext: "5"},
			{ID: "5", Visible: true, Value: "d", IDPrevious: "4", IDNext: "6"},
			{ID: "6", Visible: true, Value: "o", IDPrevious: "5", IDNext: "7"},
			{ID: "7", Visible: true, Value: "g", IDPrevious: "6", IDNext: "end"},
			{ID: "end", Visible: false, Value: "", IDPrevious: "7", IDNext: ""},
		},
	}

	tmp, err := os.CreateTemp("", "ex")
	if err != nil {
		t.Errorf("error: %v\n", err)
	}
	defer os.Remove(tmp.Name())

	// Save to a temporary file
	err = Save(tmp.Name(), doc)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}

	// Load from the temporary file
	loadedDoc, err := Load(tmp.Name())
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}
	// compare the contents of the loaded doc and the original doc
	got := Content(loadedDoc)
	want := Content(*doc)

	if !cmp.Equal(got, want) {
		t.Errorf("got != want; diff = %v\n", cmp.Diff(got, want))
	}
}
