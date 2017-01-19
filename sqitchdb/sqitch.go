// Package sqitchdb allow "unit testing" of (postgres) database whose schema is controlled by sqitch
//
// Calling New will start a container running postgres with the latest schema/data loaded.
// The Conn method return a connection to this database (sql.DB) that can be used for testing.
// Close will shutdown the container.
//
// Reset will return an open database to the state that it was initially.
// So the idea is that it in a TestMain (or something like that) the database would start
// and then individual tests can start by calling Reset
package sqitchdb

import (
	"bytes"
	"database/sql"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/minutelab/go-mlabtest"
	"github.com/minutelab/go-mlabtest/pgtest"
)

const testDB = "testdb"

// DB represent a running database
type DB struct {
	lab        *pgtest.Postgres
	globalConn *sql.DB
	conn       *sql.DB
	pgclient   string
	resetState []byte
	ip         string // internal IP of database
}

// New start a container and load the schema.
// dir is the directory that contain the sqlitch schema
// ver is the postgres version
// if tb is not nil it is used to fail tests (so that New does not return error)
// as well as a default for the logger function
func New(tb testing.TB, dir string, ver string, logger func(string)) (*DB, error) {
	db, err := doNew(tb, dir, ver, logger)
	if err != nil && tb != nil {
		tb.Fatal("failed initializing database: ", err)
	}
	return db, err
}

func doNew(tb testing.TB, dir string, ver string, logger func(string)) (*DB, error) {
	// We will need the directory later, lets get it while it is quick and easy to fail
	sqitchdir, err := mlabtest.GetSourceDir(DB{})
	if err != nil {
		return nil, err
	}

	logger = mlabtest.DefaultLogger(tb, "db:", logger)

	// run the database
	pg, err := pgtest.New(tb, ver, logger)
	if err != nil {
		return nil, err
	}

	logger("started database")

	ip, err := pg.IP()
	if err != nil {
		return nil, err
	}

	db := DB{
		lab:      pg,
		pgclient: filepath.Join(sqitchdir, "pgclient.mlab"),
		ip:       ip.String(),
	}
	success := false
	defer func() {
		if !success {
			db.Close()
		}
	}()

	db.globalConn, err = pg.GetDB("")
	if err != nil {
		return nil, err
	}

	_, err = db.globalConn.Exec("CREATE DATABASE " + testDB)
	if err != nil {
		return nil, err
	}

	db.conn, err = pg.GetDB(testDB)
	if err != nil {
		return nil, err
	}

	deployCmd := exec.Command("mlab", "run", filepath.Join(sqitchdir, "sqitch.mlab"), "-host", db.ip, "-port", "5432", "-schema", dir, "--", "--db-name", testDB, "deploy")
	lineLogger := mlabtest.NewLineLogger(logger)
	deployCmd.Stdout = lineLogger
	deployCmd.Stderr = lineLogger
	if err := deployCmd.Run(); err != nil {
		return nil, err
	}

	logger("Getting schema")
	dumpCmd := db.clientCmd("pg_dump", "-C", testDB)
	dumpCmd.Stderr = lineLogger
	out, err := dumpCmd.Output()
	// fmt.Fprintln(os.Stderr, string(out))
	if err != nil {
		db.lab.Log("Error running dump: %s", err)
		return nil, fmt.Errorf("Error running pg_dump: %s", err)
	}
	db.lab.Log("Got schema: %d bytes", len(out))

	db.resetState = out
	success = true
	return &db, nil
}

// Close shut down the contaienr
func (d *DB) Close() error { return d.lab.Close() }

// Conn return a database connection to be used in testing
func (d *DB) Conn() *sql.DB { return d.conn }

// Reset the database to its original state
func (d *DB) Reset() (*sql.DB, error) {
	if err := d.conn.Close(); err != nil {
		d.lab.Log("Error closing db: %s", err)
	}
	// It seems that even though we close the connection the go postgres implementation doesn't close the connections to the database
	// so the database think that there are still user intereseted in the db and prevent us from dropping it.
	// its a hack, but we just forcfully remove those connections`
	if _, err := d.globalConn.Exec(fmt.Sprintf("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s'", testDB)); err != nil {
		d.lab.Log("Failed removing connections: %s", err)
	}

	if _, err := d.globalConn.Exec("DROP DATABASE " + testDB); err != nil {
		return nil, err
	}

	restoreCmd := d.clientCmd("")
	restoreCmd.Stdin = bytes.NewReader(d.resetState)
	if out, err := restoreCmd.CombinedOutput(); err != nil {
		d.lab.Log("Failed reseting database err='%s', out='%s'", err, string(out))
		return nil, err
	}

	var err error
	d.conn, err = d.lab.GetDB(testDB)
	return d.conn, err
}

func (d *DB) clientCmd(cmd string, clientArgs ...string) *exec.Cmd {
	args := []string{"run", d.pgclient, "-host", d.ip}
	if cmd != "" {
		args = append(args, "-cmd", cmd)
	}
	args = append(args, "--")
	args = append(args, clientArgs...)
	d.lab.Log("creating client command: %s %s", d.pgclient, args)
	return exec.Command("mlab", args...)
}
