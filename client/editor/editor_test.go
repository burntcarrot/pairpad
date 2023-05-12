package editor

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCalcXY(t *testing.T) {
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

	e := NewEditor(EditorConfig{})
	e.Text = []rune("content\ntest")

	for _, tc := range tests {
		e.Cursor = tc.cursor
		x, y := e.calcXY(e.Cursor)

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
	}

	e := NewEditor(EditorConfig{})

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
			x              int
			y              int
			colOff         int
			expectedColOff int
			rowOff         int
			expectedRowOff int
			cursor         int
			expectedCursor int
			text           []rune
		}{
			{description: "scroll down",
				y:      1,
				colOff: 0, expectedColOff: 0,
				rowOff: 0, expectedRowOff: 1,
				cursor: 6, expectedCursor: 8,
				text: []rune("a\nb\nc\nd\ne")},

			{description: "scroll up",
				y:      -1,
				colOff: 0, expectedColOff: 0,
				rowOff: 1, expectedRowOff: 0,
				cursor: 2, expectedCursor: 0,
				text: []rune("a\nb\nc\nd\ne")},

			{description: "scroll right",
				x:      1,
				colOff: 0, expectedColOff: 1,
				rowOff: 0, expectedRowOff: 0,
				cursor: 4, expectedCursor: 5,
				text: []rune("abcde")},

			{description: "scroll left",
				x:      -1,
				colOff: 1, expectedColOff: 0,
				rowOff: 0, expectedRowOff: 0,
				cursor: 1, expectedCursor: 0,
				text: []rune("abcde")},

			{description: "horizontal jump backwards",
				x:      1,
				colOff: 4, expectedColOff: 0,
				rowOff: 0, expectedRowOff: 0,
				cursor: 8, expectedCursor: 9,
				text: []rune("abcdefgh\nijk")},

			{description: "horizontal jump forwards",
				x:      -1,
				colOff: 0, expectedColOff: 4,
				rowOff: 0, expectedRowOff: 0,
				cursor: 9, expectedCursor: 8,
				text: []rune("abcdefgh\nijk")},
		}

		e := NewEditor(EditorConfig{
			ScrollEnabled: true,
		})
		e.Width = 5
		e.Height = 5

		for _, tc := range tests {
			e.ColOff = tc.colOff
			e.RowOff = tc.rowOff
			e.Cursor = tc.cursor
			e.Text = tc.text

			e.MoveCursor(tc.x, tc.y)

			gotCursor := e.Cursor
			expectedCursor := tc.expectedCursor

			if !cmp.Equal(gotCursor, expectedCursor) {
				t.Errorf("(%s) Wrong cursor: got != expected, diff: %v\n", tc.description, cmp.Diff(gotCursor, expectedCursor))
			}

			gotRowOff := e.RowOff
			expectedRowOff := tc.expectedRowOff

			if !cmp.Equal(gotRowOff, expectedRowOff) {
				t.Errorf("(%s) Wrong row offset: got != expected, diff: %v\n", tc.description, cmp.Diff(gotRowOff, expectedRowOff))
			}

			gotColOff := e.ColOff
			expectedColOff := tc.expectedColOff

			if !cmp.Equal(gotColOff, expectedColOff) {
				t.Errorf("(%s) Wrong col offset: got != expected, diff: %v\n", tc.description, cmp.Diff(gotColOff, expectedColOff))
			}
		}
	}
}
