package docker

import (
	"context"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

func Test_scanLogs(t *testing.T) {
	t.Run("scans lines", func(t *testing.T) {
		out := make(chan string, 3)
		scanLogs(context.Background(), strings.NewReader("first\nsecond\n"), out)
		close(out)

		var got []string
		for line := range out {
			got = append(got, line)
		}
		want := []string{"first", "second"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("scanLogs() = %#v, want %#v", got, want)
		}
	})

	t.Run("cancellation closes blocking reader", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		reader, writer := io.Pipe()
		defer writer.Close()
		done := make(chan struct{})
		go func() {
			scanLogs(ctx, reader, make(chan string))
			close(done)
		}()

		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("scanLogs did not stop after context cancellation")
		}
	})
}
