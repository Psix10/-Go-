package app

import (
	"net/url"
	"testing"
)

func TestDatabaseDSNUsesDatabaseURL(t *testing.T) {
	const want = "postgres://user:secret@127.0.0.1:5444/example?sslmode=disable"

	t.Setenv("DATABASE_URL", want)
	t.Setenv("DB_HOST", "ignored-host")
	t.Setenv("DB_PORT", "9999")

	if got := databaseDSN(); got != want {
		t.Fatalf("DSN = %q, want %q", got, want)
	}
}

func TestDatabaseDSNFromEnvironment(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("DB_PORT", "5444")
	t.Setenv("DB_USER", "test-user")
	t.Setenv("DB_PASS", "p@ss:word")
	t.Setenv("DB_NAME", "cashapp")

	u, err := url.Parse(databaseDSN())
	if err != nil {
		t.Fatal(err)
	}

	if u.Scheme != "postgres" {
		t.Errorf("scheme = %q, want postgres", u.Scheme)
	}

	if u.Host != "127.0.0.1:5444" {
		t.Errorf("host = %q, want 127.0.0.1:5444", u.Host)
	}

	if u.User.Username() != "test-user" {
		t.Errorf("username = %q, want test-user", u.User.Username())
	}

	password, ok := u.User.Password()
	if !ok || password != "p@ss:word" {
		t.Error("password was not preserved")
	}

	if u.Path != "/cashapp" {
		t.Errorf("path = %q, want /cashapp", u.Path)
	}

	if u.Query().Get("sslmode") != "disable" {
		t.Error("sslmode must be disable")
	}
}
