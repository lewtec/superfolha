package project

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// ErrInvalidPath is returned when a file path is empty, absolute, or escapes the repository root.
var ErrInvalidPath = errors.New("invalid file path")

// ValidateRepoRelativePath cleans and validates a user-supplied path relative to a repository root.
// It rejects empty paths, absolute paths, paths that escape via "..", and any path under a ".git"
// directory component (so disk I/O cannot rewrite hooks/config the way collab CRDT already jails).
func ValidateRepoRelativePath(userPath string) (string, error) {
	if userPath == "" {
		return "", fmt.Errorf("%w: empty path", ErrInvalidPath)
	}

	// Normalize URL-style separators, then Clean.
	cleaned := filepath.Clean(filepath.FromSlash(userPath))
	if cleaned == "." {
		return "", fmt.Errorf("%w: empty path", ErrInvalidPath)
	}
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("%w: absolute paths are not allowed", ErrInvalidPath)
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: path escapes repository root", ErrInvalidPath)
	}
	if hasGitDirComponent(cleaned) {
		return "", fmt.Errorf("%w: .git paths are not allowed", ErrInvalidPath)
	}
	return cleaned, nil
}

// hasGitDirComponent reports whether cleaned contains a path segment exactly equal to ".git".
// ".gitignore" and similar names are allowed; only the git metadata directory is denied.
func hasGitDirComponent(cleaned string) bool {
	if cleaned == ".git" {
		return true
	}
	sep := string(filepath.Separator)
	for _, seg := range strings.Split(cleaned, sep) {
		if seg == ".git" {
			return true
		}
	}
	return false
}

// safeRepoPath joins userPath under repoRoot after validating it stays inside the repo.
// Returns the absolute (or repo-rooted) filesystem path on success.
func safeRepoPath(repoRoot, userPath string) (string, error) {
	rel, err := ValidateRepoRelativePath(userPath)
	if err != nil {
		return "", err
	}

	full := filepath.Join(repoRoot, rel)
	// Double-check with Rel so join/clean edge cases cannot slip past.
	check, err := filepath.Rel(repoRoot, full)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidPath, err)
	}
	if check == ".." || strings.HasPrefix(check, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: path escapes repository root", ErrInvalidPath)
	}
	if check == "." {
		return "", fmt.Errorf("%w: empty path", ErrInvalidPath)
	}
	return full, nil
}

// DecodeFilePath URL-decodes an encoded file path, normalizes separators, and rejects traversal.
func DecodeFilePath(encodedPath string) (string, error) {
	decoded, err := url.PathUnescape(encodedPath)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidPath, err)
	}
	return ValidateRepoRelativePath(decoded)
}
