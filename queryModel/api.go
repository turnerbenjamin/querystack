// Package queryModel contains models used throughout the query package and by
// consuming packages
//
// This file contains models used in the API with consuming packages
package queryModel

import "context"

// ExecuteResult represents the type returned when Executing a Query
type ExecuteResult struct {
	Count         *uint64      `json:"count,omitempty"`
	NextPageToken string       `json:"next_page_token,omitempty"`
	Data          []TableModel `json:"data"`
}

// QueryStatement represents a query statement and the args for any placeholder
// values
type QueryStatement struct {
	// Statement is an sql query statement
	Statement string

	// Args is the concrete values for any placeholders in the statement
	Args []any
}

// Repository executes requests against the underlying data source.
type Repository interface {
	// ExecuteJsonRequest executes a query with the supplied arguments and returns
	// the response body.
	ExecuteJsonRequest(
		ctx context.Context,
		query QueryStatement,
	) ([]byte, error)

	// ExecuteJsonRequestWithCount executes a query and a count query using the
	// supplied shared arguments, returning the response body and total count.
	ExecuteJsonRequestWithCount(
		ctx context.Context,
		mainQuery QueryStatement,
		countQuery QueryStatement,
	) ([]byte, *uint64, error)
}

// QueryConfig is used to set configuration options on the query
// executor
type QueryConfig struct {
	// DefaultPageSize represents the default limit applied when no limit is
	// specified in the query string - This property defaults to 100
	DefaultPageSize uint32

	// MaxRecordsPerPage represents the maximum number of records, including
	// both top-level and nested records, returned in a given query - This
	// property defaults to 100_000
	MaxRecordsPerPage uint32

	// MaxDepth represents the maximum depth of expand statements - A depth of 1
	// would allow a single expand statement on the top-level query with no
	// nested expand statements - The default is 10
	MaxDepth uint8
}

// QueryConfigWithDefaults populates the queryBuilderConfig with default
// values
func QueryConfigWithDefaults(config QueryConfig) QueryConfig {
	if config.DefaultPageSize == 0 {
		config.DefaultPageSize = 100
	}

	if config.MaxRecordsPerPage == 0 {
		config.MaxRecordsPerPage = 100_000
	}

	if config.MaxDepth == 0 {
		config.MaxDepth = 10
	}
	return config
}
