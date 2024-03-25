package models

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrDBNotConnected  = errors.New("database not connected")
	ErrConflict        = errors.New("data conflict")
	ErrInvalidDataType = errors.New("invalid data type")
)
