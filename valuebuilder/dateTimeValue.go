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

// dateTimeValue represents a date-time query value.
type dateTimeValue struct {
	value time.Time
}

// Type returns the query value type represented by the value.
func (v dateTimeValue) Type() mdl.ValueType {
	return mdl.ValueTypeDateTime
}

// SupportsComparison indicates if a comparison can be made with the given value
func (v dateTimeValue) SupportsComparison(
	t mdl.DbType,
	op mdl.ComparisonOperator,
) bool {
	if t != mdl.DbTypeDateTime {
		return false
	}
	return mdl.DbTypeSupportsSingleValueOperation(mdl.DbTypeDateTime, op)
}

// WriteFilterExpression writes a filter expression for the date-time value.
func (v dateTimeValue) WriteFilterExpression(
	b *querybuilder.Builder,
	w mdl.QueryWriter,
	fieldName string,
	op mdl.ComparisonOperator,
) error {
	return w.WriteFilterExpressionDateTime(b, fieldName, op, v.value)
}

// Serialise writes the date-time value to the serialiser.
func (v dateTimeValue) Serialise(s mdl.Serialiser) {
	s.SerialiseTime(v.value)
}
