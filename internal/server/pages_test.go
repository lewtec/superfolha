package server

import (
	"testing"
	"time"

	"github.com/lewtec/superfolha/internal/paths"
)

func TestSafeNext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"", paths.Projects()},
		{paths.Projects(), paths.Projects()},
		{paths.Editor("abc"), paths.Editor("abc")},
		{"https://evil.example/phish", paths.Projects()},
		{"//evil.example", paths.Projects()},
		{paths.Login() + "?x=1", paths.Login() + "?x=1"},
	}
	for _, tt := range tests {
		if got := safeNext(tt.in); got != tt.want {
			t.Errorf("safeNext(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatProjectDate(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)
	if got := formatProjectDate(ts, "en"); got != "Aug 21, 2026" {
		t.Fatalf("en = %q", got)
	}
	if got := formatProjectDate(ts, "pt"); got != "21/08/2026" {
		t.Fatalf("pt = %q", got)
	}
}

func TestErrorIDUnknown(t *testing.T) {
	t.Parallel()
	if got := errorID(errPlain("nope")); got != "errors.UNKNOWN" {
		t.Fatalf("got %q", got)
	}
}

type errPlain string

func (e errPlain) Error() string { return string(e) }
