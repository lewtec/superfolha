package server

import "testing"

func TestSafeNext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"", "/projects"},
		{"/projects", "/projects"},
		{"/editor/abc", "/editor/abc"},
		{"https://evil.example/phish", "/projects"},
		{"//evil.example", "/projects"},
		{"/login?x=1", "/login?x=1"},
	}
	for _, tt := range tests {
		if got := safeNext(tt.in); got != tt.want {
			t.Errorf("safeNext(%q) = %q; want %q", tt.in, got, tt.want)
		}
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
