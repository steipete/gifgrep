package tui

import (
	"io"
	"strings"
	"testing"
	"time"
)

func TestEscapeDoesNotWaitForAnotherKey(t *testing.T) {
	r, w := io.Pipe()
	defer func() { _ = r.Close() }()
	defer func() { _ = w.Close() }()
	events, stop := setupInputReader(r)
	defer close(stop)
	if _, err := io.WriteString(w, "\x1b"); err != nil {
		t.Fatal(err)
	}
	select {
	case ev := <-events:
		if ev.kind != keyEsc {
			t.Fatalf("got %v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("Escape waits for another key")
	}
}

type notifiedReader struct {
	io.Reader
	read chan struct{}
}

func (r notifiedReader) Read(p []byte) (int, error) {
	n, e := r.Reader.Read(p)
	select {
	case r.read <- struct{}{}:
	default:
	}
	return n, e
}

func TestInputDeliveryStopsWhenCancelled(t *testing.T) {
	read := make(chan struct{}, 1)
	events, stop, done := make(chan inputEvent), make(chan struct{}), make(chan struct{})
	go func() { defer close(done); readInput(notifiedReader{strings.NewReader("x"), read}, events, stop) }()
	<-read
	close(stop)
	select {
	case <-done:
	case <-time.After(time.Second):
		// Release the old implementation's blocked send before reporting the regression.
		<-events
		t.Fatal("cancelled input remained blocked sending an event")
	}
}

func TestControlCInterruptsPartialEscapeSequence(t *testing.T) {
	events, stop := setupInputReader(strings.NewReader("\x1b[\x03"))
	defer close(stop)
	ev := <-events
	if ev.kind != keyCtrlC {
		t.Fatalf("partial arrow sequence swallowed Ctrl-C: %v", ev)
	}
}
