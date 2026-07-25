package apierrors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestErrorStatus_AllCodes(t *testing.T) {
	t.Parallel()

	wantByCode := map[Code]int{
		CodeUnknown:            http.StatusInternalServerError,
		CodeUnauthenticated:    http.StatusUnauthorized,
		CodeUnauthorized:       http.StatusForbidden,
		CodeEmailTaken:         http.StatusConflict,
		CodeInvalidCredentials: http.StatusBadRequest,
		CodePasswordTooShort:   http.StatusBadRequest,
		CodeNotFound:           http.StatusNotFound,
		CodeProjectNotFound:    http.StatusNotFound,
		CodeFileNotFound:       http.StatusNotFound,
		CodeInvalidInput:       http.StatusBadRequest,
		CodeInternal:           http.StatusInternalServerError,
		CodeCompileFailed:      http.StatusUnprocessableEntity,
		CodeCompileToolMissing: http.StatusInternalServerError,
	}

	if len(wantByCode) != len(All) {
		t.Fatalf("wantByCode has %d entries, All has %d — update Status mapping test", len(wantByCode), len(All))
	}

	for _, code := range All {
		code := code
		t.Run(string(code), func(t *testing.T) {
			t.Parallel()
			want, ok := wantByCode[code]
			if !ok {
				t.Fatalf("code %q in All but missing from wantByCode", code)
			}
			err := New(code, "msg")
			if got := err.Status(); got != want {
				t.Errorf("New(%q).Status() = %d, want %d", code, got, want)
			}
		})
	}
}

func TestErrorStatus_WithStatusOverride(t *testing.T) {
	t.Parallel()

	// Explicit HTTPStatus wins over Code-based mapping (e.g. 401 vs default).
	err := WithStatus(CodeInvalidInput, "custom", http.StatusTeapot)
	if got := err.Status(); got != http.StatusTeapot {
		t.Errorf("WithStatus override Status() = %d, want %d", got, http.StatusTeapot)
	}

	// Zero HTTPStatus falls through to Code mapping.
	plain := New(CodeUnauthenticated, "need login")
	if got := plain.Status(); got != http.StatusUnauthorized {
		t.Errorf("New(CodeUnauthenticated).Status() = %d, want %d", got, http.StatusUnauthorized)
	}
}

func TestErrorStatus_UnknownCodeDefaultsToBadRequest(t *testing.T) {
	t.Parallel()

	err := New(Code("NOT_A_REAL_CODE"), "x")
	if got := err.Status(); got != http.StatusBadRequest {
		t.Errorf("unknown Code Status() = %d, want %d", got, http.StatusBadRequest)
	}
}

func TestAsAndCodeOf(t *testing.T) {
	t.Parallel()

	t.Run("As coded error", func(t *testing.T) {
		t.Parallel()
		inner := New(CodeNotFound, "missing")
		wrapped := fmt.Errorf("wrap: %w", inner)

		got, ok := As(wrapped)
		if !ok {
			t.Fatal("As(wrapped) = false, want true")
		}
		if got.Code != CodeNotFound {
			t.Errorf("As code = %q, want %q", got.Code, CodeNotFound)
		}
		if CodeOf(wrapped) != CodeNotFound {
			t.Errorf("CodeOf(wrapped) = %q, want %q", CodeOf(wrapped), CodeNotFound)
		}
	})

	t.Run("As plain error", func(t *testing.T) {
		t.Parallel()
		plain := errors.New("plain")
		if _, ok := As(plain); ok {
			t.Fatal("As(plain) = true, want false")
		}
		if CodeOf(plain) != CodeUnknown {
			t.Errorf("CodeOf(plain) = %q, want %q", CodeOf(plain), CodeUnknown)
		}
	})

	t.Run("CodeOf nil", func(t *testing.T) {
		t.Parallel()
		if CodeOf(nil) != CodeUnknown {
			t.Errorf("CodeOf(nil) = %q, want %q", CodeOf(nil), CodeUnknown)
		}
	})
}

func TestErrorMessageAndRESTBody(t *testing.T) {
	t.Parallel()

	withMsg := New(CodeInternal, "boom")
	if withMsg.Error() != "boom" {
		t.Errorf("Error() = %q, want boom", withMsg.Error())
	}

	noMsg := New(CodeInternal, "")
	if noMsg.Error() != string(CodeInternal) {
		t.Errorf("Error() empty message = %q, want code string", noMsg.Error())
	}

	body := withMsg.RESTBody()
	if body.Code != string(CodeInternal) || body.Message != "boom" {
		t.Errorf("RESTBody() = %+v, want code=%s message=boom", body, CodeInternal)
	}
}

func TestInternalHelper(t *testing.T) {
	t.Parallel()

	nilErr := Internal(nil)
	if nilErr.Code != CodeInternal {
		t.Errorf("Internal(nil).Code = %q, want %q", nilErr.Code, CodeInternal)
	}
	if nilErr.Status() != http.StatusInternalServerError {
		t.Errorf("Internal(nil).Status() = %d, want 500", nilErr.Status())
	}
	if nilErr.Message != "internal error" {
		t.Errorf("Internal(nil).Message = %q, want %q", nilErr.Message, "internal error")
	}

	cause := errors.New("db down: secret path /var/lib/superfolha.db")
	wrapped := Internal(cause)
	if !errors.Is(wrapped, cause) {
		t.Errorf("Internal(cause) should wrap cause; Unwrap chain missing")
	}
	if CodeOf(wrapped) != CodeInternal {
		t.Errorf("CodeOf(Internal(cause)) = %q, want %q", CodeOf(wrapped), CodeInternal)
	}
	// Client-facing Message must not embed the underlying error text.
	if wrapped.Message != "internal error" {
		t.Errorf("Internal(cause).Message = %q, want generic %q", wrapped.Message, "internal error")
	}
	body := wrapped.RESTBody()
	if body.Message != "internal error" {
		t.Errorf("RESTBody().Message = %q, want generic internal error", body.Message)
	}
	if body.Code != string(CodeInternal) {
		t.Errorf("RESTBody().Code = %q, want %s", body.Code, CodeInternal)
	}
	// Error() keeps the cause for operator logs.
	if got := wrapped.Error(); got != "internal error: "+cause.Error() {
		t.Errorf("Error() = %q, want cause appended for logs", got)
	}
}

func TestErrorString_DoesNotDoubleCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("disk full")
	// Call sites that already put err.Error() in Message should not double-append.
	w := Wrap(CodeCompileFailed, cause.Error(), cause)
	if got := w.Error(); got != cause.Error() {
		t.Errorf("Error() = %q, want %q (no double append)", got, cause.Error())
	}
}
