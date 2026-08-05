package server

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/lewtec/superfolha/internal/apierrors"
	"github.com/lewtec/superfolha/internal/project"
)

func TestMapPathAPIError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		code    apierrors.Code
		wantNil bool
	}{
		{name: "nil", err: nil, wantNil: true},
		{name: "other", err: errors.New("disk full"), wantNil: true},
		{name: "invalid path", err: project.ErrInvalidPath, code: apierrors.CodeInvalidInput},
		{name: "wrapped invalid", err: fmt.Errorf("%w: empty", project.ErrInvalidPath), code: apierrors.CodeInvalidInput},
		{name: "file not found", err: project.ErrFileNotFound, code: apierrors.CodeFileNotFound},
		{name: "os not exist", err: os.ErrNotExist, code: apierrors.CodeFileNotFound},
		{name: "wrapped not exist", err: fmt.Errorf("stat: %w", os.ErrNotExist), code: apierrors.CodeFileNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := mapPathAPIError(tt.err)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("mapPathAPIError(%v) = %v, want nil", tt.err, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("mapPathAPIError(%v) = nil, want code %s", tt.err, tt.code)
			}
			var ae *apierrors.Error
			if !errors.As(got, &ae) {
				t.Fatalf("mapPathAPIError(%v) type %T, want *apierrors.Error", tt.err, got)
			}
			if ae.Code != tt.code {
				t.Fatalf("code = %s, want %s", ae.Code, tt.code)
			}
		})
	}
}
