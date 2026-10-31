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

// IntListValue represents a list of integer query values.
type IntListValue struct {
	Value []int64
}

// Type returns the query value type represented by the value.
func (v IntListValue) Type() mdl.ValueType {
	return mdl.ValueTypeIntList
}

// SupportsComparison indicates if a comparison can be made with the given value
func (v IntListValue) SupportsComparison(
	t mdl.DbType,
	op mdl.ComparisonOperator,
) bool {
	if t != mdl.DbTypeInt && t != mdl.DbTypeFloat {
		return false
	}
	return mdl.DbTypeSupportsCollectionValueOperation(t, op)
}

// WriteFilterExpression delegates filter generation to the query writer for the
// integer list value.
func (v IntListValue) WriteFilterExpression(
	b *querybuilder.Builder,
	w mdl.QueryWriter,
	resourceAlias string,
	column *mdl.ResolvedColumn,
	op mdl.ComparisonOperator,
) error {
	return w.WriteFilterExpressionIntList(b, resourceAlias, column, op, v.Value)
}

// Serialise delegates serialization of the integer list to the provided
// serialiser.
func (v IntListValue) Serialise(s mdl.Serialiser) {
	s.SerialiseIntList(v.Value)
}

// buildIntListValue constructs an integer list value from integer values.
func buildIntListValue(els []mdl.Value) (mdl.Value, error) {
	o := IntListValue{
		Value: make([]int64, len(els)),
	}
	for i, el := range els {
		switch v := el.(type) {
		case IntValue:
			o.Value[i] = v.Value
		default:
			return nil, qerr.InternalErr(
				"int list does not support elements of type %s",
				ValueTypeString(el.Type()),
			)
		}
	}
	return o, nil
}
