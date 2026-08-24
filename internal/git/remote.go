package git

import (
	"fmt"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
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
	opts := &gogit.CloneOptions{
		URL:  remoteURL,
		Auth: auth.method(),
	}
	if branch != "" {
		opts.ReferenceName = plumbing.NewBranchReferenceName(branch)
		opts.SingleBranch = true
	}
	_, err := gogit.PlainClone(dest, false, opts)
	if err != nil {
		return fmt.Errorf("clone %s: %w", remoteURL, err)
	}
	return nil
}

// PushOrigin pushes the current HEAD branch to origin.
func PushOrigin(repoPath, branch string, auth *HTTPAuth) error {
	return Push(repoPath, branch, auth, nil)
}
