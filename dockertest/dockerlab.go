package dockertest

import (
	"errors"
	"net"
	"path"
	"testing"

	"github.com/minutelab/go-mlabtest"
)

// DockerLab is an mlab container running Docker
type DockerLab struct {
	lab *mlabtest.MLab
}

// New creates a new DockerLab object
// it starts a DockerLab of the specified version, and allows
// processes to connect to it
//
// if tb is not null the DockerLab object is related to this testing object:
// logs will be sent to it, and New either succeeds or fails the test with Fatal,
// so errors don't need to be tested.
//
// log is optional function to log stderr/stdout,
// can be nil and then default are used (either tb.Log or stdout)
func New(tb testing.TB, ver string, log func(string)) (*DockerLab, error) {
	docker, err := newDocker(tb, ver, log)
	if err != nil && tb != nil {
		tb.Fatal("Error starting DockerLab: ", err)
	}
	return docker, err
}

func newDocker(tb testing.TB, ver string, log func(string)) (*DockerLab, error) {
	scriptdir, err := mlabtest.GetSourceDir(DockerLab{})
	if err != nil {
		return nil, err
	}

	success := false

	lab, err := mlabtest.New(tb, path.Join(scriptdir, "dind.mlab"))
	if err != nil {
		return nil, err
	}
	defer func() {
		if !success {
			lab.Close()
		}
	}()

	lab.Stdout = mlabtest.NewLineLogger(mlabtest.DefaultLogger(tb, "docker:", log))
	lab.Stderr = lab.Stdout

	if err := lab.Start(); err != nil {
		return nil, err
	}
	success = true
	return &DockerLab{
		lab: lab,
	}, nil
}

// Log a string
func (p *DockerLab) Log(format string, a ...interface{}) { p.lab.Log(format, a...) }

// Close mlab
func (p *DockerLab) Close() error {
	p.Log("DockerLab:Close")
	return p.lab.Close()
}

// GetAddressPort return the address and port used to access the Docker
func (p *DockerLab) GetAddressPort() (net.IP, int, error) {
	return p.lab.GetAddressPort(2375)
}

// IP return the internal IP address of docker
func (p *DockerLab) IP() (net.IP, error) {
	conf, err := p.lab.NetConfig()
	if err != nil {
		return nil, err
	}
	if ip := conf.IP(); ip != nil {
		return ip, nil
	}
	return nil, errors.New("no IP for docker container")
}
