# Integrate MinuteLab labs into go "Unit" tests

[![godoc reference](https://godoc.org/github.com/minutelab/go-mlabtest?status.png)](https://godoc.org/github.com/minutelab/go-mlabtest)
[![Build Status](https://travis-ci.org/minutelab/go-mlabtest.svg?branch=master)](https://travis-ci.org/minutelab/go-mlabtest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Support for integrating [MinuteLab](http://minutelab.io) labs inside go "unit" testing.

## Introduction

Write "unit" tests that can set up servers on demand to run for the duration of the test.

Suppose you have a code that interact with servers in its environment (database, Hadoop, Redis, Spark etc.)
Normally if you want to test the interaction with the environment you would have to specify that the "unit testing"
would be done in an environment that contain the right database.
This specification would mostly take the form of instructions to developers of the expected environment,
and encoded in some form into the continuous integration framework.

MinuteLab allows a different approach: the test code itself start on-demand the required environment.

* Different branches can have different environment requirements. This will be completely transparent
  to who ever run the unit tests (developers or CI system), since they will just run `go test`,
  and the needed environment will be set up on demand.

* Different tests on the same branch can use different environment, and it will be transparent
  to who ever happen to run the test.

* Minute Lab environment can include several servers, with private networking between them,
  and the testing environment. Normally "unit testing" is limited to whatever can be easily installed
  on one machine (either a developer desktop, or a cloud CI container).
  Minute Lab break this barrier and allow testing with environment containing clusters of servers

## Usage

The main way to use the the integration with the code testing is something like

```go
func TestSomething(t *testing.T) {}
  lab,_ := mlabtest.NewStart(t, "script.mlab","arg1","arg2")
  defer lab.Close()
}
```

This will cause the lab to startwhen the test start, and be cleaned up at the end.

If one need a lab that start for a whole pacakge it can be started in the TestMain method:

```go
func TestMain(m *testing.M) {
    lab,err := mlabtest.NewStart(t, "script.mlab","arg1","arg2")
    if err!=nil {
        return log.Fatal("Failed starting lab: ",err)
    }
    defer lab.Close()
	os.Exit(m.Run())
}
```

### Examples

This repository contain two examples, that are both useful on their own
we use them internally in Minute Lab), and show how to use the base functionality
to create easy testing for specific environments.

### Postgres example

the `pgtest` sub-package show how mlab tests can be used to test code interacting with
[PostgreSQL](https://www.postgresql.org).

When creating a new `pgtest.Postgres` object it start a database, and then the `GetDB` method
can be used to get a normal `*sql.DB` which can be used with the normal go function

```go
func TestPostgress(t *testing.T) {
	pg, _ := New(t, "", nil)
	defer pg.Close()

	db, err := pg.GetDB("")
	if err != nil {
		t.Fatal("Failed creating db object:", err)
	}

	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatal("Failed pinging databse:", err)
	}
}
```

### Postgres + Sqitch example

The `sqitchdb` sub-package go one step forward. It not only manage the database itself,
but also uses the [Sqitch](http://sqitch.org) tool to manage the schema.
When this object is created, it start a database, load a schema using the `sqitch` tool
(that does not to be installed because it is running in its own container),
and give back a go `*sql.DB` object ready for testing.

```go
func TestSqitch(t *testing.T) {
	dbLab, _ := New(t, "scheme", "", nil)
	defer dbLab.Close()

    db:=dblab.Conn()
    rows,err:=db.Query("SELECT * FROM users")
    if err!=nil {
        t.Errorf("Failed selecting from database: %s",err)
    }
}
```
