package mlabtest

import (
	"bufio"
	"fmt"
	"io"
	"testing"
)

// LogFunc is a type of function used to log lines of output
type LogFunc func(string)

// NewLineLogger allow to log lines of output produced by MLab containers
// It create an io.WriteCloser that parses the output
// into line and send them to the specified function.
func NewLineLogger(f LogFunc) io.WriteCloser {
	pr, pw := io.Pipe()

	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			f(scanner.Text())
		}
		pr.Close()
		pw.Close()
	}()

	return pw
}

// DefaultLogger create a "default" logging function
// If one is givven it is used as is,
// If not one is built, with the testing object if given, and with stdout
// otherwize
func DefaultLogger(tb testing.TB, prefix string, logger LogFunc) LogFunc {
	if logger != nil {
		return logger
	}
	if tb != nil {
		return func(line string) { tb.Log(prefix, line) }
	}
	return func(line string) { fmt.Println(prefix, line) }
}
