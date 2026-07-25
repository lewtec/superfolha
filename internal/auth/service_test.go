package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/lewtec/superfolha/internal/apierrors"
	"github.com/lewtec/superfolha/internal/db"
)

// memRepo is a minimal in-memory db.Repository for auth Service tests.
type memRepo struct {
	byEmail map[string]db.User
}

func newMemRepo(users ...db.User) *memRepo {
	m := &memRepo{byEmail: make(map[string]db.User)}
	for _, u := range users {
		m.byEmail[u.Email] = u
	}
	return m
}

func (m *memRepo) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
	if _, ok := m.byEmail[arg.Email]; ok {
		return db.User{}, errors.New("unique")
	}
	u := db.User{ID: arg.ID, Email: arg.Email, PasswordHash: arg.PasswordHash}
	m.byEmail[arg.Email] = u
	return u, nil
}

func (m *memRepo) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return db.User{}, errors.New("not found")
	}
	return u, nil
}

func (m *memRepo) GetUserByID(ctx context.Context, id string) (db.User, error) {
	for _, u := range m.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return db.User{}, errors.New("not found")
}

func (m *memRepo) CreateProject(ctx context.Context, arg db.CreateProjectParams) (db.Project, error) {
	return db.Project{}, errors.New("not implemented")
}
func (m *memRepo) GetProject(ctx context.Context, id string) (db.Project, error) {
	return db.Project{}, errors.New("not implemented")
}
func (m *memRepo) GetUserProjects(ctx context.Context, userID string) ([]db.Project, error) {
	return nil, errors.New("not implemented")
}
func (m *memRepo) UpdateProjectTimestamp(ctx context.Context, id string) error {
	return errors.New("not implemented")
}
func (m *memRepo) DeleteProject(ctx context.Context, id string) error {
	return errors.New("not implemented")
}
func (m *memRepo) IsUniqueViolation(err error) bool {
	return err != nil && err.Error() == "unique"
}
func (m *memRepo) Close() error { return nil }

func TestLogin_UnknownEmailSameErrorAsWrongPassword(t *testing.T) {
	withJWTEnv(t, "test-secret-for-login-service", "")

	hash, err := HashPassword("correct-horse-battery")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	svc := NewService(newMemRepo(db.User{
		ID: "user-1", Email: "known@example.com", PasswordHash: hash,
	}))
	ctx := context.Background()

	_, errUnknown := svc.Login(ctx, "missing@example.com", "correct-horse-battery")
	_, errWrong := svc.Login(ctx, "known@example.com", "wrong-password-xx")

	if errUnknown == nil || errWrong == nil {
		t.Fatal("expected both logins to fail")
	}
	if apierrors.CodeOf(errUnknown) != apierrors.CodeInvalidCredentials {
		t.Fatalf("unknown email code = %v, want INVALID_CREDENTIALS", apierrors.CodeOf(errUnknown))
	}
	if apierrors.CodeOf(errWrong) != apierrors.CodeInvalidCredentials {
		t.Fatalf("wrong password code = %v, want INVALID_CREDENTIALS", apierrors.CodeOf(errWrong))
	}
	if errUnknown.Error() != errWrong.Error() {
		t.Fatalf("client messages differ: unknown=%q wrong=%q", errUnknown.Error(), errWrong.Error())
	}
}

func TestLogin_Success(t *testing.T) {
	withJWTEnv(t, "test-secret-for-login-service", "")

	const pass = "correct-horse-battery"
	hash, err := HashPassword(pass)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	svc := NewService(newMemRepo(db.User{
		ID: "user-1", Email: "known@example.com", PasswordHash: hash,
	}))

	resp, err := svc.Login(context.Background(), "Known@example.com", pass)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if resp.User == nil || resp.User.ID != "user-1" {
		t.Fatalf("unexpected user: %+v", resp.User)
	}
	if resp.Token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestDummyBcryptHashIsValid(t *testing.T) {
	// Ensures package init produced a hash CheckPasswordHash will actually verify against.
	if dummyBcryptHash == "" {
		t.Fatal("dummyBcryptHash empty")
	}
	// Wrong password must return false without panic / empty-hash short-circuit.
	if CheckPasswordHash("not-the-dummy", dummyBcryptHash) {
		t.Fatal("expected dummy hash not to match arbitrary password")
	}
}
