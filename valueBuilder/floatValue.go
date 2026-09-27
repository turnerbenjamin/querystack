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

// floatValue represents a float query value.
type floatValue struct {
	value float64
}

// supportedComparisonOperatorsFloat lists the comparison operators supported
// by float values.
var supportedComparisonOperatorsFloat = map[mdl.ComparisonOperator]struct{}{
	mdl.ComparisonEq: {},
	mdl.ComparisonNe: {},
	mdl.ComparisonGt: {},
	mdl.ComparisonGe: {},
	mdl.ComparisonLt: {},
	mdl.ComparisonLe: {},
}

// Type returns the query value type represented by the value.
func (v floatValue) Type() mdl.ValueType {
	return mdl.ValueTypeFloat
}

// SupportsType reports whether the value can be used with the given database
// type.
func (v floatValue) SupportsType(t mdl.DbType) bool {
	return t == mdl.DbTypeFloat
}

// SupportsOperator reports whether the value supports the given comparison
// operator.
func (v floatValue) SupportsOperator(op mdl.ComparisonOperator) bool {
	_, supported := supportedComparisonOperatorsFloat[op]
	return supported
}

// WriteFilterExpression delegates filter generation to the query writer for the
// float value.
func (v floatValue) WriteFilterExpression(
	b *querybuilder.Builder,
	w mdl.QueryWriter,
	fieldName string,
	op mdl.ComparisonOperator,
) error {
	return w.WriteFilterExpressionFloat(b, fieldName, op, v.value)
}

// Serialise delegates serialization of the float value to the provided
// serialiser.
func (v floatValue) Serialise(s mdl.Serialiser) {
	s.SerialiseFloat(v.value)
}
