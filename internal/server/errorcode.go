package server

import "errors"

// Sentinel errors for GraphQL ErrorCode unmarshaling (errors.Is).
var (
	ErrEnumsMustBeStrings = errors.New("enums must be strings")
	ErrInvalidErrorCode   = errors.New("not a valid ErrorCode")
)
