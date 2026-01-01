package termcaps

import (
	"bytes"
	"os"
	"time"
)

// probeKittyGraphics implements the kitty docs recommendation:
// send a graphics protocol query (a=q) followed by primary device attributes (DA1).
// If DA1 is answered but the graphics query is not, kitty graphics are not supported.
func probeKittyGraphics(tty *os.File, timeout time.Duration) kittyProbeResult {
	if tty == nil {
		return kittyProbeUnknown
	}

	// Example from kitty docs:
	// <ESC>_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA<ESC>\<ESC>[c
	_, _ = tty.Write([]byte("\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\\x1b[c"))

	deadline := time.Now().Add(timeout)
	_ = tty.SetReadDeadline(deadline)

	var buf [1024]byte
	var acc []byte
	acc = make([]byte, 0, 2048)
	daSeen := false

	for time.Now().Before(deadline) {
		n, err := tty.Read(buf[:])
		if n > 0 {
			acc = append(acc, buf[:n]...)
			if bytes.Contains(acc, []byte("\x1b_Gi=31;")) || bytes.Contains(acc, []byte("\x1b_Gi=31,")) {
				return kittyProbeSupported
			}
			if hasDA1Response(acc) {
				daSeen = true
			}
		}
		if err != nil {
			break
		}
	}

	if daSeen {
		return kittyProbeNotSupported
	}
	return kittyProbeUnknown
}

func hasDA1Response(b []byte) bool {
	// Typical primary device attributes response: ESC [ ? ... c
	for i := 0; i+3 < len(b); i++ {
		if b[i] != 0x1b || b[i+1] != '[' {
			continue
		}
		j := i + 2
		if j < len(b) && b[j] == '?' {
			j++
		}
		for j < len(b) && j-i < 64 {
			ch := b[j]
			if ch == 'c' {
				return true
			}
			if (ch >= '0' && ch <= '9') || ch == ';' {
				j++
				continue
			}
			break
		}
	}
	return false
}
