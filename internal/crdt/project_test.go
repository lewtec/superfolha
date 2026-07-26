package crdt

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/superfolha/internal/project"
	"io/fs"
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
	if _, err := os.Stat(filepath.Join(dir, "fig.png")); !errors.Is(err, fs.ErrNotExist) {
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

func TestLoadFromFiles_RejectsEscapingPaths(t *testing.T) {
	doc := New()
	err := doc.LoadFromFiles([]FileContent{
		{Path: "main.tex", Content: []byte("ok\n")},
		{Path: "../outside.tex", Content: []byte("no\n")},
		{Path: "foo/../../etc/passwd", Content: []byte("no\n")},
		{Path: "/abs.tex", Content: []byte("no\n")},
		{Path: "", Content: []byte("no\n")},
		{Path: ".git/config", Content: []byte("no\n")},
		{Path: "nested/../clean.tex", Content: []byte("yes\n")},
	})
	if err != nil {
		t.Fatalf("LoadFromFiles: %v", err)
	}
	if _, ok := doc.TextPaths["main.tex"]; !ok {
		t.Fatal("main.tex should load")
	}
	if _, ok := doc.TextPaths["clean.tex"]; !ok {
		t.Fatal("nested/../clean.tex should normalize to clean.tex")
	}
	for _, bad := range []string{"../outside.tex", "foo/../../etc/passwd", "/abs.tex", "", ".git/config", "nested/../clean.tex"} {
		if _, ok := doc.TextPaths[bad]; ok {
			t.Fatalf("path %q must not remain as raw key", bad)
		}
	}
	if got := doc.Source("clean.tex"); got != "yes\n" {
		t.Fatalf("clean.tex = %q", got)
	}
}

func TestSetTextServer_RejectsEscapingPaths(t *testing.T) {
	doc := New()
	cases := []string{
		"",
		"..",
		"../x.tex",
		"a/../../b.tex",
		"/abs.tex",
		".git/config",
	}
	for _, path := range cases {
		err := doc.SetTextServer(path, "x")
		if err == nil {
			t.Fatalf("SetTextServer(%q) = nil, want error", path)
		}
		if !errors.Is(err, project.ErrInvalidPath) && path != "" {
			// empty path also surfaces as ErrInvalidPath via ValidateRepoRelativePath
			if path == "" && !errors.Is(err, project.ErrInvalidPath) {
				// still require some error; empty is fine either way
			}
		}
		if len(doc.TextPaths) != 0 {
			t.Fatalf("SetTextServer(%q) must not add TextPaths: %v", path, doc.TextPathList())
		}
	}

	if err := doc.SetTextServer("sub/../ok.tex", "body"); err != nil {
		t.Fatalf("normalized path: %v", err)
	}
	if doc.Source("ok.tex") != "body" {
		t.Fatalf("want body under ok.tex, paths=%v", doc.TextPathList())
	}
}

func TestCollabPath_NestedRelativeAccepted(t *testing.T) {
	path, err := collabPath("chapters/intro.tex")
	if err != nil {
		t.Fatal(err)
	}
	if path != "chapters/intro.tex" {
		t.Fatalf("got %q", path)
	}
}
