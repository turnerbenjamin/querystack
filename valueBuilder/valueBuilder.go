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

	qerr "github.com/turnerbenjamin/querystack/queryError"
	mdl "github.com/turnerbenjamin/querystack/queryModel"
)

// valueBuilder constructs query values and converts serialized values back into
// their corresponding value implementations.
type valueBuilder struct{}

// NewValueBuilder creates a value builder.
func NewValueBuilder() mdl.ValueBuilder {
	return &valueBuilder{}
}

// Null creates a null query value.
func (valueBuilder) Null() mdl.Value {
	return nullValue{}
}

// String creates a string query value.
func (valueBuilder) String(s string) mdl.Value {
	return stringValue{value: s}
}

// Int creates an integer query value.
func (valueBuilder) Int(i int64) mdl.Value {
	return intValue{value: i}
}

// Float creates a float query value.
func (valueBuilder) Float(f float64) mdl.Value {
	return floatValue{value: f}
}

// Point creates a point query value from longitude and latitude coordinates.
func (valueBuilder) Point(longitude float64, latitude float64) mdl.Value {
	return pointValue{value: mdl.NewPoint(longitude, latitude)}
}

// DateTime creates a date-time query value.
func (valueBuilder) DateTime(dt time.Time) mdl.Value {
	return dateTimeValue{value: dt}
}

// List creates a typed query list from the supplied values.
func (valueBuilder) List(els []mdl.Value) (mdl.Value, error) {
	return buildListValue(els)
}

// ExecuteDeserialisation deserializes data into the value type specified by t.
func (valueBuilder) ExecuteDeserialisation(
	ds mdl.Deserialiser,
	t mdl.ValueType,
	d []byte,
) (mdl.Value, error) {
	switch t {

	// DESERIALISE - NULL
	case mdl.ValueTypeNull:
		return nullValue{}, nil

	// DESERIALISE - STRING
	case mdl.ValueTypeString:
		v, err := ds.DeserialiseString(d)
		if err != nil {
			return nil, err
		}
		return stringValue{value: v}, nil

	// DESERIALISE - INT
	case mdl.ValueTypeInt:
		v, err := ds.DeserialiseInt(d)
		if err != nil {
			return nil, err
		}
		return intValue{value: v}, nil

	// DESERIALISE - FLOAT
	case mdl.ValueTypeFloat:
		v, err := ds.DeserialiseFloat(d)
		if err != nil {
			return nil, err
		}
		return floatValue{value: v}, nil

	// DESERIALISE - TIME
	case mdl.ValueTypeDateTime:
		v, err := ds.DeserialiseTime(d)
		if err != nil {
			return nil, err
		}
		return dateTimeValue{value: v}, nil

	// DESERIALISE - POINT
	case mdl.ValueTypePoint:
		v, err := ds.DeserialisePoint(d)
		if err != nil {
			return nil, err
		}
		return pointValue{value: v}, nil

	// DESERIALISE - STRING LIST
	case mdl.ValueTypeStringList:
		stringValues, err := ds.DeserialiseListString(d)
		return stringListValue{value: stringValues}, err

	// DESERIALISE - INT LIST
	case mdl.ValueTypeIntList:
		values, err := ds.DeserialiseListInt(d)
		return intListValue{value: values}, err

	// DESERIALISE - FLOAT LIST
	case mdl.ValueTypeFloatList:
		values, err := ds.DeserialiseListFloat(d)
		return floatListValue{value: values}, err
	}
	return nil, qerr.InternalErr(
		"Unable to deserialise values to type %s",
		ValueTypeString(t),
	)
}

// buildListValue creates a supported typed list from a non-empty collection of
// values.
func buildListValue(els []mdl.Value) (mdl.Value, error) {
	if len(els) == 0 {
		return nil, qerr.InternalErr("lists must contain at least one element")
	}

	listType := els[0].Type()
	switch listType {
	case mdl.ValueTypeString:
		return buildStringListValue(els)
	case mdl.ValueTypeInt:
		return buildIntListValue(els)
	case mdl.ValueTypeFloat:
		return buildFloatListValue(els)
	default:
		return nil, qerr.InternalErr(
			"lists of type '%s' are not supported",
			ValueTypeString(listType),
		)
	}
}

// ValueTypeString returns the human-readable name of a query value type.
func ValueTypeString(t mdl.ValueType) string {
	switch t {
	case mdl.ValueTypeNull:
		return "null"

	case mdl.ValueTypeString:
		return "string"

	case mdl.ValueTypeStringList:
		return "string list"

	case mdl.ValueTypeInt:
		return "integer"

	case mdl.ValueTypeIntList:
		return "integer list"

	case mdl.ValueTypeFloat:
		return "decimal"

	case mdl.ValueTypeDateTime:
		return "date/time"

	case mdl.ValueTypePoint:
		return "point"

	case mdl.ValueTypeFloatList:
		return "decimal list"

	default:
		panic("unexpected literal type received")
	}
}
