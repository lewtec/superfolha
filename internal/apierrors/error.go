package apierrors

import (
	"errors"
	"fmt"
	"net/http"
)

// Error is returned by services and presented to clients with a stable Code.
// Message is English for logs/playground only; UI maps Code via i18n.
type Error struct {
	Code    Code
	Message string
	// HTTPStatus is used by REST handlers; 0 means 400 default for coded errors.
	HTTPStatus int
	Err        error
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error { return e.Err }

func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

func Wrap(code Code, message string, err error) *Error {
	return &Error{Code: code, Message: message, Err: err}
}

func WithStatus(code Code, message string, status int) *Error {
	return &Error{Code: code, Message: message, HTTPStatus: status}
}

func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

func CodeOf(err error) Code {
	if e, ok := As(err); ok {
		return e.Code
	}
	return CodeUnknown
}

// Status returns a suitable HTTP status for REST responses.
func (e *Error) Status() int {
	if e.HTTPStatus != 0 {
		return e.HTTPStatus
	}
	switch e.Code {
	case CodeUnauthenticated:
		return http.StatusUnauthorized
	case CodeUnauthorized:
		return http.StatusForbidden
	case CodeNotFound, CodeProjectNotFound, CodeFileNotFound:
		return http.StatusNotFound
	case CodeEmailTaken:
		return http.StatusConflict
	case CodeInvalidCredentials, CodePasswordTooShort, CodeInvalidInput:
		return http.StatusBadRequest
	case CodeCompileToolMissing, CodeInternal, CodeUnknown:
		return http.StatusInternalServerError
	case CodeCompileFailed:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusBadRequest
	}
}

// RESTBody is the JSON body for non-GraphQL APIs.
type RESTBody struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
}

func (e *Error) RESTBody() RESTBody {
	return RESTBody{Code: string(e.Code), Message: e.Message}
}

func Internal(err error) *Error {
	if err == nil {
		return New(CodeInternal, "internal error")
	}
	return Wrap(CodeInternal, fmt.Sprintf("internal error: %v", err), err)
}
