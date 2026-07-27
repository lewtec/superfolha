package crdt

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lewtec/superfolha/internal/project"
	ycrdt "github.com/reearth/ygo/crdt"
)

// Origins used for Transact so observers can distinguish writers.
const (
	OriginServer = "superfolha.server"
	OriginLoad   = "superfolha.load"
)

var (
	// ErrTextTooLarge is returned when collaborative text exceeds MaxCollabTextBytes.
	ErrTextTooLarge = errors.New("text exceeds collab size cap")
	// ErrEmptyRootDir is returned when FlushToDir is called with an empty root.
	ErrEmptyRootDir = errors.New("empty root dir")
)

// ProjectDoc wraps a ygo document with multi-file text helpers.
// Only collaborative text bodies live here; the file tree and blobs stay on disk/Git.
type ProjectDoc struct {
	Doc *ycrdt.Doc
	// TextPaths is the set of relative paths that have a Y.Text in this doc.
	// Updated on load and when text files are added/removed server-side.
	// Keys are always slash-normalized, repo-relative, non-escaping paths.
	TextPaths map[string]struct{}
}

// New empty collaborative project document.
func New() *ProjectDoc {
	return &ProjectDoc{
		Doc:       ycrdt.New(),
		TextPaths: make(map[string]struct{}),
	}
}

// collabPath cleans and jails a path for CRDT text keys.
// Uses the same rules as disk I/O (ValidateRepoRelativePath) plus a .git deny list.
func collabPath(userPath string) (string, error) {
	clean, err := project.ValidateRepoRelativePath(userPath)
	if err != nil {
		return "", err
	}
	path := filepath.ToSlash(clean)
	if path == ".git" || strings.HasPrefix(path, ".git/") {
		return "", fmt.Errorf("%w: .git path not collaborative", project.ErrInvalidPath)
	}
	return path, nil
}

// TextKey returns the Y.Text root key for a relative path.
func TextKey(relPath string) string {
	// Normalize to slash paths for stable keys across OS.
	return "text:" + filepath.ToSlash(relPath)
}

// ApplyUpdate applies a Yjs binary update.
func (p *ProjectDoc) ApplyUpdate(update []byte) error {
	return p.Doc.ApplyUpdate(update)
}

// EncodeStateAsUpdate returns full document state as a Yjs update.
func (p *ProjectDoc) EncodeStateAsUpdate() []byte {
	return p.Doc.EncodeStateAsUpdate()
}

// Source returns collaborative text for path, or empty if not in the doc.
func (p *ProjectDoc) Source(relPath string) string {
	path, err := collabPath(relPath)
	if err != nil {
		return ""
	}
	if _, ok := p.TextPaths[path]; !ok {
		return ""
	}
	return p.Doc.GetText(TextKey(path)).ToString()
}

// TextPathList returns sorted collaborative paths.
func (p *ProjectDoc) TextPathList() []string {
	out := make([]string, 0, len(p.TextPaths))
	for path := range p.TextPaths {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

// FileContent is one file read while loading a project tree into the CRDT.
type FileContent struct {
	Path    string
	Content []byte
}

// LoadFromFiles populates Y.Text entries from a project file listing.
// Blobs, oversize text, and invalid/escaping paths are skipped (not collaborative).
// Expected on a fresh doc (hub open).
func (p *ProjectDoc) LoadFromFiles(files []FileContent) error {
	// Pre-create Y.Text handles outside the transaction (ygo requirement).
	texts := make(map[string]*ycrdt.YText)
	accepted := make([]FileContent, 0, len(files))
	for _, f := range files {
		path, err := collabPath(f.Path)
		if err != nil {
			continue
		}
		if project.IsBinary(f.Content, path) {
			continue
		}
		if int64(len(f.Content)) > project.MaxCollabTextBytes {
			continue
		}
		texts[path] = p.Doc.GetText(TextKey(path))
		accepted = append(accepted, FileContent{Path: path, Content: f.Content})
	}

	return p.Doc.TransactE(func(txn *ycrdt.Transaction) error {
		// Clear previous collaborative texts if reloading into same doc.
		for path := range p.TextPaths {
			st := p.Doc.GetText(TextKey(path))
			if n := st.Len(); n > 0 {
				st.Delete(txn, 0, n)
			}
		}
		p.TextPaths = make(map[string]struct{}, len(accepted))
		for _, f := range accepted {
			st := texts[f.Path]
			if n := st.Len(); n > 0 {
				st.Delete(txn, 0, n)
			}
			if len(f.Content) > 0 {
				st.Insert(txn, 0, string(f.Content), nil)
			}
			p.TextPaths[f.Path] = struct{}{}
		}
		return nil
	}, OriginLoad)
}

// SetTextServer replaces a path's collaborative text (server origin).
func (p *ProjectDoc) SetTextServer(relPath, content string) error {
	path, err := collabPath(relPath)
	if err != nil {
		return err
	}
	if int64(len(content)) > project.MaxCollabTextBytes {
		return ErrTextTooLarge
	}
	st := p.Doc.GetText(TextKey(path))
	err = p.Doc.TransactE(func(txn *ycrdt.Transaction) error {
		if n := st.Len(); n > 0 {
			st.Delete(txn, 0, n)
		}
		if content != "" {
			st.Insert(txn, 0, content, nil)
		}
		return nil
	}, OriginServer)
	if err != nil {
		return err
	}
	p.TextPaths[path] = struct{}{}
	return nil
}

// RemoveText drops a path from the collaborative set (does not delete disk file).
func (p *ProjectDoc) RemoveText(relPath string) error {
	path, err := collabPath(relPath)
	if err != nil {
		// Invalid paths cannot be in TextPaths when keys are jail-normalized.
		return nil
	}
	if _, ok := p.TextPaths[path]; !ok {
		return nil
	}
	st := p.Doc.GetText(TextKey(path))
	err = p.Doc.TransactE(func(txn *ycrdt.Transaction) error {
		if n := st.Len(); n > 0 {
			st.Delete(txn, 0, n)
		}
		return nil
	}, OriginServer)
	if err != nil {
		return err
	}
	delete(p.TextPaths, path)
	return nil
}

// FlushToDir writes all collaborative text bodies under rootDir (project working tree).
// Creates parent directories as needed. Does not touch blob files or delete missing paths.
func (p *ProjectDoc) FlushToDir(rootDir string) error {
	if rootDir == "" {
		return ErrEmptyRootDir
	}
	for _, rel := range p.TextPathList() {
		// Re-validate path jail (defense in depth; keys should already be clean).
		clean, err := project.ValidateRepoRelativePath(rel)
		if err != nil {
			return fmt.Errorf("flush %q: %w", rel, err)
		}
		abs := filepath.Join(rootDir, filepath.FromSlash(clean))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return err
		}
		body := p.Source(rel)
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", clean, err)
		}
	}
	return nil
}

// ReadAllProjectFiles loads every file from projectService for CRDT load.
func ReadAllProjectFiles(svc *project.Service, projectID string) ([]FileContent, error) {
	list, err := svc.ListFiles(projectID)
	if err != nil {
		return nil, err
	}
	out := make([]FileContent, 0, len(list))
	for _, fi := range list {
		rc, _, err := svc.ReadFile(projectID, fi.Path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", fi.Path, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		out = append(out, FileContent{Path: fi.Path, Content: data})
	}
	return out, nil
}
