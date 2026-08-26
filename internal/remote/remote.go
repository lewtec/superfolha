// Package remote canonicalizes git HTTP URLs and parses GitHub owner/repo pairs.
package remote

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"
)

var (
	ErrEmptyRemote   = errors.New("empty remote")
	ErrInvalidRemote = errors.New("invalid remote")
	ErrNotSSH        = errors.New("ssh remote required")
)

// Key is the uniqueness key for a live session: canonical remote + branch.
func Key(canonicalRemote, branch string) string {
	return Canonical(canonicalRemote) + "\x00" + strings.TrimSpace(branch)
}

// Canonical returns a stable HTTP URL: lowercase host, no .git suffix, no userinfo, no fragment.
// SSH scp-like URLs (git@host:path) become https://host/path.
func Canonical(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if IsLocal(raw) {
		p, err := LocalPath(raw)
		if err != nil {
			return raw
		}
		return "file://" + p
	}
	if u, ok := scpURL(raw); ok {
		return u
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return strings.TrimSuffix(strings.TrimRight(raw, "/"), ".git")
	}
	u.User = nil
	u.Fragment = ""
	u.RawQuery = ""
	u.Host = strings.ToLower(u.Host)
	u.Scheme = strings.ToLower(u.Scheme)
	if u.Scheme == "ssh" || u.Scheme == "git" {
		u.Scheme = "https"
	}
	u.Path = strings.TrimSuffix(strings.TrimRight(u.Path, "/"), ".git")
	return u.String()
}

func scpURL(raw string) (string, bool) {
	// git@github.com:owner/repo.git
	if strings.Contains(raw, "://") {
		return "", false
	}
	at := strings.IndexByte(raw, '@')
	colon := strings.LastIndexByte(raw, ':')
	if at <= 0 || colon <= at {
		return "", false
	}
	host := strings.ToLower(raw[at+1 : colon])
	path := strings.TrimSuffix(strings.TrimRight(raw[colon+1:], "/"), ".git")
	if host == "" || path == "" {
		return "", false
	}
	return "https://" + host + "/" + path, true
}

// ParseGitHub returns owner and repo for github.com remotes.
func ParseGitHub(raw string) (owner, repo string, ok bool) {
	c := Canonical(raw)
	u, err := url.Parse(c)
	if err != nil {
		return "", "", false
	}
	host := strings.ToLower(u.Hostname())
	if host != "github.com" && host != "www.github.com" {
		return "", "", false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// IsSSH reports whether raw is an SSH git URL (scp or ssh://).
func IsSSH(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	if strings.HasPrefix(strings.ToLower(raw), "ssh://") {
		return true
	}
	_, ok := scpURL(raw)
	return ok
}

// IsLocal reports whether raw is an absolute filesystem path or file:// URL.
func IsLocal(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	if strings.HasPrefix(strings.ToLower(raw), "file://") {
		return true
	}
	return strings.HasPrefix(raw, "/")
}

// Display is the human label: local paths without the file:// prefix.
func Display(raw string) string {
	raw = strings.TrimSpace(raw)
	if IsLocal(raw) {
		p, err := LocalPath(raw)
		if err == nil {
			return p
		}
	}
	return raw
}

// LocalPath returns the cleaned absolute path for a local remote.
func LocalPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToLower(raw), "file://") {
		u, err := url.Parse(raw)
		if err != nil || u.Path == "" {
			return "", ErrInvalidRemote
		}
		return filepath.Clean(u.Path), nil
	}
	if strings.HasPrefix(raw, "/") {
		return filepath.Clean(raw), nil
	}
	return "", ErrInvalidRemote
}

// TransportURL is the URL used to clone: SSH form kept, file:// for local, otherwise Canonical HTTP.
func TransportURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if IsLocal(raw) {
		return Canonical(raw)
	}
	if IsSSH(raw) {
		if strings.HasPrefix(strings.ToLower(raw), "ssh://") {
			return strings.TrimSuffix(strings.TrimRight(raw, "/"), ".git")
		}
		// normalize scp: git@host:path
		at := strings.IndexByte(raw, '@')
		colon := strings.LastIndexByte(raw, ':')
		if at > 0 && colon > at {
			host := raw[at+1 : colon]
			path := strings.TrimSuffix(strings.TrimRight(raw[colon+1:], "/"), ".git")
			user := raw[:at]
			return user + "@" + host + ":" + path
		}
		return raw
	}
	return Canonical(raw)
}

// Validate reports whether raw is an SSH git remote or a local path.
func Validate(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return ErrEmptyRemote
	}
	if IsLocal(raw) {
		p, err := LocalPath(raw)
		if err != nil || p == "/" || p == "." {
			return ErrInvalidRemote
		}
		return nil
	}
	if !IsSSH(raw) {
		return ErrNotSSH
	}
	c := Canonical(raw)
	if c == "" {
		return ErrInvalidRemote
	}
	u, err := url.Parse(c)
	if err != nil || u.Host == "" || u.Path == "" || u.Path == "/" {
		return ErrInvalidRemote
	}
	return nil
}
