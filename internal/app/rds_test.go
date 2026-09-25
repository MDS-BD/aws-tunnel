package app

import "testing"

func TestConnectHint(t *testing.T) {
	cases := []struct {
		engine   string
		iam      bool
		expected string
	}{
		{"aurora-postgresql", false, `psql "host=localhost port=5433 user=admin dbname=postgres sslmode=require"`},
		{"postgres", true, `psql "host=localhost port=5433 user=admin dbname=postgres sslmode=require"`},
		{"aurora-mysql", false, "mysql -h 127.0.0.1 -P 5433 -u admin --ssl-mode=REQUIRED -p"},
		{"mariadb", true, "mysql -h 127.0.0.1 -P 5433 -u admin --ssl-mode=REQUIRED -p --enable-cleartext-plugin"},
		{"sqlserver-ex", false, ""},
	}
	for _, c := range cases {
		if got := connectHint(c.engine, "5433", "admin", c.iam); got != c.expected {
			t.Errorf("connectHint(%q, iam=%v) mismatch:\nGot: %s\nExpected: %s", c.engine, c.iam, got, c.expected)
		}
	}
}
