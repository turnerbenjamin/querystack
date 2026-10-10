package querymodel

// DbType represents the normalized database type
type DbType string

const (
	// DbTypeString represents a string database value
	DbTypeString DbType = "string"

	// DbTypeInt represents an integer database value
	DbTypeInt DbType = "int"

	// DbTypeFloat represents a floating-point database value
	DbTypeFloat DbType = "float"

	// DbTypePoint represents a geographic point database value
	DbTypePoint DbType = "point"

	// DbTypeDateType represents a date and time database value
	DbTypeDateTime DbType = "date/time"

	// DbTypeEnum represents an enum value
	DbTypeEnum DbType = "enum"
)

type operatorSupport struct {
	singleValue     map[ComparisonOperator]struct{}
	collectionValue map[ComparisonOperator]struct{}
}

var ComparisonSupport = map[DbType]operatorSupport{
	DbTypeString: {
		singleValue: map[ComparisonOperator]struct{}{
			ComparisonEq:            {},
			ComparisonNe:            {},
			ComparisonContains:      {},
			ComparisonStartsWith:    {},
			ComparisonEndsWith:      {},
			ComparisonNotContains:   {},
			ComparisonNotStartsWith: {},
			ComparisonNotEndsWith:   {},
			ComparisonGe:            {},
			ComparisonGt:            {},
			ComparisonLt:            {},
			ComparisonLe:            {},
		},
		collectionValue: map[ComparisonOperator]struct{}{
			ComparisonIn:    {},
			ComparisonNotIn: {},
		},
	},
	DbTypeInt: {
		singleValue: map[ComparisonOperator]struct{}{
			ComparisonEq: {},
			ComparisonNe: {},
			ComparisonGt: {},
			ComparisonGe: {},
			ComparisonLt: {},
			ComparisonLe: {},
		},
		collectionValue: map[ComparisonOperator]struct{}{
			ComparisonIn:    {},
			ComparisonNotIn: {},
		},
	},
	DbTypeFloat: {
		singleValue: map[ComparisonOperator]struct{}{
			ComparisonEq: {},
			ComparisonNe: {},
			ComparisonGt: {},
			ComparisonGe: {},
			ComparisonLt: {},
			ComparisonLe: {},
		},
		collectionValue: map[ComparisonOperator]struct{}{
			ComparisonIn:    {},
			ComparisonNotIn: {},
		},
	},
	DbTypePoint: {
		singleValue:     map[ComparisonOperator]struct{}{},
		collectionValue: map[ComparisonOperator]struct{}{},
	},
	DbTypeDateTime: {
		singleValue: map[ComparisonOperator]struct{}{
			ComparisonEq: {},
			ComparisonNe: {},
			ComparisonGt: {},
			ComparisonGe: {},
			ComparisonLt: {},
			ComparisonLe: {},
		},
		collectionValue: map[ComparisonOperator]struct{}{},
	},
	DbTypeEnum: {
		singleValue: map[ComparisonOperator]struct{}{
			ComparisonEq: {},
			ComparisonNe: {},
		},
		collectionValue: map[ComparisonOperator]struct{}{
			ComparisonIn:    {},
			ComparisonNotIn: {},
		},
	},
}

func DbTypeSupportsSingleValueOperation(t DbType, op ComparisonOperator) bool {
	supportedOps, exists := ComparisonSupport[t]
	if !exists {
		return false
	}

	_, supported := supportedOps.singleValue[op]
	return supported
}

func DbTypeSupportsCollectionValueOperation(t DbType, op ComparisonOperator) bool {
	supportedOps, exists := ComparisonSupport[t]
	if !exists {
		return false
	}

	_, supported := supportedOps.collectionValue[op]
	return supported
}
