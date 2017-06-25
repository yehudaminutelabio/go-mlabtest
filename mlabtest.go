// Package mlabtest is used to run unit tests that set up a whole lab
package mlabtest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

var loopbackAddress = net.ParseIP("127.0.0.1")

// MLab represent a running minutelab
//
// After creating the lab object, and possibly setting other paramters,
// the lab is started with the Start method.
//
// The lab should be closed with the Close method (even if it was only created or if start return an error)
// The lab should be built such that when it is ready it will detach
type MLab struct {
	cmd        *exec.Cmd
	outStream  io.Writer
	outErr     io.Writer
	idfile     string        // file containing the container id
	id         string        // the container id
	closechan  chan struct{} // this channel will be closed once the lab process is dead (or if start failed to start it)
	toBeClosed []io.Closer
	netConfig  *NetConfig

	t testing.TB // testing object related to this lab

	// if logger is set it would be called to log "internal" MLab events
	Logger func(string)
	// if Stdin is not nill it is connected to the mlab
	// otherwise it is connected to the null device.
	// Must be set before calling Start
	Stdin io.Reader
	// id Stdout and/or Stderr are not nill they are connected to the mlab
	// (stdout is not connected directly)
	Stdout io.Writer
	Stderr io.Writer
}

// NetConfig is the network configuration of a lab
type NetConfig struct {
	Interfaces   map[string]net.IP
	ExposedPorts map[int]int
}

// IP get the main IP of the lab
func (n *NetConfig) IP() net.IP {
	if n == nil || len(n.Interfaces) == 0 {
		return nil
	}

	// try eth0
	if ip, ok := n.Interfaces["eth0"]; ok {
		return ip
	}

	// get the first
	for _, ip := range n.Interfaces {
		return ip
	}

	return nil
}

// New create (but does not start) a new mlab with the specified arguments
// The lab can be optionally associated with a testing framework object
// (like *testing.T), if it does it behave in a way more suitable
// for unit testing:
//
// 1. Both New and Start won't return errors, instead the would abort the test
//    using Fatal, so testing code does not need to explicitly check for errors
// 2. It will send logs about lab setup through the testing log function
func New(tb testing.TB, script string, args ...string) (*MLab, error) {
	// create temporary file to hold the container id
	idfile, err := ioutil.TempFile("", "tmlab.")
	if err != nil {
		if tb != nil {
			tb.Fatal("mlabtest::New failed creating temp file:", err)
		}
		return nil, err
	}
	idfname := idfile.Name()
	idfile.Close()

	pargs := []string{"run", "--wait", "--id", idfname, script}
	pargs = append(pargs, args...)

	lab := &MLab{
		cmd:       exec.Command("mlab", pargs...),
		idfile:    idfname,
		t:         tb,
		closechan: make(chan struct{}),
	}
	if tb != nil {
		lab.Logger = func(line string) { tb.Log(line) }
	}
	return lab, nil
}

// NewStart create a new lab and start it immediatly
// if tb is not nil testing code can assume that the function succeed
// and not test for errors
func NewStart(tb testing.TB, script string, args ...string) (*MLab, error) {
	lab, err := New(tb, script, args...)
	if err != nil {
		return nil, err
	}
	lab.Logger = func(line string) { fmt.Println(line) }
	return lab, lab.Start()
}

// Log a line to the mlab configured logger
func (m *MLab) Log(format string, a ...interface{}) {
	if m.Logger != nil {
		m.Logger(fmt.Sprintf(format, a...))
	}
}

// Close kills the mlab if neccesary and clean after it
func (m *MLab) Close() error {
	m.Log("MLab:Close")
	if !m.IsClosed() {
		m.Log("Actually killing process")
		// process is still alive
		m.cmd.Process.Kill()
	}

	for _, c := range m.toBeClosed {
		c.Close()
	}

	os.Remove(m.idfile)
	return nil
}

// Start the lab
func (m *MLab) Start() error {
	err := m.doStart()
	if err != nil && m.t != nil {
		m.t.Fatal("Failed starting MLab")
	}
	return err
}

func (m *MLab) doStart() error {
	if m.Stdin != nil {
		m.cmd.Stdin = m.Stdin
	} else {
		// TODO: at the time of writing, if we want to detach+wait, we cannot close stdin
		// So if we don't have anything else (the normal thing) we use a reader that blocks
		br := newBlockingReader()
		m.cmd.Stdin = br
		m.toBeClosed = append(m.toBeClosed, br)
	}

	if m.Stderr != nil {
		m.cmd.Stderr = m.Stderr
	}

	out := m.Stdout
	if out == nil {
		out = &nullWriter{}
	}

	pipe, err := m.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := m.cmd.Start(); err != nil {
		close(m.closechan)
		return err
	}

	go func() {
		m.cmd.Wait()
		close(m.closechan)
	}()

	if _, err := io.Copy(out, pipe); err != nil {
		return err
	}

	select {
	case <-m.closechan:
		return fmt.Errorf("mlab exited: %s", m.cmd.ProcessState.String())

	case <-time.NewTimer(50 * time.Millisecond).C:
		// TODO: do we really need to wait here
	}

	id, err := ioutil.ReadFile(m.idfile)
	if err != nil {
		return err
	}
	m.id = string(bytes.TrimSpace(id))
	return nil
}

// Wait until the lab died (or start failed to start it)
func (m *MLab) Wait() {
	<-m.closechan
}

// IsClosed true if the command is closed
func (m *MLab) IsClosed() bool {
	select {
	case <-m.closechan:
		return true
	default:
		return false
	}
}

// NetConfig get the network configuration of a running container
// result is cached, so future calls are fast
func (m *MLab) NetConfig() (*NetConfig, error) {
	if m.netConfig != nil {
		return m.netConfig, nil
	}

	nc, err := m.getNetConfig()
	if err == nil {
		m.netConfig = nc
	}
	return nc, err
}

func (m *MLab) getNetConfig() (*NetConfig, error) {
	out, err := exec.Command("mlab", "inspect", "-f", `{{json .config.Network.interfaces}} {{json .config.Network.exposed}}`, m.id).CombinedOutput()
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewBuffer(out))

	// decode interfaces
	var ifcs []struct {
		Address string `json:"address"`
		IFName  string `json:"ifname"`
	}
	if err := decoder.Decode(&ifcs); err != nil {
		return nil, err
	}

	interfaces := make(map[string]net.IP)
	for _, i := range ifcs {
		ip := net.ParseIP(i.Address)
		if ip == nil {
			return nil, fmt.Errorf("error parsing address (%s) of %s", i.Address, i.IFName)
		}
		interfaces[i.IFName] = ip
	}

	// decode ports
	var portList []struct {
		Internal int `json:"internal"`
		External int `json:"external"`
	}
	if err := decoder.Decode(&portList); err != nil {
		return nil, err
	}

	ports := make(map[int]int)
	for _, p := range portList {
		ports[p.Internal] = p.External
	}

	return &NetConfig{
		Interfaces:   interfaces,
		ExposedPorts: ports,
	}, nil
}

// GetAddressPort return the address and port to be used to access the specified internal port
func (m *MLab) GetAddressPort(port int) (net.IP, int, error) {
	conf, err := m.NetConfig()
	if err != nil {
		return nil, 0, err
	}

	if !NeedForwarding() {
		ip := conf.IP()
		if ip != nil {
			return ip, port, nil
		}
		// we don't have IP? we would probably fail, but let's try also forwarding
	}

	if fport, ok := conf.ExposedPorts[port]; ok {
		return loopbackAddress, fport, nil
	}
	return nil, 0, fmt.Errorf("could not find port mapping for %d", port)
}

// NeedForwarding return true if the labs need to be accessed at 127.0.0.1 with the mapped ports
// otherwise they can be accessed at their own address with the original port
func NeedForwarding() bool {
	return !strings.HasPrefix(os.Getenv("MLAB_HOST"), "unix:")
}

type nullWriter struct{}

func (n *nullWriter) Write(p []byte) (int, error) { return len(p), nil }

// blockingRreader is a dummy io.ReadCloser: any read on it will block,
// until it is closed. Once it is closed, any reads (past and future)
// will return immediatly with no data and EOF
type blockingReader struct {
	once sync.Once
	c    chan interface{}
}

func newBlockingReader() io.ReadCloser {
	return &blockingReader{c: make(chan interface{})}
}

func (b *blockingReader) Read(_ []byte) (int, error) {
	<-b.c
	return 0, io.EOF
}

func (b *blockingReader) Close() error {
	b.once.Do(func() { close(b.c) })
	return nil
}
