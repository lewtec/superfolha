package project

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed templates/simple/main.tex
var defaultMainTeX embed.FS

// EnsureMainTeX writes the default template when dest/main.tex is missing.
func (s *Service) EnsureMainTeX(projectID string) error {
	dest := filepath.Join(s.GetProjectPath(projectID), "main.tex")
	if _, err := os.Stat(dest); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	body, err := defaultMainTeX.ReadFile("templates/simple/main.tex")
	if err != nil {
		return err
	}
	return s.SaveFile(projectID, "main.tex", string(body))
}
