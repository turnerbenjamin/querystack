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

// NullValue represents a null query value.
type NullValue struct{}

// Type returns the query value type represented by the value.
func (v NullValue) Type() mdl.ValueType {
	return mdl.ValueTypeNull
}

// SupportsComparison indicates if a comparison can be made with the given value
func (v NullValue) SupportsComparison(
	t mdl.DbType,
	op mdl.ComparisonOperator,
) bool {
	return op == mdl.ComparisonEq || op == mdl.ComparisonNe
}

// WriteFilterExpression delegates filter generation for the null value to the
// query writer.
func (v NullValue) WriteFilterExpression(
	b *querybuilder.Builder,
	w mdl.QueryWriter,
	resourceAlias string,
	column *mdl.ResolvedColumn,
	op mdl.ComparisonOperator,
) error {
	return w.WriteFilterExpressionNull(b, resourceAlias, column, op)
}

// Serialise delegates serialization of the null value to the provided
// serialiser.
func (v NullValue) Serialise(s mdl.Serialiser) {
	s.SerialiseNull()
}
