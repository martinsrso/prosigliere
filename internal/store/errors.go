package store

import "errors"

var (
	ErrPostNotFound = errors.New("post not found")
	ErrInvalidInput = errors.New("invalid input")
)
