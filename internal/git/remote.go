package git

import (
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// HTTPAuth is RAM-only clone/push credentials. Do not log Password.
type HTTPAuth struct {
	Username string
	Password string
}

func (a *HTTPAuth) method() transport.AuthMethod {
	if a == nil || (a.Username == "" && a.Password == "") {
		return nil
	}
	user := a.Username
	if user == "" {
		user = "x-access-token"
	}
	return &http.BasicAuth{Username: user, Password: a.Password}
}

// CloneHTTP clones remote at branch into dest. Empty branch uses the remote HEAD.
func CloneHTTP(dest, remoteURL, branch string, auth *HTTPAuth) error {
	return authClone{dest: dest, remoteURL: remoteURL, branch: branch, auth: auth.method()}.run()
}

// PushOrigin pushes the current HEAD branch to origin over HTTP.
func PushOrigin(repoPath, branch string, auth *HTTPAuth) error {
	return pushOriginWithAuth(repoPath, branch, auth.method())
}
