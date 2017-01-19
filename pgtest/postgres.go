package pgtest

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"path"
	"testing"

	_ "github.com/lib/pq" // we use postgres, we need to iport the library for side effect

	"github.com/minutelab/go-mlabtest"
)

// Postgres is an mlab container running postgres
type Postgres struct {
	lab *mlabtest.MLab
}

// New create a new Postgres object
// it start a postgress database of the specified version, and allow
// processes to connect to it
//
// if tb is not null the postgres object is related to this testing object:
// logs will be sent to it, and New either succeed or fail the test with Fatal,
// so errors don't need to be tested.
//
// log is optional function to log stderr/stdout of the database,
// can be nil and then default are used (either tb.Log or stdout)
func New(tb testing.TB, ver string, log func(string)) (*Postgres, error) {
	pg, err := newPostgres(tb, ver, log)
	if err != nil && tb != nil {
		tb.Fatal("Error starting postgres: ", err)
	}
	return pg, err
}

func newPostgres(tb testing.TB, ver string, log func(string)) (*Postgres, error) {
	scriptdir, err := mlabtest.GetSourceDir(Postgres{})
	if err != nil {
		return nil, err
	}

	success := false

	args := []string{"-port", "0", "-detach"}
	if ver != "" {
		args = append(args, "-ver", ver)
	}
	lab, err := mlabtest.New(tb, path.Join(scriptdir, "postgres.mlab"), args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if !success {
			lab.Close()
		}
	}()

	lab.Stdout = mlabtest.NewLineLogger(mlabtest.DefaultLogger(tb, "pg:", log))
	lab.Stderr = lab.Stdout

	if err := lab.Start(); err != nil {
		return nil, err
	}

	success = true
	return &Postgres{
		lab: lab,
	}, nil
}

// Log a string
func (p *Postgres) Log(format string, a ...interface{}) { p.lab.Log(format, a...) }

// Close releases resources connected to the postgres object (in particular kill the container)
func (p *Postgres) Close() error {
	p.lab.Log("Posgres:Close")
	return p.lab.Close()
}

// GetDB get an sql.DB object conected to the postgres with the specified database name
func (p *Postgres) GetDB(name string) (*sql.DB, error) {
	ip, port, err := p.GetAddressPort()
	if err != nil {
		return nil, err
	}
	postgresurl := fmt.Sprintf("postgres://postgres@%s:%d/%s?sslmode=disable", ip.String(), port, name)
	p.Log("DB URL: %s", postgresurl)
	return sql.Open("postgres", postgresurl)
}

// GetAddressPort return the address and port used to access the DB
func (p *Postgres) GetAddressPort() (net.IP, int, error) { return p.lab.GetAddressPort(5432) }

// IP return the internal IP address of postgress
func (p *Postgres) IP() (net.IP, error) {
	conf, err := p.lab.NetConfig()
	if err != nil {
		return nil, err
	}
	if ip := conf.IP(); ip != nil {
		return ip, nil
	}
	return nil, errors.New("no IP for postgess container")
}
