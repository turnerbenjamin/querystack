// Package queryParser contains query string parsing functionality
//
// This file contains functionality for parsing literal values
package queryParser

import (
	"strconv"
	"strings"
	"time"

	qerr "github.com/turnerbenjamin/querystack/queryError"
	mdl "github.com/turnerbenjamin/querystack/queryModel"
)

// parseValue parses the next token as a literal value, supporting null,
// strings, numbers, lists, and recognised reference types such as points
// and datetimes.
func parseValue(t *Tokeniser, v mdl.ValueBuilder) (mdl.Value, error) {
	tkn := t.Next()

	switch tkn.Type {

	case TokenNull:
		return v.Null(), nil

	case TokenStringRaw:
		return parseStringValue(t, tkn, v)

	case TokenNumberRaw:
		doNegate := false
		return parseNumberValue(tkn, v, doNegate)

	case TokenParenL:
		return parseList(t, v)

	case TokenHyphen:
		if t.Peek().Type == TokenNumberRaw {
			//consume hyphen
			tkn := t.Next()
			// parse number and negate
			doNegate := true
			return parseNumberValue(tkn, v, doNegate)
		}

	case TokenIdentifier:
		return tryParseReferenceType(t, tkn, v)

	}

	return nil, t.TknErr(tkn, "unexpected value %s", tkn.Value)
}

// parseStringValue validates and parses a raw quoted string token into a string
// value. Both single and double-quoted strings are supported
func parseStringValue(t *Tokeniser, raw token, v mdl.ValueBuilder) (mdl.Value, error) {
	if raw.Type != TokenStringRaw {
		return nil, t.TknErr(raw, "unexpected token type received: '%s'", raw.Value)
	}

	if len(raw.Value) < 2 {
		return nil, qerr.SyntaxErr("raw string token should be at least 2 characters")
	}

	openingQuotationMark := raw.Value[0]
	closingQuotationMark := raw.Value[len(raw.Value)-1]

	if openingQuotationMark != '\'' && openingQuotationMark != '"' {
		return nil, qerr.SyntaxErr("raw string token should be prefixed with a ' or \"")
	}

	if closingQuotationMark != openingQuotationMark {
		return nil, qerr.SyntaxErr("unterminated string literal %s", raw.Value)
	}

	strValue := raw.Value[1 : len(raw.Value)-1]
	return v.String(strValue), nil
}

// parseList parses a list literal beginning at the current token position.
// List literals must contain at least one element and all elements must have
// the same value type.
func parseList(t *Tokeniser, v mdl.ValueBuilder) (mdl.Value, error) {
	first, err := parseValue(t, v)
	if err != nil {
		return nil, err
	}

	if first == nil {
		return nil, qerr.SyntaxErr("a list literal must have at least 1 element")
	}

	els, err := parseListElements(t, v, first)
	if err != nil {
		return nil, err
	}

	return v.List(els)
}

// parseListElements parses the remaining elements of a list after its first
// element has been parsed. Elements must have the same type as the first
// element, and the closing parenthesis terminates the list.
func parseListElements(
	t *Tokeniser,
	v mdl.ValueBuilder,
	firstEl mdl.Value,
) ([]mdl.Value, error) {
	els := []mdl.Value{firstEl}
	for {
		if t.Peek().Type == TokenParenR {
			_ = t.Next()
			return els, nil
		}

		if t.Peek().Type == TokenComma {
			_ = t.Next()
			continue
		}

		v, err := parseValue(t, v)
		if err != nil {
			return nil, err
		}

		if v.Type() != firstEl.Type() {
			return nil, qerr.SyntaxErr("mixed type lists are not supported")
		}

		els = append(els, v)
	}
}

// parseNumberValue parses a raw numeric token as either an integer or a
// floating-point value. When doNegate is true, the resulting value is
// negated.
func parseNumberValue(
	tkn token,
	v mdl.ValueBuilder,
	doNegate bool,
) (mdl.Value, error) {
	dpCount := 0
	for _, c := range tkn.Value {
		if c == '.' {
			dpCount++
		}
		if dpCount > 1 {
			break
		}
	}

	multiplier := 0
	if doNegate {
		multiplier = -1
	}

	switch dpCount {
	case 0:
		i, err := strconv.Atoi(tkn.Value)
		if err != nil {
			return nil, qerr.InternalErr("unable to convert string to int: %v", err)
		}

		return v.Int(int64(i) * int64(multiplier)), nil
	case 1:
		f, err := strconv.ParseFloat(tkn.Value, 64)
		if err != nil {
			return nil, qerr.InternalErr("unable to convert string to float: %v", err)
		}
		return v.Float(f * float64(multiplier)), nil
	default:
		return nil, qerr.InternalErr("invalid number value: %s", tkn.Value)
	}
}

// tryParseReferenceType identifies and parses a supported reference-type
// literal following an identifier. Supported types include point and
// datetime.
func tryParseReferenceType(
	t *Tokeniser,
	tkn token,
	v mdl.ValueBuilder,
) (mdl.Value, error) {
	nxtTkn := t.Peek()
	if nxtTkn.Type != TokenParenL {
		return nil, t.TknErr(tkn, "unexpected value %s", tkn.Value)
	}

	typeId := strings.ToLower(tkn.Value)
	switch typeId {
	case "point":
		tkn = t.Next()
		return tryParsePoint(t, tkn, v)
	case "datetime":
		tkn := t.Next()
		return tryParseDateTime(t, tkn, v)
	default:
		return nil, t.TknErr(tkn, "unexpected value %s", tkn.Value)
	}
}

// tryParsePoint parses a point literal containing longitude and latitude
// coordinates and constructs the corresponding point value.
func tryParsePoint(t *Tokeniser, tkn token, v mdl.ValueBuilder) (mdl.Value, error) {
	// Expect first token to be an opening parenthesis
	if tkn.Type != TokenParenL {
		return nil, t.TknErr(tkn, "unexpected value %s", tkn.Value)
	}
	tkn = t.Next()

	// Expect a longitude value
	if tkn.Type != TokenNumberRaw {
		return nil, t.TknErr(tkn, "expected a longitude value but received '%s'", tkn.Value)
	}
	longitude, err := strconv.ParseFloat(tkn.Value, 64)
	if err != nil {
		return nil, t.TknErr(tkn, "unable to parse longitude value to a float: '%s'", tkn.Value)
	}

	// Expect a comma separator
	tkn = t.Next()
	if tkn.Type != TokenComma {
		return nil, t.TknErr(tkn, "expected a comma separator but received '%s'", tkn.Value)
	}
	tkn = t.Next()

	//Expect latitude value
	if tkn.Type != TokenNumberRaw {
		return nil, t.TknErr(tkn, "expected a latitude value but received '%s'", tkn.Value)
	}
	latitude, err := strconv.ParseFloat(tkn.Value, 64)
	if err != nil {
		return nil, t.TknErr(tkn, "unable to parse latitude value to a float: '%s'", tkn.Value)
	}

	// Expect a closing parenthesis
	tkn = t.Next()
	if tkn.Type != TokenParenR {
		return nil, t.TknErr(tkn, "expected a closing parenthesis but received '%s'", tkn.Value)
	}
	// consume the closing parenthesis
	t.Next()

	return v.Point(longitude, latitude), nil
}

// tryParseDateTime parses a datetime literal and converts its contents from
// RFC 3339 format into a datetime value.
func tryParseDateTime(t *Tokeniser, tkn token, v mdl.ValueBuilder) (mdl.Value, error) {
	// Expect first token to be an opening parenthesis
	if tkn.Type != TokenParenL {
		return nil, t.TknErr(tkn, "unexpected value %s", tkn.Value)
	}
	tkn = t.Next()

	dtBuilder := strings.Builder{}
	for {
		// if closing parenthesis, consume and break
		if tkn.Type == TokenParenR {
			t.Next()
			break
		}

		// if EOF reached freak out
		if tkn.Type == TokenEOF {
			return nil, t.TknErr(
				tkn,
				"expected closing parenthesis but received '%s'",
				tkn.Value,
			)
		}

		dtBuilder.WriteString(tkn.Value)
		tkn = t.Next()
	}

	d, err := time.Parse(time.RFC3339, dtBuilder.String())
	if err != nil {
		return nil, t.TknErr(
			tkn,
			"unable to parse datetime - specify datetime values in RFC 3339 format",
		)
	}

	return v.DateTime(d), nil
}
