// Package queryError is used to return errors with specific categories from the
// query package. Messages for all error categories other than
// queryErrInternalErr and queryErrUnknown are intended for display to users
package queryError

import (
	"errors"
	"fmt"
)

// queryErrorCategory represents a specific category of error returned from the
// query package
type queryErrorCategory uint8

const (
	// QueryErrUnknown is used when the query category cannot be determined
	QueryErrUnknown queryErrorCategory = iota

	// QueryErrSyntaxErr is used to represent bad query strings. Error messages
	// are designed for display to users to help them correct the error
	QueryErrSyntaxErr

	// QueryErrBindingErr is used to represent faiures to bind metadata to the
	// query, generally due to incorrect identifiers. Error messages are
	// designed for display to users to help them correct the error
	QueryErrBindingErr

	// QueryErrAccessErr is used to represent access requests that are
	// incompatible with the access policy
	QueryErrAccessErr

	// QueryErrInvalidPagingTokenErr is used to represent errors due to invalid
	// paging tokens
	QueryErrInvalidPagingTokenErr

	// QueryErrInternalErr is used to represent internal failures. Error
	// messages are designed to help debugging and are not intended for display
	// to users
	QueryErrInternalErr
)

// queryError is used to return an error with a category. This may be used to
// determine whether to surface the specifc error to the user or not
type queryError struct {
	category queryErrorCategory
	err      error
}

// GetErrorCategory returns the queryErrorCategory for a given error. If the
// error is nil or a type other than queryError it will return QueryErrUnknown
func GetErrorCategory(err error) queryErrorCategory {
	if err == nil {
		return QueryErrUnknown
	}

	qerr, success := errors.AsType[queryError](err)
	if success {
		return qerr.category
	}

	return QueryErrUnknown
}

// Error returns the error message
func (e queryError) Error() string {
	return e.err.Error()
}

// SyntaxErr builds a queryError with the category QueryErrSyntaxErr
func SyntaxErr(m string, a ...any) error {
	return queryError{
		category: QueryErrSyntaxErr,
		err:      fmt.Errorf(m, a...),
	}
}

// BindingErr builds a queryError with the category QueryErrBindingErr
func BindingErr(m string, a ...any) error {
	return queryError{
		category: QueryErrBindingErr,
		err:      fmt.Errorf(m, a...),
	}
}

// AccessErr builds a queryError with the category QueryErrAccessErr
func AccessErr(m string, a ...any) error {
	return queryError{
		category: QueryErrAccessErr,
		err:      fmt.Errorf(m, a...),
	}
}

// PagingTokenErr builds a queryError with the category
// QueryErrInvalidPagingTokenErr
func PagingTokenErr(m string, a ...any) error {
	return queryError{
		category: QueryErrInvalidPagingTokenErr,
		err:      fmt.Errorf(m, a...),
	}
}

// InternalErr builds a queryError with the category QueryErrInternalErr
func InternalErr(m string, a ...any) error {
	return queryError{
		category: QueryErrInternalErr,
		err:      fmt.Errorf(m, a...),
	}
}
