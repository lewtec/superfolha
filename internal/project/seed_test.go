package project

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureMainTeXCreatesWhenMissing(t *testing.T) {
	svc := NewService(t.TempDir())
	id := "p1"
	if err := svc.InitProjectRepo(id); err != nil {
		t.Fatal(err)
	}
	if err := svc.EnsureMainTeX(id); err != nil {
		t.Fatal(err)
	}
	r, _, err := svc.ReadFile(id, "main.tex")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	body, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `\documentclass{article}`) {
		t.Fatalf("main.tex = %q", body)
	}
	if err := svc.SaveFile(id, "main.tex", "keep\n"); err != nil {
		t.Fatal(err)
	}
	if err := svc.EnsureMainTeX(id); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(svc.GetProjectPath(id), "main.tex"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "keep\n" {
		t.Fatalf("EnsureMainTeX overwrote existing file: %q", got)
	}
}
