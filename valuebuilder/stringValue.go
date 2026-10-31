// Package valuebuilder provides type-safe abstractions for query values. Values
// act as a boundary between the concrete Go types used by parsed queries and
// the rest of the query system.
//
// Values identify their value and value type and control which operations are
// permitted. Filter generation and serialization are delegated to injected
// query components. Values act as type switches for these operations and do not
// implement the underlying business logic.
package valuebuilder

import (
	querybuilder "github.com/turnerbenjamin/querystack/querybuilder"
	mdl "github.com/turnerbenjamin/querystack/querymodel"
)

// StringValue represents a string query value.
type StringValue struct {
	Value string
}

// Type returns the query value type represented by the value.
func (v StringValue) Type() mdl.ValueType {
	return mdl.ValueTypeString
}

// SupportsComparison indicates if a comparison can be made with the given value
func (v StringValue) SupportsComparison(
	t mdl.DbType,
	op mdl.ComparisonOperator,
) bool {
	if t != mdl.DbTypeString {
		return false
	}
	return mdl.DbTypeSupportsSingleValueOperation(mdl.DbTypeString, op)
}

// WriteFilterExpression delegates filter generation to the query writer for the
// string value.
func (v StringValue) WriteFilterExpression(
	b *querybuilder.Builder,
	w mdl.QueryWriter,
	resourceAlias string,
	column *mdl.ResolvedColumn,
	op mdl.ComparisonOperator,
) error {
	return w.WriteFilterExpressionString(b, resourceAlias, column, op, v.Value)
}

// Serialise delegates serialization of the string value to the provided
// serialiser.
func (v StringValue) Serialise(s mdl.Serialiser) {
	s.SerialiseString(v.Value)
}
