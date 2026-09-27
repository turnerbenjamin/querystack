// Package querymodel contains models used throughout the query package and by
// consuming packages
//
// This file contains models used to construct an AST when parsing the query
// string
package querymodel

import (
	"time"

	querybuilder "github.com/turnerbenjamin/querystack/querybuilder"
)

// DbType represents the normalized database type
type DbType string

const (
	// DbTypeString represents a string database value
	DbTypeString DbType = "string"

	// DbTypeInt represents an integer database value
	DbTypeInt DbType = "int"

	// DbTypeFloat represents a floating-point database value
	DbTypeFloat DbType = "float"

	// DbTypePoint represents a geographic point database value
	DbTypePoint DbType = "point"

	// DbTypeDateType represents a date and time database value
	DbTypeDateTime DbType = "date/time"
)

// ValueType represents a value for a comparison expression
type ValueType uint8

const (

	// ValueTypeNull indicates that no value type is available because the value
	// is null.
	ValueTypeNull ValueType = iota

	// ValueTypeString represents a string value
	ValueTypeString

	// ValueTypeInt represents an integer value
	ValueTypeInt

	// ValueTypeFloat represents a floating-point value
	ValueTypeFloat

	// ValueTypePoint represents a geographic point value
	ValueTypePoint

	// ValueTypeDateType represents a date and time value
	ValueTypeDateTime

	// ValueTypeIntList represents a list of integer values
	ValueTypeIntList

	// ValueTypeStringList represents a list of string values
	ValueTypeStringList

	// ValueTypeFloatList represents a list of floating-point values
	ValueTypeFloatList
)

// Serialiser is used to serialise values when constructing paging tokens
type Serialiser interface {

	// SerialiseNull serialises a null value
	SerialiseNull()

	// SerialiseString serialises a string value
	SerialiseString(str string)

	// SerialiseInt serialises an integer value
	SerialiseInt(n int64)

	// SerialiseFloat serialises a floating-point value
	SerialiseFloat(f float64)

	// SerialiseTime serialises a date and time value
	SerialiseTime(t time.Time)

	// SerialisePoint serialises a geographic point value
	SerialisePoint(p Point)

	// SerialiseStringList serialises a slice of string values
	SerialiseStringList(els []string)

	// SerialiseStringList serialises a slice of integer values
	SerialiseIntList(els []int64)

	// SerialiseStringList serialises a slice of floating-point values
	SerialiseFloatList(els []float64)
}

// Deserialiser is used when reading paging tokens
type Deserialiser interface {

	// DeserialiseString deserialises a string value
	DeserialiseString(d []byte) (string, error)

	// DeserialiseListString deserialises a list of string values
	DeserialiseListString(d []byte) ([]string, error)

	// DeserialiseInt deserialises an integer value
	DeserialiseInt(d []byte) (int64, error)

	// DeserialiseListInt deserialises a list of integer values
	DeserialiseListInt(d []byte) ([]int64, error)

	// DeserialiseFloat deserialises a floating-point value
	DeserialiseFloat(d []byte) (float64, error)

	// DeserialiseListFloat deserialises a list of floating-point values
	DeserialiseListFloat(d []byte) ([]float64, error)

	// DeserialiseTime deserialises a time value
	DeserialiseTime(d []byte) (time.Time, error)

	// DeserialisePoint deserialises a geographic point value
	DeserialisePoint(d []byte) (Point, error)
}

// QueryWriter is used to construct sql queries for a given sql flavour
type QueryWriter interface {

	// WriteQueryStatement builds a query statement
	WriteQueryStatement() (QueryStatement, error)

	// WriteCountStatement builds a statement which returns the count of records
	// that match the main query
	WriteCountStatement() (QueryStatement, error)

	// WriteFilterExpressionNull writes a comparison expression for a null value
	WriteFilterExpressionNull(
		sb *querybuilder.Builder,
		field string,
		op ComparisonOperator,
	) error

	// WriteFilterExpressionString writes a comparison expression for a string
	// value
	WriteFilterExpressionString(
		sb *querybuilder.Builder,
		field string,
		op ComparisonOperator,
		value string,
	) error

	// WriteFilterExpressionInt writes a comparison expression for an integer
	// value
	WriteFilterExpressionInt(
		sb *querybuilder.Builder,
		field string,
		op ComparisonOperator,
		value int64,
	) error

	// WriteFilterExpressionFloat writes a comparison expression for a
	// floating-point value
	WriteFilterExpressionFloat(
		sb *querybuilder.Builder,
		field string,
		op ComparisonOperator,
		value float64,
	) error

	// WriteFilterExpressionPoint writes a comparison expression for a
	// geographic point value
	WriteFilterExpressionPoint(
		sb *querybuilder.Builder,
		field string,
		op ComparisonOperator,
		value Point,
	) error

	// WriteFilterExpressionDateTime writes a comparison expression for a time
	// value
	WriteFilterExpressionDateTime(
		sb *querybuilder.Builder,
		field string,
		op ComparisonOperator,
		value time.Time,
	) error

	// WriteFilterExpressionStringList writes a comparison expression where the
	// value is list of string values
	WriteFilterExpressionStringList(
		sb *querybuilder.Builder,
		field string,
		op ComparisonOperator,
		value []string,
	) error

	// WriteFilterExpressionIntList writes a comparison expression where the
	// value is list of integer values
	WriteFilterExpressionIntList(
		sb *querybuilder.Builder,
		field string,
		op ComparisonOperator,
		value []int64,
	) error

	// WriteFilterExpressionFloatList writes a comparison expression where the
	// value is list of floating-point values
	WriteFilterExpressionFloatList(
		sb *querybuilder.Builder,
		field string,
		op ComparisonOperator,
		value []float64,
	) error
}

// Value represents a concrete value type within a query. Value abstracts the
// concrete value types from the rest of the packag
type Value interface {

	// Type returns the associated ValueType
	Type() ValueType

	// SupportsType indicates whether the value can be used in a comparison
	// expression against a column of a given DbType
	SupportsType(t DbType) bool

	// SupportsOperator indicates whether the value can be used in a comparison
	// expression with a given comparison operator
	SupportsOperator(op ComparisonOperator) bool

	// Serialise is used to serialise a given value when building paging tokens.
	// It receives a serialiser, which provides the business logic to serialise
	// all value types - The implementation should just pass the underlying
	// value to the appropriate serialiser method
	Serialise(s Serialiser)

	// WriteFilterExpression is used to write a comparison expression. It
	// receives a QueryWriter which provides the necessary business logic to
	// write comparison expressions for all types - The implementation should
	// just pass the underlying value to the appropriate QueryWriter method
	WriteFilterExpression(
		b *querybuilder.Builder,
		writer QueryWriter,
		fieldName string,
		operator ComparisonOperator,
	) error
}

// ValueBuilder is responsible for converting concrete values into Value types
type ValueBuilder interface {
	// Null returns a Value representing null
	Null() Value

	// String returns a Value backed by the provided string
	String(str string) Value

	// Int returns a Value backed by the provided int
	Int(n int64) Value

	// Float returns a Value backed by the provided float
	Float(f float64) Value

	// Point returns a Value backed by the provided point
	Point(longitude float64, latitude float64) Value

	// DateTime returns a Value backed by the provided time
	DateTime(dt time.Time) Value

	// List returns a list type Value backed by the provided elements. It will
	// return an error if the elements are of mixed types or if the type is not
	// supported for lists
	List(elements []Value) (Value, error)

	// Execute deserialisation is used to read values from paging tokens. It
	// recevies a deserialiser which provides the necessary logic to deserialise
	// all value types - The implementation should use the appropriate
	// Deserialiser method to deserialise the data and return a Value type
	ExecuteDeserialisation(
		ds Deserialiser,
		t ValueType,
		d []byte,
	) (Value, error)
}

// ResolvedPath represents a path from one resource to another. It includes
// metadata for all resources in the path and all relationships involved in the
// traversal
type ResolvedPath struct {
	// Id is a unique id for the path which may be used as a key
	Id string

	// StartResource is metadata for the root resource in the path
	StartResource TableMetadata

	// Steps is a collection of Traversal steps from the start resource to the
	// end resource
	Steps []*TraversalStep

	// EndResource is metadata for the end resource in the path
	EndResource TableMetadata

	// Type is the relationship type of the final step, this can be used to
	// determine if the end resource will point to a single record or a
	// collection of records
	Type RelationshipType
}

// TraversalStep represents a path between one resource and another
type TraversalStep struct {

	// SubPathId is a unique id for for the path from the start resource of the
	// parent path to the end resource of the traversal step. It may be used as
	// a key
	SubPathId string

	// Relationship is the relationship metadata for the traversal
	Relationship RelationshipMetadata
}

// ResolvedColumn represents a database column
type ResolvedColumn struct {

	// ResolvedPath is a path from a root resource to the resource that the
	// column is on
	ResolvedPath ResolvedPath

	// Metadata is metadata for the column
	Metadata ColumnMetadata
}

// LogicalOperator represents a logical operatior, e.g. and/or
type LogicalOperator string

const (
	// LogicalAnd represents a logical and operator
	LogicalAnd LogicalOperator = "and"

	// LogicalOr represents a logical or operator
	LogicalOr LogicalOperator = "or"
)

// SupportedLogicalOperators is a mapping from the operator keyword to a logical
// operator. Keywords are in lowercase.
var SupportedLogicalOperators = map[string]LogicalOperator{
	"and": LogicalAnd,
	"or":  LogicalOr,
}

// SortDirectionOperator represents a sorting direction in an order by operation
type SortDirectionOperator string

const (
	// SortDirectionAsc represents an ascending sort direction
	SortDirectionAsc SortDirectionOperator = "asc"

	// SortDirectionDesc represents a descending sort direction
	SortDirectionDesc SortDirectionOperator = "desc"
)

// ComparisonOperator represents a comparison operator within a query
type ComparisonOperator string

const (
	// ComparisonEq represents an equality comparison operator
	ComparisonEq ComparisonOperator = "eq"

	// ComparisonNe represents a not equal comparison operator
	ComparisonNe ComparisonOperator = "ne"

	// ComparisonGt represents a greater than comparison operator
	ComparisonGt ComparisonOperator = "gt"

	// ComparisonGe represents a greater than or equal comparison operator
	ComparisonGe ComparisonOperator = "ge"

	// ComparisonLt represents a less than comparison operator
	ComparisonLt ComparisonOperator = "lt"

	// ComparisonLe represents a less than or equal comparison operator
	ComparisonLe ComparisonOperator = "le"

	// ComparisonIn represents an in comparison operator
	ComparisonIn ComparisonOperator = "in"

	// ComparisonContains represents a contains comparison operator
	ComparisonContains ComparisonOperator = "contains"

	// ComparisonStartsWith represents a starts with comparison operator
	ComparisonStartsWith ComparisonOperator = "startswith"

	// ComparisonEndsWith represents an ends with comparison operator
	ComparisonEndsWith ComparisonOperator = "endswith"

	// -- NOT SUPPORTED FOR QUERY STRINGS -- //

	// ComparisonNotIn represents a not in operator. It is not currently
	// supported for query strings, it is used for negation logic only
	ComparisonNotIn ComparisonOperator = "notin"

	// ComparisonNotContains represents a not contains operator. It is not
	// currently supported for query strings, it is used for negation logic only
	ComparisonNotContains ComparisonOperator = "notcontains"

	// ComparisonNotStartsWith represents a not starts with operator. It is not
	// currently supported for query strings, it is used for negation logic only
	ComparisonNotStartsWith ComparisonOperator = "notstartswith"

	// ComparisonNotEndsWith represents a not ends with operator. It is not
	// currently supported for query strings, it is used for negation logic only
	ComparisonNotEndsWith ComparisonOperator = "notendswith"
)

// SupportedComparisonOperators is a mapping from the operator keyword to a
// comparison operator. Keywords are in lowercase.
var SupportedComparisonOperators = map[string]ComparisonOperator{
	"eq":         ComparisonEq,
	"ne":         ComparisonNe,
	"gt":         ComparisonGt,
	"ge":         ComparisonGe,
	"lt":         ComparisonLt,
	"le":         ComparisonLe,
	"in":         ComparisonIn,
	"contains":   ComparisonContains,
	"startswith": ComparisonStartsWith,
	"endswith":   ComparisonEndsWith,
}

// ComparableDbTypes is a set of DbTypes that can be used in comparison
// expressions
var ComparableDbTypes = map[DbType]struct{}{
	DbTypeString:   {},
	DbTypeInt:      {},
	DbTypeFloat:    {},
	DbTypeDateTime: {},
}

// CollectionOperator represents a collection operator, e.g. 'any' and 'all'
type CollectionOperator string

const (
	// CollectionAny represents the 'any' collection operator
	CollectionAny CollectionOperator = "any"

	// CollectionAll represents the 'all' collection operator
	CollectionAll CollectionOperator = "all"
)

// SupportedCollectionOperators is a mapping from the operator keyword to a
// collection operator. Keywords are in lowercase.
var SupportedCollectionOperators = map[string]CollectionOperator{
	"any": CollectionAny,
	"all": CollectionAll,
}

// FilterExpression represents a filter expression, such as a
// ComparisonExpression, CollectionExpression, or LogicalExpression.
type FilterExpression interface {
	// IsFilterExpression marks the type as a FilterExpression.
	IsFilterExpression()
}

// LogicalExpression combines two filter expressions using a logical operator.
type LogicalExpression struct {
	// Left is the left-hand filter expression.
	Left FilterExpression `json:"left"`

	// Operator defines the logical operation to perform.
	Operator LogicalOperator `json:"operator"`

	// Right is the right-hand filter expression.
	Right FilterExpression `json:"right"`
}

// IsFilterExpression marks the type as a FilterExpression.
func (e *LogicalExpression) IsFilterExpression() {}

// ExistsNode represents a node in an EXISTS query traversal.
type ExistsNode struct {
	// Step is the relationship traversal step used to build the EXISTS query.
	Step *TraversalStep

	// Alias is the SQL alias assigned to the resource in the EXISTS query.
	Alias string

	// ParentAlias is the SQL alias of the parent resource used to correlate the
	// EXISTS query with its parent query.
	ParentAlias string
}

// ComparisonExpression compares a resolved column with a value using a
// comparison operator.
type ComparisonExpression struct {
	// ResolvedColumn is the column to which the comparison is applied.
	ResolvedColumn *ResolvedColumn

	// Operator defines the comparison to perform.
	Operator ComparisonOperator

	// Value is the value to compare against the resolved column.
	Value Value

	// ExistsNodes contains the traversal path used to build an EXISTS query for
	// the comparison.
	ExistsNodes []ExistsNode
}

// IsFilterExpression marks the type as a FilterExpression.
func (e *ComparisonExpression) IsFilterExpression() {}

// WithValue returns a copy of the expression with the given operator and value.
func (e *ComparisonExpression) WithValue(
	operator ComparisonOperator,
	value Value,
) *ComparisonExpression {
	c := *e

	c.Operator = operator
	c.Value = value

	return &c
}

// CollectionExpression applies a filter expression to a resolved collection
// path using a collection operator.
type CollectionExpression struct {
	// ResolvedPath is the collection path to which the expression is applied.
	ResolvedPath *ResolvedPath

	// Operator defines the collection operation to perform.
	Operator CollectionOperator

	// FilterExpression is the filter expression applied to the collection.
	FilterExpression FilterExpression

	// ExistsNodes contains the traversal path used to build an EXISTS query for
	// the collection expression.
	ExistsNodes []ExistsNode
}

// WithValues returns a copy of the expression with the given operator and
// filter expression.
func (e *CollectionExpression) WithValues(
	operator CollectionOperator,
	expression FilterExpression,
) *CollectionExpression {
	c := *e

	c.Operator = operator
	c.FilterExpression = expression

	return &c
}

// IsFilterExpression marks the type as a FilterExpression.
func (e *CollectionExpression) IsFilterExpression() {}

// SortingRule represents a single order by value
type SortingRule struct {
	// ResolvedColumn is the column to sort by.
	ResolvedColumn ResolvedColumn

	// Direction is the sort direction to apply.
	Direction SortDirectionOperator
}

// PayloadSigner signs payloads and verifies signed payloads using a secret.
type PayloadSigner interface {
	// Sign returns a signed representation of the supplied data.
	Sign(secret []byte, data []byte) string

	// Verify verifies a signed value and returns the original data when valid.
	Verify(secret []byte, value string) (data []byte, ok bool)
}
