package errors

import "errors"

// Domain errors
var (
	// ErrNotFound indicates a resource was not found
	// This error is used for both "not found" and "forbidden" cases to prevent enumeration attacks
	ErrNotFound = errors.New("resource not found")

	// ErrInvalidInput indicates invalid input data
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized indicates unauthorized access
	ErrUnauthorized = errors.New("unauthorized")

	// ErrInvalidTenant indicates a resource does not belong to the specified tenant
	// This is an internal error that should be mapped to ErrNotFound in handlers
	ErrInvalidTenant = errors.New("resource does not belong to tenant")

	// ErrProjectNotInTenant indicates a project does not belong to the specified tenant
	// This is an internal error that should be mapped to ErrNotFound in handlers
	ErrProjectNotInTenant = errors.New("project does not belong to tenant")
)

// IsNotFoundError checks if an error should be returned as a 404 Not Found response
// This includes actual not-found errors and forbidden errors to prevent enumeration
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrInvalidTenant) ||
		errors.Is(err, ErrProjectNotInTenant)
}
