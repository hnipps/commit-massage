package log

import (
	"fmt"
	"time"
)

var frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner displays an animated spinner with a message.
type Spinner struct {
	done    chan struct{}
	stopped chan struct{}
}

// Start begins a spinner with the given message and returns it.
// Call Stop or Fail to end the spinner.
func Start(msg string) *Spinner {
	s := &Spinner{
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go func() {
		defer close(s.stopped)
		i := 0
		for {
			select {
			case <-s.done:
				return
			default:
				fmt.Printf("\r  %s %s", frames[i%len(frames)], msg)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
	return s
}

// Stop ends the spinner and prints a success message.
func (s *Spinner) Stop(msg string) {
	close(s.done)
	<-s.stopped
	fmt.Printf("\r\033[K  ✓ %s\n", msg)
}

// Fail ends the spinner and prints a failure message.
func (s *Spinner) Fail(msg string) {
	close(s.done)
	<-s.stopped
	fmt.Printf("\r\033[K  ✗ %s\n", msg)
}
