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

// FloatListValue represents a list of float values.
type FloatListValue struct {
	Value []float64
}

// Type returns the query value type represented by the value.
func (v FloatListValue) Type() mdl.ValueType {
	return mdl.ValueTypeIntList
}

// SupportsComparison indicates if a comparison can be made with the given value
func (v FloatListValue) SupportsComparison(
	t mdl.DbType,
	op mdl.ComparisonOperator,
) bool {
	if t != mdl.DbTypeFloat {
		return false
	}
	return mdl.DbTypeSupportsCollectionValueOperation(mdl.DbTypeFloat, op)
}

// WriteFilterExpression delegates filter generation to the query writer for the
// float list value.
func (v FloatListValue) WriteFilterExpression(
	b *querybuilder.Builder,
	w mdl.QueryWriter,
	resourceAlias string,
	column *mdl.ResolvedColumn,
	op mdl.ComparisonOperator,
) error {
	return w.WriteFilterExpressionFloatList(b, resourceAlias, column, op, v.Value)
}

// Serialise delegates serialization of the float list to the provided
// serialiser.
func (v FloatListValue) Serialise(s mdl.Serialiser) {
	s.SerialiseFloatList(v.Value)
}

// buildFloatListValue constructs a float list value from float values.
func buildFloatListValue(els []mdl.Value) (mdl.Value, error) {
	o := FloatListValue{
		Value: make([]float64, len(els)),
	}
	for i, el := range els {
		switch v := el.(type) {
		case FloatValue:
			o.Value[i] = v.Value
		default:
			return nil, qerr.InternalErr(
				"float list does not support elements of type %s",
				ValueTypeString(el.Type()),
			)
		}
	}
	return o, nil
}
