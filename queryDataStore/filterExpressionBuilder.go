// Package queryDataStore is responsible for maintaining query data in a central
// store. In earlier iterations, a pipeline approach was taken where the query
// data was enriched at each stage of the pipeline; this resulted in a clear
// separation of responsibilities but it created complications when accessing
// the data which may or may not have been fully enriched.
//
// This package takes the alternative approach of calling the relevant
// metadataBinder and relationshipPlanner methods as operations are added to the
// data. This allows consuming packages to access the data with the assurance
// that all metadata and relationship data will always be populated
//
// This file contains the filterExpressionBuilder. Unlike other operations such
// as select or orderby, filters are relatively complex and built incrementally;
// it is not helpful in this instance to bind the metadata only once the filter
// is added to the store - The filterExpressionBuilder solves this by exposing
// methods to build fully enriched filterExpressions without adding them to the
// store
package queryDataStore

import (
	qerr "github.com/turnerbenjamin/querystack/queryError"
	mdl "github.com/turnerbenjamin/querystack/queryModel"
	"github.com/turnerbenjamin/querystack/relationships"
	"github.com/turnerbenjamin/querystack/valueBuilder"
)

// filterExpressionBuilder is used to build filterExpressions which are enriched
// with metadata, and processed by the metadata planner as required, on creation
type filterExpressionBuilder struct {
	rootResourceMetadata mdl.TableMetadata
	metadataBinder       MetadataBinder
	relationshipPlanner  *relationships.RelationshipPlanner
	valueBuilder         mdl.ValueBuilder
}

// ValueBuilder returns the Value builder instance for the
// filterExpressionBuilder
func (eb filterExpressionBuilder) ValueBuilder() mdl.ValueBuilder {
	return eb.valueBuilder
}

// NewLogicalExpression creates a new logical expression
func (eb filterExpressionBuilder) NewLogicalExpression(
	left mdl.FilterExpression,
	operator mdl.LogicalOperator,
	right mdl.FilterExpression,
) (*mdl.LogicalExpression, error) {
	if left == nil || right == nil {
		return nil, qerr.InternalErr(
			"unable to build logical expression: left and right must not be nil",
		)
	}

	return &mdl.LogicalExpression{
		Left:     left,
		Operator: operator,
		Right:    right,
	}, nil
}

// NewComparisonExpression builds a new comparison expression. It ensures that
// the columnPath is bound to schema metadata, invokes validation functions that
// use this metadata and ensures that the relationshipPlanner processes the
// column
func (eb filterExpressionBuilder) NewComparisonExpression(
	columnPath string,
	operator mdl.ComparisonOperator,
	value mdl.Value,
) (*mdl.ComparisonExpression, error) {
	column, err := eb.metadataBinder.ResolveColumn(columnPath)
	if err != nil {
		return nil, err
	}

	return eb.NewComparisonExpressionFromResolvedColumn(column, operator, value)
}

// NewComparisonExpressionFromResolvedColumn is used to build a new comparison
// expression where the column has been already been resolved - For instance,
// when creating an expression for a cursor filter where the column has already
// been resolved when builing the associated orderbyrule
func (eb filterExpressionBuilder) NewComparisonExpressionFromResolvedColumn(
	column mdl.ResolvedColumn,
	operator mdl.ComparisonOperator,
	value mdl.Value,
) (*mdl.ComparisonExpression, error) {
	// If the column resolves to an entity other than the root entity, ensure
	// that the relationship type is N:1 so that there is a single record to
	// make the comparison against
	if len(column.ResolvedPath.Steps) > 0 && column.ResolvedPath.Type != mdl.RelationshipManyToOne {
		return nil, qerr.BindingErr(
			"invalid path: %s. Comparison operations are only supported "+
				"for N:1 and 1:1 relationships. Please use a collection "+
				"operator",
			column.ResolvedPath.Id,
		)
	}

	// Validate that column and value have compatible types
	if !value.SupportsType(column.Metadata.Type) {
		return nil, qerr.SyntaxErr(
			"%s is not compatible with %s values",
			column.Metadata.Name,
			valueBuilder.ValueTypeString(value.Type()),
		)
	}

	// Validate that the value is compatible with the operator
	if !value.SupportsOperator(operator) {
		return nil, qerr.SyntaxErr(
			"values of type %s are not compatible with the %s operator",
			valueBuilder.ValueTypeString(value.Type()),
			operator,
		)
	}

	existsNodes := eb.relationshipPlanner.ProcessExists(column.ResolvedPath)

	return &mdl.ComparisonExpression{
		ResolvedColumn: &column,
		Operator:       operator,
		Value:          value,
		ExistsNodes:    existsNodes,
	}, nil
}

// NewCollectionExpression builds a new collection expression. It ensures that
// the path is bound to metadata and processed by the relationship planner
func (eb filterExpressionBuilder) NewCollectionExpression(
	resourcePath string,
	operator mdl.CollectionOperator,
	filterExpression mdl.FilterExpression,
) (*mdl.CollectionExpression, error) {
	if filterExpression == nil {
		return nil, qerr.InternalErr(
			"unable to build collection expression: filter expression cannot be nil",
		)
	}

	resolvedPath, err := eb.metadataBinder.ResolvePath(resourcePath)
	if err != nil {
		return nil, qerr.InternalErr("unable to build collection expression path: %w", err)
	}

	if resolvedPath.Type != mdl.RelationshipOneToMany {
		return nil, qerr.BindingErr(
			"invalid value - '%s'. Collection operations are only supported "+
				"for 1:N relationships",
			resolvedPath.Id,
		)
	}

	existsNodes := eb.relationshipPlanner.ProcessExists(resolvedPath)

	return &mdl.CollectionExpression{
		ResolvedPath:     &resolvedPath,
		Operator:         operator,
		FilterExpression: filterExpression,
		ExistsNodes:      existsNodes,
	}, nil
}
