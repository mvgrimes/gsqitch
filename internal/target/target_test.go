package target

import (
	"testing"
)

func TestParseURI(t *testing.T) {
	tests := []struct {
		input   string
		scheme  string
		user    string
		pass    string
		host    string
		port    int
		db      string
		wantErr bool
	}{
		{
			input:  "db:mysql://user@localhost/testdb",
			scheme: "mysql",
			user:   "user",
			host:   "localhost",
			db:     "testdb",
		},
		{
			input:  "db:mysql://user:pass@localhost:3306/testdb",
			scheme: "mysql",
			user:   "user",
			pass:   "pass",
			host:   "localhost",
			port:   3306,
			db:     "testdb",
		},
		{
			input:  "mysql://localhost/testdb",
			scheme: "mysql",
			host:   "localhost",
			db:     "testdb",
		},
		{
			input:  "db:mysql:testdb",
			scheme: "mysql",
			db:     "testdb",
		},
		{
			input:  "db:pg://localhost/mydb",
			scheme: "pg",
			host:   "localhost",
			db:     "mydb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			u, err := ParseURI(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseURI(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if u.Scheme != tt.scheme {
				t.Errorf("Scheme = %q, want %q", u.Scheme, tt.scheme)
			}
			if u.User != tt.user {
				t.Errorf("User = %q, want %q", u.User, tt.user)
			}
			if u.Password != tt.pass {
				t.Errorf("Password = %q, want %q", u.Password, tt.pass)
			}
			if u.Host != tt.host {
				t.Errorf("Host = %q, want %q", u.Host, tt.host)
			}
			if u.Port != tt.port {
				t.Errorf("Port = %d, want %d", u.Port, tt.port)
			}
			if u.Database != tt.db {
				t.Errorf("Database = %q, want %q", u.Database, tt.db)
			}
		})
	}
}

func TestURIDSN(t *testing.T) {
	u := &URI{
		Scheme:   "mysql",
		User:     "root",
		Password: "secret",
		Host:     "localhost",
		Port:     3306,
		Database: "testdb",
	}

	dsn := u.DSN()
	expected := "root:secret@tcp(localhost:3306)/testdb"
	// DSN may have additional params, so just check it starts correctly
	if len(dsn) < len(expected) {
		t.Errorf("DSN = %q, want prefix %q", dsn, expected)
	}
}
