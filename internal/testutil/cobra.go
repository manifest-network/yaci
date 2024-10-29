package testutil

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func Execute(t *testing.T, c *cobra.Command, args ...string) (string, error) {
	t.Helper()

	// Capture the output of the command to a string
	// https://stackoverflow.com/questions/10473800/in-go-how-do-i-capture-stdout-of-a-function-into-a-string#comment46866149_10476304
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()

	c.SetArgs(args)
	err = c.Execute()

	w.Close()
	os.Stdout = old
	out := <-outC

	return strings.TrimSpace(out), err
}
