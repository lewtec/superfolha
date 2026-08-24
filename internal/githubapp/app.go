// Package githubapp talks to GitHub App OAuth and installation-token APIs.
package githubapp

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lewtec/superfolha/internal/git"
)

var (
	ErrNotConfigured = errors.New("github app not configured")
	ErrNeedInstall   = errors.New("github app not installed on repository")
	ErrOAuth         = errors.New("github oauth failed")
	ErrInstallation  = errors.New("github installation")
)

// Config is deploy-time GitHub App credentials.
type Config struct {
	AppID        string
	ClientID     string
	ClientSecret string
	PrivateKey   *rsa.PrivateKey
	Slug         string
	APIBase      string // default https://api.github.com
	WebBase      string // default https://github.com
	HTTPClient   *http.Client
}

// Enabled reports whether user OAuth can run.
func (c Config) Enabled() bool {
	return c.ClientID != "" && c.ClientSecret != ""
}

func (c Config) apiBase() string {
	if c.APIBase != "" {
		return strings.TrimRight(c.APIBase, "/")
	}
	return "https://api.github.com"
}

func (c Config) webBase() string {
	if c.WebBase != "" {
		return strings.TrimRight(c.WebBase, "/")
	}
	return "https://github.com"
}

func (c Config) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

// AuthCodeURL is the GitHub authorize URL.
func (c Config) AuthCodeURL(state, redirectURI string) string {
	q := url.Values{
		"client_id":    {c.ClientID},
		"redirect_uri": {redirectURI},
		"state":        {state},
	}
	return c.webBase() + "/login/oauth/authorize?" + q.Encode()
}

// InstallURL sends the user to grant the App on repositories.
func (c Config) InstallURL() string {
	if c.Slug == "" {
		return c.webBase() + "/settings/apps"
	}
	return c.webBase() + "/apps/" + c.Slug + "/installations/new"
}

// Identity is a GitHub user after OAuth.
type Identity struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

// Exchange trades an OAuth code for identity.
func (c Config) Exchange(code, redirectURI string) (Identity, error) {
	var zero Identity
	if !c.Enabled() {
		return zero, ErrNotConfigured
	}
	form := url.Values{
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}
	req, err := http.NewRequest(http.MethodPost, c.webBase()+"/login/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return zero, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.client().Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return zero, err
	}
	var tok struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return zero, fmt.Errorf("%w: %s", ErrOAuth, body)
	}
	if tok.AccessToken == "" {
		return zero, fmt.Errorf("%w: %s", ErrOAuth, tok.Error)
	}
	ureq, err := http.NewRequest(http.MethodGet, c.apiBase()+"/user", nil)
	if err != nil {
		return zero, err
	}
	ureq.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	ureq.Header.Set("Accept", "application/vnd.github+json")
	uresp, err := c.client().Do(ureq)
	if err != nil {
		return zero, err
	}
	defer uresp.Body.Close()
	if uresp.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("%w: user %s", ErrOAuth, uresp.Status)
	}
	var id Identity
	if err := json.NewDecoder(uresp.Body).Decode(&id); err != nil {
		return zero, err
	}
	if id.Login == "" {
		return zero, fmt.Errorf("%w: empty login", ErrOAuth)
	}
	return id, nil
}

// InstallationAuth returns HTTP clone credentials for owner/repo.
func (c Config) InstallationAuth(owner, repo string) (*git.HTTPAuth, error) {
	if c.PrivateKey == nil || c.AppID == "" {
		return nil, ErrNotConfigured
	}
	appJWT, err := c.appJWT()
	if err != nil {
		return nil, err
	}
	instURL := fmt.Sprintf("%s/repos/%s/%s/installation", c.apiBase(), url.PathEscape(owner), url.PathEscape(repo))
	req, err := http.NewRequest(http.MethodGet, instURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNeedInstall
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: lookup %s", ErrInstallation, resp.Status)
	}
	var inst struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&inst); err != nil {
		return nil, err
	}
	tokURL := fmt.Sprintf("%s/app/installations/%d/access_tokens", c.apiBase(), inst.ID)
	treq, err := http.NewRequest(http.MethodPost, tokURL, strings.NewReader(`{}`))
	if err != nil {
		return nil, err
	}
	treq.Header.Set("Authorization", "Bearer "+appJWT)
	treq.Header.Set("Accept", "application/vnd.github+json")
	treq.Header.Set("Content-Type", "application/json")
	tresp, err := c.client().Do(treq)
	if err != nil {
		return nil, err
	}
	defer tresp.Body.Close()
	if tresp.StatusCode != http.StatusCreated && tresp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: token %s", ErrInstallation, tresp.Status)
	}
	var tok struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(tresp.Body).Decode(&tok); err != nil {
		return nil, err
	}
	if tok.Token == "" {
		return nil, fmt.Errorf("%w: empty token", ErrInstallation)
	}
	return &git.HTTPAuth{Username: "x-access-token", Password: tok.Token}, nil
}

func (c Config) appJWT() (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    c.AppID,
		IssuedAt:  jwt.NewNumericDate(now.Add(-30 * time.Second)),
		ExpiresAt: jwt.NewNumericDate(now.Add(9 * time.Minute)),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return tok.SignedString(c.PrivateKey)
}
