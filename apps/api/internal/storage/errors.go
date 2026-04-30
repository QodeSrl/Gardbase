package storage

import "errors"

var (
	ErrNotFound          = errors.New("not found")
	ErrTableNotFound     = errors.New("table not found")
	ErrNotFoundOrDeleted = errors.New("not found or already deleted")
	ErrAlreadyExists     = errors.New("already exists")
	ErrVersionMismatch   = errors.New("version mismatch")
)
