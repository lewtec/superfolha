package compiler

import (
	"errors"
	"os/exec"
	"testing"
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
