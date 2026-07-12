package db

import "testing"

func TestInferDriver(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{name: "empty", dsn: "", want: "sqlite"},
		{name: "path", dsn: "data/superfolha.db", want: "sqlite"},
		{name: "absolute path", dsn: "/var/lib/superfolha/app.db", want: "sqlite"},
		{name: "file scheme path", dsn: "file:./app.db", want: "sqlite"},
		{name: "postgres scheme", dsn: "postgres://user:pass@localhost:5432/db", want: "postgres"},
		{name: "postgresql scheme", dsn: "postgresql://user:pass@localhost/db", want: "postgres"},
		{name: "postgres uppercase", dsn: "POSTGRES://user@host/db", want: "postgres"},
		{name: "postgresql mixed case", dsn: "PostgreSQL://user@host/db", want: "postgres"},
		{name: "postgres leading whitespace", dsn: "  postgres://user@host/db", want: "postgres"},
		{name: "postgres trailing whitespace", dsn: "postgres://user@host/db  ", want: "postgres"},
		{name: "postgres surrounding whitespace", dsn: "\t postgresql://user@host/db \n", want: "postgres"},
		{name: "non-postgres url", dsn: "mysql://user:pass@localhost/db", want: "sqlite"},
		{name: "http url", dsn: "https://example.com/db", want: "sqlite"},
		{name: "postgres substring not prefix", dsn: "notpostgres://host/db", want: "sqlite"},
		{name: "postgres in path only", dsn: "/tmp/postgres://weird.db", want: "sqlite"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := InferDriver(tt.dsn); got != tt.want {
				t.Fatalf("InferDriver(%q) = %q, want %q", tt.dsn, got, tt.want)
			}
		})
	}
}
