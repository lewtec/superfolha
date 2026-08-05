package server

import (
	"errors"
	"os"

	"github.com/lewtec/superfolha/internal/apierrors"
	"github.com/lewtec/superfolha/internal/project"
)

// mapPathAPIError maps path-jail and missing-file sentinels to client API errors.
// Returns nil when err is not one of those sentinels (caller should handle further).
func mapPathAPIError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, project.ErrInvalidPath):
		return apierrors.New(apierrors.CodeInvalidInput, "invalid file path")
	case errors.Is(err, project.ErrFileNotFound), errors.Is(err, os.ErrNotExist):
		return apierrors.New(apierrors.CodeFileNotFound, "file not found")
	default:
		return nil
	}
}
