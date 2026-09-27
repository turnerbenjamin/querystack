// Package queryModel contains models used throughout the query package and by
// consuming packages
//
// This file contains the interfaces that allow the query package to work with
// the database schema without reflection
package queryModel

// relationshipType represents different table relationships
type RelationshipType string

const (
	// RelationshipOneToMany represents a 1:N relationship
	RelationshipOneToMany RelationshipType = "1:N"

	// RelationshipManyToOne represents an N:1 relationship
	RelationshipManyToOne RelationshipType = "N:1"
)

// Schema provides access to metadata for tables in a schema.
type Schema interface {

	// GetTableMetadata returns the metadata for the specified table and whether
	// the table was found.
	GetResource(tableName string) (Resource, bool)
}

// Resource represents a specific resource in the schema
type Resource interface {
	// GetMetadata returns TableMetadata for the resource
	GetMetadata() TableMetadata

	// InitProjection initialises an empty projection for the resource
	InitProjection() Projection

	// SliceFromJSON unmarshals a json array of records for the resource and
	// returns a slice. The unmarshalled records are bound to a projection which
	// can be used to control the data projected when marshalling those records
	SliceFromJSON(
		jsonData []byte,
		projectionNode *ProjectionNode,
	) ([]TableModel, error)
}

// TableModel is a representation of a specific database table
type TableModel interface {
	// GetValueExpression returns the value stored at a given path as a value
	// expression. This is used to extract values from results for the purposes
	// of cursor pagination
	GetValue(
		valueBuilder ValueBuilder,
		path []*TraversalStep,
		columnName string,
	) (Value, error)
}

// Projection represents the columns that should be projected when marshalling
// a given table model
type Projection interface {
	// Add is used to add a column to the projection, it returns an error if the
	// column does not exist on the resource
	Add(columnName string) error

	// Is empty returns true if the projection has zero columns
	IsEmpty() bool
}

// Projection node represents a projection against a specific resource in a
// query - It is designed to allow projections to be applied to nested resources
type ProjectionNode struct {
	// Projection is the projection for the current resource
	Projection Projection

	// Children is a collection of projection nodes for expanded relationships
	// on the table. It is keyed by the relevant expansion table on the resource
	Children map[string]*ProjectionNode
}

// ColumnMetadata provides data for a specific column in a table
type ColumnMetadata struct {
	// Name is the name of the column as it appears in the database
	Name string

	// DbType is the normalised database type of the column
	Type DbType
}

// RelationshipMetadata provides data for a relationship from one table to
// another. Relationships are defined at the table level, so for each database
// relationship, there should be two RelationshipMetadata objects describing the
// relationship from the perspective of each table
type RelationshipMetadata struct {

	// Id is the unique name for the database relationship
	Id string

	// Type is the relationship type, e.g. N:1 or 1:N
	Type RelationshipType

	// ColumnName, for a N:1 relationship, will be the name of the relevant
	// foreign key column, as it appears on the database. For a 1:N
	// relationship, it will be a pseudo column
	ColumnName string

	// ExpansionColumnName is the name of the expansion column; this is a pseudo
	// column which can hold the results of an expansion of the relationship
	ExpansionColumnName string

	// From is the resource the relationship is defined on
	From Resource

	// To is the resource that the relationship links to
	To Resource

	// FromColumn is the joining column on the resource the relationship is
	// defined on
	FromColumn ColumnMetadata

	// ToColumn is the joining column on the resource that the relationship
	// links to
	ToColumn ColumnMetadata
}

// TableMetadata is metadata for a given table in the database
type TableMetadata struct {
	// Name is the name of the table as it appears in the database
	Name string

	// FullyQualifiedName is the fully qualified name of the table, including
	// the schema name
	FullyQualifiedName string

	// PrimaryKeyColumn is the ColumnMetadata for the primary key column
	PrimaryKeyColumn ColumnMetadata

	// Columns represents all of the columns defined on the table, it is keyed
	// by the name of the column
	Columns map[string]ColumnMetadata

	// Relationships represents all of the database relationships that include
	// the table. It is keyed by the column name, as defined in the relationship
	// metadata
	Relationships map[string]RelationshipMetadata
}
