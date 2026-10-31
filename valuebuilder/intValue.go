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

// IntValue represents an integer query value.
type IntValue struct {
	Value int64
}

// Type returns the query value type represented by the value.
func (v IntValue) Type() mdl.ValueType {
	return mdl.ValueTypeInt
}

// SupportsComparison indicates if a comparison can be made with the given value
func (v IntValue) SupportsComparison(
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
func (v IntValue) WriteFilterExpression(
	b *querybuilder.Builder,
	w mdl.QueryWriter,
	resourceAlias string,
	column *mdl.ResolvedColumn,
	op mdl.ComparisonOperator,
) error {
	return w.WriteFilterExpressionInt(b, resourceAlias, column, op, v.Value)
}

// Serialise delegates serialization of the integer value to the provided
// serialiser.
func (v IntValue) Serialise(s mdl.Serialiser) {
	s.SerialiseInt(v.Value)
}
