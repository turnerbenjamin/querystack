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

// floatValue represents a float query value.
type floatValue struct {
	value float64
}

// Type returns the query value type represented by the value.
func (v floatValue) Type() mdl.ValueType {
	return mdl.ValueTypeFloat
}

// SupportsComparison indicates if a comparison can be made with the given value
func (v floatValue) SupportsComparison(
	t mdl.DbType,
	op mdl.ComparisonOperator,
) bool {
	if t != mdl.DbTypeFloat {
		return false
	}
	return mdl.DbTypeSupportsSingleValueOperation(mdl.DbTypeFloat, op)
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
