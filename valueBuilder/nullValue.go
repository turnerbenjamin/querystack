// Package valueBuilder provides type-safe abstractions for query values. Values
// act as a boundary between the concrete Go types used by parsed queries and
// the rest of the query system.
//
// Values identify their value and value type and control which operations are
// permitted. Filter generation and serialization are delegated to injected
// query components. Values act as type switches for these operations and do not
// implement the underlying business logic.
package valueBuilder

import (
	querybuilder "github.com/turnerbenjamin/querystack/queryBuilder"
	mdl "github.com/turnerbenjamin/querystack/queryModel"
)

// nullValue represents a null query value.
type nullValue struct{}

// supportedComparisonOperatorsNull lists the comparison operators supported by
// null values.
var supportedComparisonOperatorsNull = map[mdl.ComparisonOperator]struct{}{
	mdl.ComparisonEq: {},
	mdl.ComparisonNe: {},
}

// Type returns the query value type represented by the value.
func (v nullValue) Type() mdl.ValueType {
	return mdl.ValueTypeNull
}

// SupportsType reports that null values can be used with any database type.
func (v nullValue) SupportsType(_ mdl.DbType) bool {
	return true
}

// SupportsOperator reports whether the value supports the given comparison
// operator.
func (v nullValue) SupportsOperator(op mdl.ComparisonOperator) bool {
	_, supported := supportedComparisonOperatorsNull[op]
	return supported
}

// WriteFilterExpression delegates filter generation for the null value to the
// query writer.
func (v nullValue) WriteFilterExpression(
	b *querybuilder.Builder,
	w mdl.QueryWriter,
	fieldName string,
	op mdl.ComparisonOperator,
) error {
	return w.WriteFilterExpressionNull(b, fieldName, op)
}

// Serialise delegates serialization of the null value to the provided
// serialiser.
func (v nullValue) Serialise(s mdl.Serialiser) {
	s.SerialiseNull()
}
