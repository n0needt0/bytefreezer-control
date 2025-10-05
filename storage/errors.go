package storage

import "errors"

var (
	// ErrNotFound indicates the requested resource was not found
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists indicates the resource already exists
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrInvalidInput indicates invalid input parameters
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnsupportedStorageType indicates an unsupported storage backend
	ErrUnsupportedStorageType = errors.New("unsupported storage type")

	// ErrStorageNotAvailable indicates the storage backend is not available
	ErrStorageNotAvailable = errors.New("storage backend not available")

	// ErrDuplicateEmail indicates email already exists
	ErrDuplicateEmail = errors.New("email already exists")

	// ErrInvalidTenantID indicates invalid tenant ID
	ErrInvalidTenantID = errors.New("invalid tenant ID")

	// ErrInvalidDatasetID indicates invalid dataset ID
	ErrInvalidDatasetID = errors.New("invalid dataset ID")
)