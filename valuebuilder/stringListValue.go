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

// StringListValue represents a list of string query values.
type StringListValue struct {
	Value []string
}

// Type returns the query value type represented by the value.
func (v StringListValue) Type() mdl.ValueType {
	return mdl.ValueTypeStringList
}

// SupportsComparison indicates if a comparison can be made with the given value
func (v StringListValue) SupportsComparison(
	t mdl.DbType,
	op mdl.ComparisonOperator,
) bool {
	if t != mdl.DbTypeString {
		return false
	}
	return mdl.DbTypeSupportsCollectionValueOperation(mdl.DbTypeString, op)
}

// WriteFilterExpression delegates filter generation to the query writer for the
// string list value.
func (v StringListValue) WriteFilterExpression(
	b *querybuilder.Builder,
	w mdl.QueryWriter,
	resourceAlias string,
	column *mdl.ResolvedColumn,
	op mdl.ComparisonOperator,
) error {
	return w.WriteFilterExpressionStringList(b, resourceAlias, column, op, v.Value)
}

// buildStringListValue constructs a string list value from string values.
func buildStringListValue(els []mdl.Value) (mdl.Value, error) {
	o := StringListValue{
		Value: make([]string, len(els)),
	}
	for i, el := range els {
		switch v := el.(type) {
		case StringValue:
			o.Value[i] = v.Value
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
func (v StringListValue) Serialise(s mdl.Serialiser) {
	s.SerialiseStringList(v.Value)
}
