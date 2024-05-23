package errs

import "errors"

var (
	ErrNotFound       = errors.New("not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrConflict       = errors.New("data conflict")
	ErrInvalidRequest = errors.New("invalid request")
	ErrDBNotConnected = errors.New("database not connected")
)
