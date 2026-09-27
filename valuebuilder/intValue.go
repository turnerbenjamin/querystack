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

// intValue represents an integer query value.
type intValue struct {
	value int64
}

// supportedComparisonOperatorsInt lists the comparison operators supported by
// integer values.
var supportedComparisonOperatorsInt = map[mdl.ComparisonOperator]struct{}{
	mdl.ComparisonEq: {},
	mdl.ComparisonNe: {},
	mdl.ComparisonGt: {},
	mdl.ComparisonGe: {},
	mdl.ComparisonLt: {},
	mdl.ComparisonLe: {},
}

// Type returns the query value type represented by the value.
func (v intValue) Type() mdl.ValueType {
	return mdl.ValueTypeInt
}

// SupportsType reports whether the value can be used with the given database
// type.
func (v intValue) SupportsType(t mdl.DbType) bool {
	return t == mdl.DbTypeInt || t == mdl.DbTypeFloat
}

// SupportsOperator reports whether the value supports the given comparison
// operator.
func (v intValue) SupportsOperator(op mdl.ComparisonOperator) bool {
	_, supported := supportedComparisonOperatorsInt[op]
	return supported
}

// WriteFilterExpression delegates filter generation to the query writer for the
// integer value.
func (v intValue) WriteFilterExpression(
	b *querybuilder.Builder,
	w mdl.QueryWriter,
	fieldName string,
	op mdl.ComparisonOperator,
) error {
	return w.WriteFilterExpressionInt(b, fieldName, op, v.value)
}

// Serialise delegates serialization of the integer value to the provided
// serialiser.
func (v intValue) Serialise(s mdl.Serialiser) {
	s.SerialiseInt(v.value)
}
