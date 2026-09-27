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

// intListValue represents a list of integer query values.
type intListValue struct {
	value []int64
}

// supportedComparisonOperatorsIntList lists the comparison operators supported
// by integer list values.
var supportedComparisonOperatorsIntList = map[mdl.ComparisonOperator]struct{}{
	mdl.ComparisonIn:    {},
	mdl.ComparisonNotIn: {},
}

// Type returns the query value type represented by the value.
func (v intListValue) Type() mdl.ValueType {
	return mdl.ValueTypeIntList
}

// SupportsType reports whether the value can be used with the given database
// type.
func (v intListValue) SupportsType(t mdl.DbType) bool {
	return t == mdl.DbTypeInt || t == mdl.DbTypeFloat
}

// SupportsOperator reports whether the value supports the given comparison
// operator.
func (v intListValue) SupportsOperator(op mdl.ComparisonOperator) bool {
	_, supported := supportedComparisonOperatorsIntList[op]
	return supported
}

// WriteFilterExpression delegates filter generation to the query writer for the
// integer list value.
func (v intListValue) WriteFilterExpression(
	b *querybuilder.Builder,
	w mdl.QueryWriter,
	fieldName string,
	op mdl.ComparisonOperator,
) error {
	return w.WriteFilterExpressionIntList(b, fieldName, op, v.value)
}

// Serialise delegates serialization of the integer list to the provided
// serialiser.
func (v intListValue) Serialise(s mdl.Serialiser) {
	s.SerialiseIntList(v.value)
}

// buildIntListValue constructs an integer list value from integer values.
func buildIntListValue(els []mdl.Value) (mdl.Value, error) {
	o := intListValue{
		value: make([]int64, len(els)),
	}
	for i, el := range els {
		switch v := el.(type) {
		case intValue:
			o.value[i] = v.value
		default:
			return nil, qerr.InternalErr(
				"int list does not support elements of type %s",
				ValueTypeString(el.Type()),
			)
		}
	}
	return o, nil
}
