package mlabtest

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestMlab(t *testing.T) {
	lab, _ := New(t, "test.mlab")

	var outbuf bytes.Buffer
	lab.Stdout = &outbuf

	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	lab.Stderr = logger("stderr")

	lab.Start()
	defer lab.Close()
	t.Log("Output is:", strconv.Quote(outbuf.String()))
	if strings.TrimSpace(outbuf.String()) != "starting" {
		t.Error("Wrong output")
	}

	conf, err := lab.NetConfig()
	if err != nil {
		t.Fatal("failed getting ports:", err)
	}
	t.Log("Network configuiration", conf)

	if ip := conf.IP(); ip == nil {
		t.Error("No IP for container")
	} else {
		t.Log("Container IP is:", ip)
	}

	if p := conf.ExposedPorts[1000]; p == 0 {
		t.Error("Did not map port 1000")
	} else {
		t.Log("Port 1000 mapped to ", p)
	}
}

func logger(name string) io.Writer {
	return NewLineLogger(func(s string) {
		fmt.Fprintf(os.Stderr, "%s: %s\n", name, s)
	})
}
