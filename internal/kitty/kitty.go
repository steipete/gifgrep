package kitty

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/steipete/gifgrep/gifdecode"
)

type kittyData struct {
	Action      string
	ID          uint32
	Data        []byte
	Cols        int
	Rows        int
	PlacementID int
	Delay       time.Duration
	NoCursor    bool
}

func SendAnimation(out *bufio.Writer, id uint32, frames []gifdecode.Frame, cols, rows int) {
	if len(frames) == 0 {
		return
	}
	base := frames[0]
	sendKittyData(out, kittyData{
		Action:      "T",
		ID:          id,
		Data:        base.PNG,
		Cols:        cols,
		Rows:        rows,
		PlacementID: 1,
		NoCursor:    true,
	})
	for i := 1; i < len(frames); i++ {
		frame := frames[i]
		sendKittyData(out, kittyData{
			Action: "f",
			ID:     id,
			Data:   frame.PNG,
			Delay:  frame.Delay,
		})
	}
	sendKittyAnimDelay(out, id, delayMS(base.Delay))
	sendKittyAnimStart(out, id)
}

func SendFrame(out *bufio.Writer, id uint32, frame gifdecode.Frame, cols, rows int) {
	sendKittyData(out, kittyData{
		Action:      "T",
		ID:          id,
		Data:        frame.PNG,
		Cols:        cols,
		Rows:        rows,
		PlacementID: 1,
		NoCursor:    true,
	})
}

func sendKittyData(out *bufio.Writer, data kittyData) {
	encoded := base64.StdEncoding.EncodeToString(data.Data)
	const chunkSize = 4096
	first := true
	for encoded != "" {
		chunk := encoded
		if len(chunk) > chunkSize {
			chunk = chunk[:chunkSize]
		}
		encoded = encoded[len(chunk):]
		more := 0
		if encoded != "" {
			more = 1
		}
		if first {
			params := []string{
				fmt.Sprintf("a=%s", data.Action),
				"f=100",
				fmt.Sprintf("i=%d", data.ID),
				fmt.Sprintf("m=%d", more),
				"q=2",
			}
			if data.Cols > 0 {
				params = append(params, fmt.Sprintf("c=%d", data.Cols))
			}
			if data.Rows > 0 {
				params = append(params, fmt.Sprintf("r=%d", data.Rows))
			}
			if data.PlacementID > 0 {
				params = append(params, fmt.Sprintf("p=%d", data.PlacementID))
			}
			if data.NoCursor {
				params = append(params, "C=1")
			}
			if data.Action == "f" && data.Delay > 0 {
				params = append(params, fmt.Sprintf("z=%d", delayMS(data.Delay)))
			}
			_, _ = fmt.Fprintf(out, "\x1b_G%s;", strings.Join(params, ","))
			first = false
		} else {
			if data.Action == "f" {
				_, _ = fmt.Fprintf(out, "\x1b_Ga=f,m=%d;", more)
			} else {
				_, _ = fmt.Fprintf(out, "\x1b_Gm=%d;", more)
			}
		}
		_, _ = fmt.Fprint(out, chunk)
		_, _ = fmt.Fprint(out, "\x1b\\")
	}
}

func sendKittyAnimDelay(out *bufio.Writer, id uint32, delayMS int) {
	if delayMS <= 0 {
		return
	}
	_, _ = fmt.Fprintf(out, "\x1b_Ga=a,i=%d,r=1,z=%d,q=2\x1b\\", id, delayMS)
}

func sendKittyAnimStart(out *bufio.Writer, id uint32) {
	_, _ = fmt.Fprintf(out, "\x1b_Ga=a,i=%d,s=3,v=1,q=2\x1b\\", id)
}

func PlaceImage(out *bufio.Writer, id uint32, cols, rows int) {
	if id == 0 {
		return
	}
	_, _ = fmt.Fprintf(out, "\x1b_Ga=p,i=%d,p=1,c=%d,r=%d,C=1,q=2\x1b\\", id, cols, rows)
}

func DeleteImage(out *bufio.Writer, id uint32) {
	if id == 0 {
		return
	}
	_, _ = fmt.Fprintf(out, "\x1b_Ga=d,d=I,i=%d,q=2\x1b\\", id)
}

func clampDelay(delay time.Duration) time.Duration {
	if delay < 10*time.Millisecond {
		return 10 * time.Millisecond
	}
	if delay > time.Second {
		return time.Second
	}
	return delay
}

func delayMS(delay time.Duration) int {
	return int(clampDelay(delay).Milliseconds())
}
