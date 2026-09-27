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

// stringValue represents a string query value.
type stringValue struct {
	value string
}

// supportedComparisonOperatorsString lists the comparison operators supported
// by string values.
var supportedComparisonOperatorsString = map[mdl.ComparisonOperator]struct{}{
	mdl.ComparisonEq:            {},
	mdl.ComparisonNe:            {},
	mdl.ComparisonContains:      {},
	mdl.ComparisonStartsWith:    {},
	mdl.ComparisonEndsWith:      {},
	mdl.ComparisonNotContains:   {},
	mdl.ComparisonNotStartsWith: {},
	mdl.ComparisonNotEndsWith:   {},
	mdl.ComparisonGe:            {},
	mdl.ComparisonGt:            {},
	mdl.ComparisonLt:            {},
	mdl.ComparisonLe:            {},
}

// Type returns the query value type represented by the value.
func (v stringValue) Type() mdl.ValueType {
	return mdl.ValueTypeString
}

// SupportsType reports whether the value can be used with the given database
// type.
func (v stringValue) SupportsType(t mdl.DbType) bool {
	return t == mdl.DbTypeString
}

// SupportsOperator reports whether the value supports the given comparison
// operator.
func (v stringValue) SupportsOperator(op mdl.ComparisonOperator) bool {
	_, supported := supportedComparisonOperatorsString[op]
	return supported
}

// WriteFilterExpression delegates filter generation to the query writer for the
// string value.
func (v stringValue) WriteFilterExpression(
	b *querybuilder.Builder,
	w mdl.QueryWriter,
	fieldName string,
	op mdl.ComparisonOperator,
) error {
	return w.WriteFilterExpressionString(b, fieldName, op, v.value)
}

// Serialise delegates serialization of the string value to the provided
// serialiser.
func (v stringValue) Serialise(s mdl.Serialiser) {
	s.SerialiseString(v.value)
}
