package crdt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFlushRoundTrip(t *testing.T) {
	doc := New()
	err := doc.LoadFromFiles([]FileContent{
		{Path: "main.tex", Content: []byte("\\documentclass{article}\n")},
		{Path: "notes.md", Content: []byte("# hi\n")},
		{Path: "fig.png", Content: []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}},
	})
	if err != nil {
		t.Fatalf("LoadFromFiles: %v", err)
	}
	if got := doc.Source("main.tex"); got != "\\documentclass{article}\n" {
		t.Fatalf("main.tex = %q", got)
	}
	if _, ok := doc.TextPaths["fig.png"]; ok {
		t.Fatal("png should not be collaborative text")
	}
	if len(doc.TextPathList()) != 2 {
		t.Fatalf("paths = %v", doc.TextPathList())
	}

	dir := t.TempDir()
	if err := doc.FlushToDir(dir); err != nil {
		t.Fatalf("FlushToDir: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "main.tex"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "\\documentclass{article}\n" {
		t.Fatalf("flushed body = %q", body)
	}
	if _, err := os.Stat(filepath.Join(dir, "fig.png")); !os.IsNotExist(err) {
		t.Fatal("flush must not invent blob files")
	}
}

func TestSetTextServerAndRemove(t *testing.T) {
	doc := New()
	if err := doc.SetTextServer("a.tex", "one"); err != nil {
		t.Fatal(err)
	}
	if doc.Source("a.tex") != "one" {
		t.Fatalf("got %q", doc.Source("a.tex"))
	}
	if err := doc.SetTextServer("a.tex", "two"); err != nil {
		t.Fatal(err)
	}
	if doc.Source("a.tex") != "two" {
		t.Fatalf("got %q", doc.Source("a.tex"))
	}
	if err := doc.RemoveText("a.tex"); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc.TextPaths["a.tex"]; ok {
		t.Fatal("expected removed")
	}
}

func TestSkipOversizeText(t *testing.T) {
	doc := New()
	// Cap is 5 MiB; build slightly over without allocating full if possible — use 5MiB+1
	// This is heavy but once; skip if short tests.
	if testing.Short() {
		t.Skip("large allocation")
	}
	big := make([]byte, (5<<20)+1)
	for i := range big {
		big[i] = 'a'
	}
	if err := doc.LoadFromFiles([]FileContent{{Path: "huge.tex", Content: big}}); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc.TextPaths["huge.tex"]; ok {
		t.Fatal("oversize text must not enter CRDT")
	}
}
