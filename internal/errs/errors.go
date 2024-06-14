// Package errs contains common error constants.
package errs

import "errors"

// ErrNotFound is returned when a requested resource is not found.
var ErrNotFound = errors.New("not found")

// ErrUnauthorized is returned when the client is not authorized
// to perform the requested operation.
var ErrUnauthorized = errors.New("unauthorized")

// ErrConflict is returned when the requested operation
// would result in a conflict with existing data.
var ErrConflict = errors.New("data conflict")

// ErrInvalidRequest is returned when the request is invalid or incomplete.
var ErrInvalidRequest = errors.New("invalid request")

// ErrDBNotConnected is returned when the database connection is not established.
var ErrDBNotConnected = errors.New("database not connected")

// ErrNilDependency indicates unproper initialization.
var ErrNilDependency = errors.New("nil dependency")
