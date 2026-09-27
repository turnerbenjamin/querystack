// Package queryModel contains models used throughout the query package and by
// consuming packages
//
// This file contains models used to define an Access Policy for the schema
package queryModel

// AccessPolicy defines the access policy for a database
type AccessPolicy interface {
	// GetTableAccessPolicy returns the access policy for the specified table
	// and whether the table has an access policy.
	GetTableAccessPolicy(tableName string) (TableAccessPolicy, bool)
}

// TableAccessPolicy defines the query access policy for a given database table
type TableAccessPolicy interface {
	// CanAccess defines, at the table level, if a user can perform any
	// operations on that table. ColumnAccessPolicies can restrict access but
	// will override a false value returned from this method
	CanAccess() bool

	// GetColumnAccessPolicy returns whether a given column can be accessed, it
	// returns an error if the column cannot be found
	CanAccessColumn(columnName string) (bool, error)
}
