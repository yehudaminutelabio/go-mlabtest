package sqitchdb

import (
	"database/sql"
	"fmt"
	"testing"
)

func TestSqitch(t *testing.T) {
	db, _ := New(t, "scheme", "", nil)
	defer db.Close()

	if err := db.Conn().Ping(); err != nil {
		t.Error("Error pinging database:", err)
	}

	// basic test
	hasTable(t, db.Conn(), "table1")
	noSuchTable(t, db.Conn(), "tabel")

	// Now drop the hosts table
	if _, err := db.Conn().Exec("DROP TABLE table1"); err != nil {
		t.Fatal("Failed dropping table: ", err)
	}
	t.Log("Dropped table1")
	noSuchTable(t, db.Conn(), "table1")

	if _, err := db.Reset(); err != nil {
		t.Fatal("Failed reseting database", err)
	}
	t.Log("Reset DB")
	hasTable(t, db.Conn(), "table1")
}

func noSuchTable(t *testing.T, db *sql.DB, table string) {
	columns, err := getColumns(db, table)
	if err != nil {
		t.Logf("(expected) error reading from unknown table %s: %s", table, err)
		return
	}
	t.Errorf("Expected error selecting from %s, but got columns: %s", table, columns)
}

func hasTable(t *testing.T, db *sql.DB, table string) {
	columns, err := getColumns(db, table)
	if err != nil {
		t.Errorf("Error getting colums of %s: %s", table, err)
		return
	}
	t.Logf("table %s has columns: %s", table, columns)
}

func getColumns(db *sql.DB, table string) ([]string, error) {
	query := fmt.Sprintf("SELECT * FROM %s LIMIT 1", table)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return rows.Columns()
}
