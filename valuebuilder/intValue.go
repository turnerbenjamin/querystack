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

// Type returns the query value type represented by the value.
func (v intValue) Type() mdl.ValueType {
	return mdl.ValueTypeInt
}

// SupportsComparison indicates if a comparison can be made with the given value
func (v intValue) SupportsComparison(
	t mdl.DbType,
	op mdl.ComparisonOperator,
) bool {
	if t != mdl.DbTypeInt && t != mdl.DbTypeFloat && t != mdl.DbTypeEnum {
		return false
	}
	return mdl.DbTypeSupportsSingleValueOperation(t, op)
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
