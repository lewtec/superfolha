package compiler

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/lewtec/superfolha/internal/project"
)

func TestCompile_LatexmkNotFound(t *testing.T) {
	// Empty PATH so LookPath("latexmk") fails before any project I/O.
	t.Setenv("PATH", t.TempDir())

	_, err := Compile(t.Context(), nil, "unused", "main.tex")
	if err == nil {
		t.Fatal("expected error when latexmk is missing")
	}
	if !errors.Is(err, ErrLatexmkNotFound) {
		t.Fatalf("errors.Is(err, ErrLatexmkNotFound) = false, err=%v", err)
	}
	if !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("errors.Is(err, exec.ErrNotFound) = false, err=%v", err)
	}
}

func TestCompile_InvalidFilePath(t *testing.T) {
	// Path validation runs before LookPath / project I/O so nil service is fine.
	cases := []string{
		"",
		".",
		"..",
		"../etc/passwd",
		"foo/../../etc/passwd",
		"/etc/passwd",
		"/tmp/main.tex",
	}
	for _, path := range cases {
		name := path
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			_, err := Compile(t.Context(), nil, "unused", path)
			if err == nil {
				t.Fatal("expected error for invalid compile path")
			}
			if !errors.Is(err, project.ErrInvalidPath) {
				t.Fatalf("errors.Is(err, ErrInvalidPath) = false, err=%v", err)
			}
			if errors.Is(err, ErrLatexmkNotFound) {
				t.Fatalf("should fail path jail before latexmk lookup, err=%v", err)
			}
		})
	}
}

func TestCompile_AcceptsNestedRelativePath(t *testing.T) {
	// Nested relative paths must pass the jail (latexmk may still be missing).
	t.Setenv("PATH", t.TempDir())

	_, err := Compile(t.Context(), nil, "unused", "chapters/intro.tex")
	if err == nil {
		t.Fatal("expected latexmk-not-found after path validation")
	}
	if errors.Is(err, project.ErrInvalidPath) {
		t.Fatalf("nested relative path should be valid, err=%v", err)
	}
	if !errors.Is(err, ErrLatexmkNotFound) {
		t.Fatalf("errors.Is(err, ErrLatexmkNotFound) = false, err=%v", err)
	}
}
