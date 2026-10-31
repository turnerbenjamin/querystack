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
	"time"

	querybuilder "github.com/turnerbenjamin/querystack/querybuilder"
	mdl "github.com/turnerbenjamin/querystack/querymodel"
)

// DateTimeValue represents a date-time query value.
type DateTimeValue struct {
	Value time.Time
}

// Type returns the query value type represented by the value.
func (v DateTimeValue) Type() mdl.ValueType {
	return mdl.ValueTypeDateTime
}

// SupportsComparison indicates if a comparison can be made with the given value
func (v DateTimeValue) SupportsComparison(
	t mdl.DbType,
	op mdl.ComparisonOperator,
) bool {
	if t != mdl.DbTypeDateTime && t != mdl.DbTypeDate {
		return false
	}
	return mdl.DbTypeSupportsSingleValueOperation(t, op)
}

// WriteFilterExpression writes a filter expression for the date-time value.
func (v DateTimeValue) WriteFilterExpression(
	b *querybuilder.Builder,
	w mdl.QueryWriter,
	resourceAlias string,
	column *mdl.ResolvedColumn,
	op mdl.ComparisonOperator,
) error {
	if column.Metadata.Type == mdl.DbTypeDate {
		return w.WriteFilterExpressionDate(b, resourceAlias, column, op, v.Value)
	} else {
		return w.WriteFilterExpressionDateTime(b, resourceAlias, column, op, v.Value)
	}
}

// Serialise writes the date-time value to the serialiser.
func (v DateTimeValue) Serialise(s mdl.Serialiser) {
	s.SerialiseTime(v.Value)
}
