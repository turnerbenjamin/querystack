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

// pointValue represents a point query value.
type pointValue struct {
	value mdl.Point
}

// supportedComparisonOperatorsPoint lists the comparison operators supported by
// point values.
var supportedComparisonOperatorsPoint = map[mdl.ComparisonOperator]struct{}{}

// Type returns the query value type represented by the value.
func (v pointValue) Type() mdl.ValueType {
	return mdl.ValueTypePoint
}

// SupportsType reports whether the value can be used with the given database
// type.
func (v pointValue) SupportsType(t mdl.DbType) bool {
	return t == mdl.DbTypePoint
}

// SupportsOperator reports whether the value supports the given comparison
// operator.
func (v pointValue) SupportsOperator(op mdl.ComparisonOperator) bool {
	_, supported := supportedComparisonOperatorsPoint[op]
	return supported
}

// WriteFilterExpression delegates filter generation to the query writer for
// the point value.
func (v pointValue) WriteFilterExpression(
	b *querybuilder.Builder,
	w mdl.QueryWriter,
	fieldName string,
	op mdl.ComparisonOperator,
) error {
	return w.WriteFilterExpressionPoint(b, fieldName, op, v.value)
}

// Serialise delegates serialization of the point value to the provided
// serialiser.
func (v pointValue) Serialise(s mdl.Serialiser) {
	s.SerialisePoint(v.value)
}
