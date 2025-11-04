package file

import (
	"testing"
)

func TestRingBuffer(t *testing.T) {
	rb := newRingBuffer(3)

	// Add fewer items than capacity
	rb.add("line1")
	rb.add("line2")
	lines := rb.getLines()
	expected := []string{"line1", "line2"}
	if len(lines) != len(expected) {
		t.Fatalf("Expected %d lines, got %d", len(expected), len(lines))
	}
	for i, exp := range expected {
		if lines[i] != exp {
			t.Errorf("Expected line %d to be %q, got %q", i, exp, lines[i])
		}
	}

	// Fill to capacity
	rb.add("line3")
	lines = rb.getLines()
	expected = []string{"line1", "line2", "line3"}
	if len(lines) != len(expected) {
		t.Fatalf("Expected %d lines, got %d", len(expected), len(lines))
	}
	for i, exp := range expected {
		if lines[i] != exp {
			t.Errorf("Expected line %d to be %q, got %q", i, exp, lines[i])
		}
	}

	// Overflow - should drop oldest
	rb.add("line4")
	rb.add("line5")
	lines = rb.getLines()
	expected = []string{"line3", "line4", "line5"}
	if len(lines) != len(expected) {
		t.Fatalf("Expected %d lines, got %d", len(expected), len(lines))
	}
	for i, exp := range expected {
		if lines[i] != exp {
			t.Errorf("Expected line %d to be %q, got %q", i, exp, lines[i])
		}
	}
}
