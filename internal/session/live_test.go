package session

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	igit "github.com/lewtec/superfolha/internal/git"
	"github.com/lewtec/superfolha/internal/project"
)

func stubClone(dest, _, _ string, _ igit.SessionSSH) error {
	if err := igit.InitRepo(dest); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dest, "main.tex"), []byte("body\n"), 0o644)
}

func stubProbe(_ string, _ string, _ igit.SessionSSH) error { return nil }

func testKey(t *testing.T) *igit.SSHKey {
	t.Helper()
	k, err := igit.NewSessionSSHKey()
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestCreateDuplicateFails(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-for-session")
	t.Setenv("GO_ENV", "development")
	reg := NewRegistry(project.NewService(t.TempDir()))
	k := testKey(t)

	a, err := reg.Create("alice", "git@github.com:t/paper", "main", k.Authorized)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == "" {
		t.Fatal("missing session id")
	}
	if a.Ready {
		t.Fatal("create must stay pending")
	}
	again, err := reg.Create("alice", "git@github.com:t/paper.git", "main", k.Authorized)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != a.ID {
		t.Fatalf("host retry id = %s; want %s", again.ID, a.ID)
	}
	_, err = reg.Create("bob", "git@github.com:t/paper", "main", k.Authorized)
	if !errors.Is(err, ErrAlreadyLive) {
		t.Fatalf("bob create: %v; want ErrAlreadyLive", err)
	}
}

func TestSSHCreateWaitsForDeployKey(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-for-session")
	t.Setenv("GO_ENV", "development")
	cloned := false
	reg := NewRegistry(project.NewService(t.TempDir()))
	reg.SetCloner(func(dest, _, _ string, _ igit.SessionSSH) error {
		cloned = true
		return stubClone(dest, "", "", nil)
	})
	k := testKey(t)
	live, err := reg.Create("alice", "git@github.com:t/paper", "main", k.Authorized)
	if err != nil {
		t.Fatal(err)
	}
	if cloned {
		t.Fatal("create must not clone")
	}
	if live.Ready {
		t.Fatal("SSH session should stay pending")
	}
	if live.SSHPublic == "" {
		t.Fatal("expected session public key")
	}
}

func TestKnockClosedAndPreauth(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-for-session")
	t.Setenv("GO_ENV", "development")
	reg := NewRegistry(project.NewService(t.TempDir()))
	k := testKey(t)
	live, err := reg.Create("alice", "git@github.com:t/paper", "main", k.Authorized)
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.Knock(live.ID, "bob"); !errors.Is(err, ErrKnockClosed) {
		t.Fatalf("knock off: %v", err)
	}
	if reg.CanOpen(live.ID, "bob") {
		t.Fatal("bob must not open")
	}
	tok, err := MintPreauth(live.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.RedeemPreauth(live.ID, "bob", tok); err != nil {
		t.Fatal(err)
	}
	if !reg.CanOpen(live.ID, "bob") {
		t.Fatal("bob should be admitted")
	}
}

func TestKnockOnAndAdmit(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-for-session")
	t.Setenv("GO_ENV", "development")
	reg := NewRegistry(project.NewService(t.TempDir()))
	k := testKey(t)
	live, err := reg.Create("alice", "git@github.com:t/paper", "main", k.Authorized)
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.SetKnock(live.ID, "alice", true); err != nil {
		t.Fatal(err)
	}
	if err := reg.Knock(live.ID, "carol"); err != nil {
		t.Fatal(err)
	}
	if reg.CanOpen(live.ID, "carol") {
		t.Fatal("knocking is not admit")
	}
	if err := reg.Admit(live.ID, "alice", "carol"); err != nil {
		t.Fatal(err)
	}
	if !reg.CanOpen(live.ID, "carol") {
		t.Fatal("carol should be admitted")
	}
}

func TestCreateSeedsMainTeX(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-for-session")
	t.Setenv("GO_ENV", "development")
	svc := project.NewService(t.TempDir())
	reg := NewRegistry(svc)
	reg.SetCloner(func(dest, _, _ string, _ igit.SessionSSH) error {
		return igit.InitRepo(dest)
	})
	reg.SetProber(stubProbe)
	k := testKey(t)
	live, err := reg.Create("alice", "git@github.com:t/empty", "main", k.Authorized)
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.CloneAndProbe(live.ID, "alice", k); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(svc.GetProjectPath(live.ID), "main.tex"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte(`\documentclass`)) {
		t.Fatalf("seeded main.tex = %q", body)
	}
}

func TestCreateStoresPublic(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-for-session")
	t.Setenv("GO_ENV", "development")
	k := testKey(t)
	reg := NewRegistry(project.NewService(t.TempDir()))
	live, err := reg.Create("alice", "git@github.com:t/paper", "main", k.Authorized)
	if err != nil {
		t.Fatal(err)
	}
	if !SamePublic(live.SSHPublic, k.Authorized) {
		t.Fatalf("SSHPublic = %q; want %q", live.SSHPublic, k.Authorized)
	}
}

func TestCreateRejectsHTTP(t *testing.T) {
	reg := NewRegistry(project.NewService(t.TempDir()))
	k := testKey(t)
	_, err := reg.Create("alice", "https://github.com/t/paper", "main", k.Authorized)
	if err == nil {
		t.Fatal("https remote must fail")
	}
}

func TestCloneAndProbeRejectsWrongKey(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-for-session")
	t.Setenv("GO_ENV", "development")
	reg := NewRegistry(project.NewService(t.TempDir()))
	reg.SetCloner(stubClone)
	reg.SetProber(stubProbe)
	k := testKey(t)
	other := testKey(t)
	live, err := reg.Create("alice", "git@github.com:t/paper", "main", k.Authorized)
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.CloneAndProbe(live.ID, "alice", other); !errors.Is(err, ErrNoSigner) {
		t.Fatalf("wrong key: %v", err)
	}
}
