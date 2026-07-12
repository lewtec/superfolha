package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueViolation(t *testing.T) {
	repo := &Repository{}

	unique := &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
	// 23503 = foreign_key_violation; same SQLSTATE class, not unique.
	fk := &pgconn.PgError{Code: "23503", Message: "foreign key violation"}
	// 23502 = not_null_violation
	notNull := &pgconn.PgError{Code: "23502", Message: "null value in column violates not-null constraint"}
	// 42P01 = undefined_table (unrelated class)
	other := &pgconn.PgError{Code: "42P01", Message: "relation does not exist"}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "plain error", err: errors.New("duplicate key value violates unique constraint"), want: false},
		{name: "wrapped plain error", err: fmt.Errorf("create user: %w", errors.New("duplicate key")), want: false},
		{name: "pg unique 23505", err: unique, want: true},
		{name: "wrapped pg unique", err: fmt.Errorf("create user: %w", unique), want: true},
		{name: "double-wrapped pg unique", err: fmt.Errorf("auth: %w", fmt.Errorf("create user: %w", unique)), want: true},
		{name: "pg foreign key 23503", err: fk, want: false},
		{name: "wrapped pg foreign key", err: fmt.Errorf("wrap: %w", fk), want: false},
		{name: "pg not null 23502", err: notNull, want: false},
		{name: "pg other code", err: other, want: false},
		{name: "empty code pg error", err: &pgconn.PgError{Code: "", Message: "x"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repo.IsUniqueViolation(tt.err)
			if got != tt.want {
				t.Fatalf("IsUniqueViolation(%v)=%v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
