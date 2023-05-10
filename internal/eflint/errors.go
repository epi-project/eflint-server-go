package eflint

import "errors"

// ErrUnsupportedVersion is returned when the input version is not supported.
var ErrUnsupportedVersion = errors.New("unsupported version")

// ErrUnsupportedFields is returned when the input contains fields that are not
// expected.
var ErrUnsupportedFields = errors.New("unsupported fields for this kind")

// ErrUnknownKind is returned when an unknown kind is provided.
var ErrUnknownKind = errors.New("unknown kind")
