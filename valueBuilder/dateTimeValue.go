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
	"time"

	querybuilder "github.com/turnerbenjamin/querystack/queryBuilder"
	mdl "github.com/turnerbenjamin/querystack/queryModel"
)

// dateTimeValue represents a date-time query value.
type dateTimeValue struct {
	value time.Time
}

// supportedComparisonOperatorsDateTime is the set of comparison operators that
// dateTime supports
var supportedComparisonOperatorsDateTime = map[mdl.ComparisonOperator]struct{}{
	mdl.ComparisonEq: {},
	mdl.ComparisonNe: {},
	mdl.ComparisonGt: {},
	mdl.ComparisonGe: {},
	mdl.ComparisonLt: {},
	mdl.ComparisonLe: {},
}

// Type returns the query value type represented by the value.
func (v dateTimeValue) Type() mdl.ValueType {
	return mdl.ValueTypeDateTime
}

// SupportsType reports whether the value can be used with the given database
// type.
func (v dateTimeValue) SupportsType(t mdl.DbType) bool {
	return t == mdl.DbTypeDateTime
}

// SupportsOperator reports whether the value supports the given comparison
// operator.
func (v dateTimeValue) SupportsOperator(op mdl.ComparisonOperator) bool {
	_, supported := supportedComparisonOperatorsDateTime[op]
	return supported
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
