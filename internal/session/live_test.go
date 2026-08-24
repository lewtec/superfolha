package session

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	igit "github.com/lewtec/superfolha/internal/git"
	"github.com/lewtec/superfolha/internal/project"
)

func stubClone(dest, _, _ string, _ *igit.HTTPAuth, _ *igit.SSHKey) error {
	if err := igit.InitRepo(dest); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dest, "main.tex"), []byte("body\n"), 0o644)
}

func TestCreateDuplicateFails(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-for-session")
	t.Setenv("GO_ENV", "development")
	reg := NewRegistry(project.NewService(t.TempDir()))
	reg.SetCloner(stubClone)

	a, err := reg.Create("alice", "https://github.com/t/paper", "main", nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == "" {
		t.Fatal("missing session id")
	}
	again, err := reg.Create("alice", "https://github.com/t/paper.git", "main", nil)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != a.ID {
		t.Fatalf("host retry id = %s; want %s", again.ID, a.ID)
	}
	_, err = reg.Create("bob", "https://github.com/t/paper", "main", nil)
	if !errors.Is(err, ErrAlreadyLive) {
		t.Fatalf("bob create: %v; want ErrAlreadyLive", err)
	}
}

func TestKnockClosedAndPreauth(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-for-session")
	t.Setenv("GO_ENV", "development")
	reg := NewRegistry(project.NewService(t.TempDir()))
	reg.SetCloner(stubClone)
	live, err := reg.Create("alice", "https://github.com/t/paper", "main", nil)
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
	reg.SetCloner(stubClone)
	live, err := reg.Create("alice", "https://github.com/t/paper", "main", nil)
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
