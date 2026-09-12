package tui

import (
	"bufio"
	"io"
	"strings"
	"testing"
	"time"
)

func TestInputEOFClosesEventStream(t *testing.T) {
	events, stop := setupInputReader(strings.NewReader(""))
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	done := make(chan bool, 1)
	go func() { done <- handleEvents(nil, bufio.NewWriter(io.Discard), nil, events, stop, nil, ticker) }()
	select {
	case quit := <-done:
		if !quit {
			t.Fatal("EOF did not quit")
		}
	case <-time.After(time.Second):
		close(stop)
		t.Fatal("EOF left event loop waiting")
	}
}
