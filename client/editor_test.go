package main

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
		e.cursor = tc.cursor
		e.AddRune(tc.r)
		if !cmp.Equal(e.GetText(), tc.expected) {
			t.Errorf("got != expected, diff: %v\n", cmp.Diff(e.text, tc.expected))
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
	e.text = []rune("content\ntest")

	for _, tc := range tests {
		e.cursor = tc.cursor
		x, y := e.calcCursorXY(e.cursor)

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
		expectedCursor int
	}{
		{description: "move forward", cursor: 0, x: 1, expectedCursor: 1},
		{description: "move backward", cursor: 1, x: -1, expectedCursor: 0},
		{description: "negative (out of bounds)", cursor: 0, x: -10, expectedCursor: 0},
		{description: "positive (out of bounds)", cursor: 12, x: 2, expectedCursor: 12},
	}

	e := NewEditor()
	e.text = []rune("content\ntest")

	for _, tc := range tests {
		e.cursor = tc.cursor
		e.MoveCursor(tc.x, 0)

		got := e.cursor
		expected := tc.expectedCursor

		if !cmp.Equal(got, expected) {
			t.Errorf("(%s) got != expected, diff: %v\n", tc.description, cmp.Diff(got, expected))
		}
	}
}
