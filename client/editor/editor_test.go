package editor

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAddRune(t *testing.T) {
	tests := []struct {
		r        rune
		cursor   int
		expected []rune
	}{
		{r: 'a', cursor: 0, expected: []rune{'a'}},
		{r: 'b', cursor: 1, expected: []rune{'a', 'b'}},
		{r: 'c', cursor: 2, expected: []rune{'a', 'b', 'c'}},
		{r: 'e', cursor: 3, expected: []rune{'a', 'b', 'c', 'e'}},
		{r: 'd', cursor: 3, expected: []rune{'a', 'b', 'c', 'd', 'e'}},
	}

	e := NewEditor()

	for _, tc := range tests {
		e.Cursor = tc.cursor
		e.AddRune(tc.r)
		if !cmp.Equal(e.GetText(), tc.expected) {
			t.Errorf("got != expected, diff: %v\n", cmp.Diff(e.Text, tc.expected))
		}
	}
}

func TestCalcCursorXY(t *testing.T) {
	tests := []struct {
		description string
		cursor      int
		expectedX   int
		expectedY   int
	}{
		{description: "initial position", cursor: 0, expectedX: 1, expectedY: 1},
		{description: "negative index", cursor: -1, expectedX: 1, expectedY: 1},
		{description: "normal editing", cursor: 6, expectedX: 7, expectedY: 1},
		{description: "after newline", cursor: 10, expectedX: 3, expectedY: 2},
		{description: "large number", cursor: 100000, expectedX: 5, expectedY: 2},
	}

	e := NewEditor()
	e.Text = []rune("content\ntest")

	for _, tc := range tests {
		e.Cursor = tc.cursor
		x, y := e.calcCursorXY(e.Cursor)

		got := []int{x, y}
		expected := []int{tc.expectedX, tc.expectedY}

		if !cmp.Equal(got, expected) {
			t.Errorf("(%s) got != expected, diff: %v\n", tc.description, cmp.Diff(got, expected))
		}
	}
}

func TestMoveCursor(t *testing.T) {
	tests := []struct {
		description    string
		cursor         int
		x              int
		y              int
		expectedCursor int
		text           []rune
	}{
		// test horizontal movement
		{description: "move forward (empty document)", cursor: 0, x: 1, expectedCursor: 0,
			text: []rune("")},
		{description: "move backward (empty document)", cursor: 0, x: -1, expectedCursor: 0,
			text: []rune("")},
		{description: "move forward", cursor: 0, x: 1, expectedCursor: 1,
			text: []rune("foo\n")},
		{description: "move backward", cursor: 1, x: -1, expectedCursor: 0,
			text: []rune("foo\n")},
		{description: "move backward (out of bounds)", cursor: 0, x: -10, expectedCursor: 0,
			text: []rune("foo\n")},
		{description: "move forward (out of bounds)", cursor: 3, x: 2, expectedCursor: 4,
			text: []rune("foo\n")},
		// test vertical movement
		{description: "move up", cursor: 6, y: -1, expectedCursor: 2,
			text: []rune("foo\nbar")},
		{description: "move down", cursor: 1, y: 2, expectedCursor: 5,
			text: []rune("foo\nbar")},
		{description: "move up (empty document)", cursor: 0, y: -1, expectedCursor: 0,
			text: []rune("")},
		{description: "move down (empty document)", cursor: 0, y: 1, expectedCursor: 0,
			text: []rune("")},
		{description: "move up (first line)", cursor: 1, y: -1, expectedCursor: 0,
			text: []rune("foo\nbar")},
		{description: "move down (last line)", cursor: 4, y: 1, expectedCursor: 7,
			text: []rune("foo\nbar")},
		{description: "move up (middle line)", cursor: 5, y: -1, expectedCursor: 1,
			text: []rune("foo\nbar\nbaz")},
		{description: "move down (middle line)", cursor: 5, y: 1, expectedCursor: 9,
			text: []rune("foo\nbar\nbaz")},
		{description: "move up (on newline)", cursor: 3, y: -1, expectedCursor: 0,
			text: []rune("foo\nbar\nbaz")},
		{description: "move down (on newline)", cursor: 3, y: 1, expectedCursor: 7,
			text: []rune("foo\nbar\nbaz")},
		{description: "move up (on newline, first line)", cursor: 3, y: -1, expectedCursor: 0,
			text: []rune("foo\nbar\nbaz")},
		{description: "move down (on newline, last line)", cursor: 7, y: 1, expectedCursor: 11,
			text: []rune("foo\nbar\nbaz")},
		{description: "move up (different lengths, short to long)", cursor: 8, y: -1, expectedCursor: 3,
			text: []rune("fool\nbar\nbaz")},
		{description: "move down (different lengths, short to long)", cursor: 3, y: 1, expectedCursor: 7,
			text: []rune("foo\nbare\nbaz")},
		{description: "move up (different lengths, long to short)", cursor: 8, y: -1, expectedCursor: 3,
			text: []rune("foo\nbare\nbaz")},
		{description: "move down (different lengths, long to short)", cursor: 4, y: 1, expectedCursor: 8,
			text: []rune("fool\nbar\nbaz")},
		{description: "move up (from empty line)", cursor: 4, y: -1, expectedCursor: 0,
			text: []rune("foo\n\nbaz")},
		{description: "move down (from empty line)", cursor: 4, y: 1, expectedCursor: 5,
			text: []rune("fool\n\nbaz")},
		{description: "move up (from empty line to empty line)", cursor: 5, y: -1, expectedCursor: 4,
			text: []rune("foo\n\n\n")},
		{description: "move down (from empty line to empty line)", cursor: 1, y: 1, expectedCursor: 2,
			text: []rune("\n\n\nfoo")},
		{description: "move up (from empty last line to empty line)", cursor: 3, y: -1, expectedCursor: 2,
			text: []rune("\n\n\n")},
		{description: "move down (from empty first line to empty line)", cursor: 0, y: 1, expectedCursor: 1,
			text: []rune("\n\n\n")},
		{description: "move up (from empty last line)", cursor: 6, y: -1, expectedCursor: 2,
			text: []rune("\n\nfoo\n")},
		{description: "move down (from empty first line)", cursor: 0, y: 1, expectedCursor: 1,
			text: []rune("\nfoo\n\n")},
		{description: "move up (from empty first line)", cursor: 0, y: -1, expectedCursor: 0,
			text: []rune("\n\nfoo\n")},
		{description: "move down (from empty last line)", cursor: 6, y: 1, expectedCursor: 6,
			text: []rune("\nfoo\n\n")},
		{description: "move up (from last line to empty line)", cursor: 2, y: -1, expectedCursor: 1,
			text: []rune("\n\nfoo")},
		{description: "move down (from first line to empty line)", cursor: 2, y: 1, expectedCursor: 4,
			text: []rune("foo\n\n")},
		{description: "move up (from empty line to empty line 2)", cursor: 2, y: -1, expectedCursor: 1,
			text: []rune("\n\n\n\n\n")},
		{description: "move down (from empty line to empty line 2)", cursor: 2, y: 1, expectedCursor: 3,
			text: []rune("\n\n\n\n\n")},
		//TODO: Tests to add:
		// moving cursor down at bottom of screen causing scroll
		// moving cursor up at top causing scroll
	}

	e := NewEditor()

	for _, tc := range tests {
		e.Cursor = tc.cursor
		e.Text = tc.text
		e.MoveCursor(tc.x, tc.y)

		got := e.Cursor
		expected := tc.expectedCursor

		if !cmp.Equal(got, expected) {
			t.Errorf("(%s) got != expected, diff: %v\n", tc.description, cmp.Diff(got, expected))
		}
	}
}

func TestScroll(t *testing.T) {
	{
		tests := []struct {
			description    string
			scrollAmt      int
			rowOff         int
			expectedRowOff int
			cursor         int
			expectedCursor int
			// runeAtCursor rune
			// expectedRuneAtCursor rune
			text []rune
		}{
			{description: "normal scroll down",
				scrollAmt: 1,
				rowOff:    0, expectedRowOff: 1,
				cursor: 0, expectedCursor: 2,
				// runeAtCursor: a, expectedRuneAtCursor: ,
				text: []rune("a\nb\nc\nd\ne\nf\ng\n")},

			{description: "normal scroll up",
				scrollAmt: -1,
				rowOff:    1, expectedRowOff: 0,
				cursor: 2, expectedCursor: 2,
				text: []rune("a\nb\nc\nd\ne\nf\ng\n")},

			{description: "normal scroll up",
				scrollAmt: -1,
				rowOff:    1, expectedRowOff: 0,
				cursor: 0, expectedCursor: 0,
				text: []rune("a\nb\nc\nd\ne\nf\ng\n")},
			//TODO: tests to add:
			// scrolling down with short text is rejected
			// scrolling up at top is rejected
			// scrolling down with cursor at top keeps cursor in same place
			// scrolling
		}

		e := NewEditor()
		e.Width = 80
		e.Height = 5

		for _, tc := range tests {
			e.RowOff = tc.rowOff
			e.Cursor = tc.cursor
			e.Text = tc.text

			e.Scroll(tc.scrollAmt)

			gotCursor := e.Cursor
			expectedCursor := tc.expectedCursor

			if !cmp.Equal(gotCursor, expectedCursor) {
				t.Errorf("(%s) Wrong cursor: got != expected, diff: %v\n", tc.description, cmp.Diff(gotCursor, expectedCursor))
			}

			gotRowOff := e.RowOff
			expectedRowOff := tc.expectedRowOff

			if !cmp.Equal(gotRowOff, expectedRowOff) {
				t.Errorf("(%s) Wrong offset: got != expected, diff: %v\n", tc.description, cmp.Diff(gotRowOff, expectedRowOff))
			}
		}
	}
}

// func TestSetCursorY(t *testing.T) {
// 	tests := []struct {
// 		description    string
// 		yDest          int
// 		rowOff         int
// 		expectedRowOff int
// 		cursor         int
// 		expectedCursor int
// 		// runeAtCursor rune
// 		// expectedRuneAtCursor rune
// 		text []rune
// 	}{
// 		{
// 			description: "Set cursor y to first line",
// 			yDest:       0,
// 		},
// 		{
// 			description: "Set cursor y to last line",
// 		},
// 		{
// 			description: "Set cursor y out of bounds, negative",
// 		},
// 		{
// 			description: "Set cursor y out of bounds, positive",
// 		},
// 		{
// 			description: "Set cursor y to start of editor window",
// 		},
// 		{
// 			description: "Set cursor y to end of editor window",
// 		},
// 	}

// 	e := NewEditor()

// 	for _, tc := range tests {
// 		e.Cursor = tc.cursor
// 		e.Text = tc.text

// 		e.setCursorY(tc.yDest)

// 		got := e.Cursor
// 		expected := tc.expectedCursor

// 		if !cmp.Equal(got, expected) {
// 			t.Errorf("(%s) got != expected, diff: %v\n", tc.description, cmp.Diff(got, expected))
// 		}
// 	}
// }
