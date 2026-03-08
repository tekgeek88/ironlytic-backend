package main

import "testing"

// TestSayHello checks that the SayHello function returns the expected output
func TestSayHello(t *testing.T) {
	expected := "Hello, World!"
	got := SayHello()

	if got != expected {
		t.Errorf("Expected %q, but got %q", expected, got)
	}
}
