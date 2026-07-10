package apierrors

// Code is a stable, machine-readable error identifier for the frontend.
// Values must match the GraphQL ErrorCode enum and frontend error catalogs.
type Code string

const (
	CodeUnknown            Code = "UNKNOWN"
	CodeUnauthenticated    Code = "UNAUTHENTICATED"
	CodeUnauthorized       Code = "UNAUTHORIZED"
	CodeEmailTaken         Code = "EMAIL_TAKEN"
	CodeInvalidCredentials Code = "INVALID_CREDENTIALS"
	CodePasswordTooShort   Code = "PASSWORD_TOO_SHORT"
	CodeNotFound           Code = "NOT_FOUND"
	CodeProjectNotFound    Code = "PROJECT_NOT_FOUND"
	CodeFileNotFound       Code = "FILE_NOT_FOUND"
	CodeInvalidInput       Code = "INVALID_INPUT"
	CodeInternal           Code = "INTERNAL"
	CodeCompileFailed      Code = "COMPILE_FAILED"
	CodeCompileToolMissing Code = "COMPILE_TOOL_MISSING"
)

// All is the set of public codes (for docs/tests).
var All = []Code{
	CodeUnknown,
	CodeUnauthenticated,
	CodeUnauthorized,
	CodeEmailTaken,
	CodeInvalidCredentials,
	CodePasswordTooShort,
	CodeNotFound,
	CodeProjectNotFound,
	CodeFileNotFound,
	CodeInvalidInput,
	CodeInternal,
	CodeCompileFailed,
	CodeCompileToolMissing,
}
