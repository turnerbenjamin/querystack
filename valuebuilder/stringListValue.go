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
	qerr "github.com/turnerbenjamin/querystack/queryerror"
	mdl "github.com/turnerbenjamin/querystack/querymodel"
)

// stringListValue represents a list of string query values.
type stringListValue struct {
	value []string
}

// supportedComparisonOperatorsStringList lists the comparison operators
// supported by string list values.
var supportedComparisonOperatorsStringList = map[mdl.ComparisonOperator]struct{}{
	mdl.ComparisonIn:    {},
	mdl.ComparisonNotIn: {},
}

// Type returns the query value type represented by the value.
func (v stringListValue) Type() mdl.ValueType {
	return mdl.ValueTypeStringList
}

// SupportsType reports whether the value can be used with the given database
// type.
func (v stringListValue) SupportsType(t mdl.DbType) bool {
	return t == mdl.DbTypeString
}

// SupportsOperator reports whether the value supports the given comparison
// operator.
func (v stringListValue) SupportsOperator(op mdl.ComparisonOperator) bool {
	_, supported := supportedComparisonOperatorsStringList[op]
	return supported
}

// WriteFilterExpression delegates filter generation to the query writer for the
// string list value.
func (v stringListValue) WriteFilterExpression(
	b *querybuilder.Builder,
	w mdl.QueryWriter,
	fieldName string,
	op mdl.ComparisonOperator,
) error {
	return w.WriteFilterExpressionStringList(b, fieldName, op, v.value)
}

// buildStringListValue constructs a string list value from string values.
func buildStringListValue(els []mdl.Value) (mdl.Value, error) {
	o := stringListValue{
		value: make([]string, len(els)),
	}
	for i, el := range els {
		switch v := el.(type) {
		case stringValue:
			o.value[i] = v.value
		default:
			return nil, qerr.InternalErr(
				"string list does not support elements of type %s",
				ValueTypeString(el.Type()),
			)
		}
	}
	return o, nil
}

// Serialise delegates serialization of the string list to the provided
// serialiser.
func (v stringListValue) Serialise(s mdl.Serialiser) {
	s.SerialiseStringList(v.value)
}
