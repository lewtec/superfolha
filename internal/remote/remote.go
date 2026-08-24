// Package remote canonicalizes git HTTP URLs and parses GitHub owner/repo pairs.
package remote

import (
	"errors"
	"net/url"
	"strings"
)

var (
	ErrEmptyRemote   = errors.New("empty remote")
	ErrInvalidRemote = errors.New("invalid remote")
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

// Validate reports whether raw can be used as a clone URL.
func Validate(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return ErrEmptyRemote
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
