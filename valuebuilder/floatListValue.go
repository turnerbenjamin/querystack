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

// floatListValue represents a list of float values.
type floatListValue struct {
	value []float64
}

// supportedComparisonOperatorsFloatList is the set of comparison operations
// that are supported by the FloatList type
var supportedComparisonOperatorsFloatList = map[mdl.ComparisonOperator]struct{}{
	mdl.ComparisonIn:    {},
	mdl.ComparisonNotIn: {},
}

// Type returns the query value type represented by the value.
func (v floatListValue) Type() mdl.ValueType {
	return mdl.ValueTypeIntList
}

// SupportsType reports whether the value can be used with the given database
// type.
func (v floatListValue) SupportsType(t mdl.DbType) bool {
	return t == mdl.DbTypeFloat
}

// SupportsOperator reports whether the value supports the given comparison
// operator.
func (v floatListValue) SupportsOperator(op mdl.ComparisonOperator) bool {
	_, supported := supportedComparisonOperatorsFloatList[op]
	return supported
}

// WriteFilterExpression delegates filter generation to the query writer for the
// float list value.
func (v floatListValue) WriteFilterExpression(
	b *querybuilder.Builder,
	w mdl.QueryWriter,
	fieldName string,
	op mdl.ComparisonOperator,
) error {
	return w.WriteFilterExpressionFloatList(b, fieldName, op, v.value)
}

// Serialise delegates serialization of the float list to the provided
// serialiser.
func (v floatListValue) Serialise(s mdl.Serialiser) {
	s.SerialiseFloatList(v.value)
}

// buildFloatListValue constructs a float list value from float values.
func buildFloatListValue(els []mdl.Value) (mdl.Value, error) {
	o := floatListValue{
		value: make([]float64, len(els)),
	}
	for i, el := range els {
		switch v := el.(type) {
		case floatValue:
			o.value[i] = v.value
		default:
			return nil, qerr.InternalErr(
				"float list does not support elements of type %s",
				ValueTypeString(el.Type()),
			)
		}
	}
	return o, nil
}
