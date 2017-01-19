package pgtest

import "testing"

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
