package remote

import (
	"errors"
	"testing"
)

func TestCanonical(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"https://github.com/Foo/Bar.git", "https://github.com/Foo/Bar"},
		{"https://GITHUB.COM/Foo/Bar/", "https://github.com/Foo/Bar"},
		{"git@github.com:Foo/Bar.git", "https://github.com/Foo/Bar"},
		{"https://user:token@github.com/Foo/Bar.git", "https://github.com/Foo/Bar"},
		{"github.com/Foo/Bar", "https://github.com/Foo/Bar"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			got := Canonical(tt.in)
			if got != tt.want {
				t.Errorf("Canonical(%q) = %q; want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseGitHub(t *testing.T) {
	t.Parallel()
	owner, repo, ok := ParseGitHub("git@github.com:lewtec/superfolha.git")
	if !ok || owner != "lewtec" || repo != "superfolha" {
		t.Fatalf("ParseGitHub = %q %q %v", owner, repo, ok)
	}
	if _, _, ok := ParseGitHub("https://gitlab.com/g/r"); ok {
		t.Fatal("gitlab must not parse as GitHub")
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()
	if err := Validate(""); !errors.Is(err, ErrEmptyRemote) {
		t.Fatalf("empty: %v", err)
	}
	if err := Validate("https://github.com/a/b"); err != nil {
		t.Fatalf("valid: %v", err)
	}
}
