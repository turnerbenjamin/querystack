// Package queryParser contains query string parsing functionality
//
// This file contains functionality for parsing filter operations
package queryParser

import (
	"strings"

	qstore "github.com/turnerbenjamin/querystack/queryDataStore"
	mdl "github.com/turnerbenjamin/querystack/queryModel"
)

// parseFilterOperation parses a complete filter operation from the tokeniser
// and stores the resulting filter expression in the query data store.
func parseFilterOperation(
	s qstore.QueryDataStore,
	t *Tokeniser,
	operationSeparator tokenType,
	endOfOperationsSentinal tokenType,
) error {
	b := s.FilterExpressionBuilder()
	expression, err := parseFilterExpression(b, t)
	if err != nil {
		return err
	}

	nxtTkn := t.Peek()
	if nxtTkn.Type != operationSeparator && nxtTkn.Type != endOfOperationsSentinal {
		return t.TknErr(
			nxtTkn,
			"expected end of filter value but received '%s'",
			nxtTkn.Value,
		)
	}

	s.SetFilterExpression(expression)
	return nil
}

// parseFilterExpression parses a filter expression.
func parseFilterExpression(
	b qstore.FilterExpressionBuilder,
	t *Tokeniser,
) (mdl.FilterExpression, error) {
	return parseOr(b, t)
}

// parseOr parses a sequence of filter expressions joined by the logical OR
// operator and constructs a corresponding logical expression tree.
func parseOr(
	b qstore.FilterExpressionBuilder,
	t *Tokeniser,
) (mdl.FilterExpression, error) {
	left, err := parseAnd(b, t)
	if err != nil {
		return nil, err
	}

	for {
		tkn := t.Peek()
		if tkn.Type != TokenLogicalOperator || tkn.Value != "or" {
			break
		}
		t.Next()

		right, err := parseAnd(b, t)
		if err != nil {
			return nil, err
		}

		if left, err = b.NewLogicalExpression(
			left,
			mdl.LogicalOr,
			right,
		); err != nil {
			return nil, err
		}
	}

	return left, nil
}

// parseAnd parses a sequence of filter expressions joined by the logical AND
// operator and constructs a corresponding logical expression tree.
func parseAnd(
	b qstore.FilterExpressionBuilder,
	t *Tokeniser,
) (mdl.FilterExpression, error) {
	left, err := parsePrimary(b, t)
	if err != nil {
		return nil, err
	}

	for {
		tkn := t.Peek()
		if tkn.Type != TokenLogicalOperator || tkn.Value != "and" {
			break
		}
		t.Next()

		right, err := parsePrimary(b, t)
		if err != nil {
			return nil, err
		}

		if left, err = b.NewLogicalExpression(
			left,
			mdl.LogicalAnd,
			right,
		); err != nil {
			return nil, err
		}
	}

	return left, nil
}

// parsePrimary parses a primary filter expression.
//
// Primary expressions are either parenthesised filter expressions or a path
// followed by a comparison or collection operator.
func parsePrimary(
	b qstore.FilterExpressionBuilder,
	t *Tokeniser,
) (mdl.FilterExpression, error) {
	tkn := t.Peek()

	switch tkn.Type {

	case TokenParenL:
		// consume opening parenthesis
		t.Next()

		expression, err := parseFilterExpression(b, t)
		if err != nil {
			return nil, err
		}

		// consume closing parenthesis
		close := t.Next()
		if close.Type != TokenParenR {
			return nil, t.TknErr(close, "expected ')' but received '%s'", close.Value)
		}

		return expression, nil

	default:
		path, err := parsePath(t)
		if err != nil {
			return nil, err
		}

		nxtTkn := t.Peek()

		if nxtTkn.Type == TokenCollectionOperator {
			return parseCollectionOperator(b, t, path)
		}
		return parseComparison(b, t, path)
	}
}

// parseComparison parses a comparison expression consisting of a resource
// path, comparison operator, and value, and constructs the corresponding
// filter expression.
func parseComparison(
	b qstore.FilterExpressionBuilder,
	t *Tokeniser,
	columnPath string,
) (mdl.FilterExpression, error) {
	operator := t.Next()
	if operator.Type != TokenComparisonOperator {
		return nil, t.TknErr(
			operator,
			"expected comparison operator, got %s",
			operator.Value,
		)
	}

	op, err := parseComparisonOperator(t, operator)
	if err != nil {
		return nil, err
	}

	valueBuilder := b.ValueBuilder()
	right, err := parseValue(t, valueBuilder)
	if err != nil {
		return nil, err
	}

	return b.NewComparisonExpression(columnPath, op, right)
}

// parseComparisonOperator resolves a comparison operator token to its
// corresponding model comparison operator.
func parseComparisonOperator(
	t *Tokeniser, tkn token) (mdl.ComparisonOperator, error) {
	operator, ok := mdl.SupportedComparisonOperators[tkn.Value]
	if !ok {
		return "", t.TknErr(
			tkn,
			"unknown comparison operator %s",
			tkn.Value,
		)
	}
	return operator, nil
}

// parseCollectionOperator parses a collection filter expression consisting of
// a resource path, collection operator, and nested filter expression.
func parseCollectionOperator(
	b qstore.FilterExpressionBuilder,
	t *Tokeniser,
	resourcePath string,
) (mdl.FilterExpression, error) {

	tkn := t.Next()
	operator, exists := mdl.SupportedCollectionOperators[tkn.Value]
	if !exists {
		return nil, t.TknErr(
			tkn,
			"expected collection operator but received '%s'",
			tkn.Value,
		)
	}

	if tkn = t.Next(); tkn.Type != TokenParenL {
		return nil, t.TknErr(tkn, "expected '(' but received '%s'", tkn.Value)
	}

	filterExpression, err := parseFilterExpression(b, t)
	if err != nil {
		return nil, err
	}

	if tkn := t.Next(); tkn.Type != TokenParenR {
		return nil, t.TknErr(tkn, "expected ')' but received '%s'", tkn.Value)
	}

	return b.NewCollectionExpression(
		resourcePath,
		operator,
		filterExpression,
	)
}

// parsePath parses a contiguous resource path from the tokeniser.
//
// Whitespace immediately before a path is ignored, while whitespace within
// the path terminates parsing.
func parsePath(t *Tokeniser) (string, error) {
	// Generally, space tokens are skipped, however, paths must be contiguous
	pathBuilder := strings.Builder{}
	i := 0

outer:
	for {
		// Ignore whitespace at the start of the path only - paths should be
		// contiguous
		tkn := t.NextIncSpace()
		if tkn.Type == TokenSpace && i == 0 {
			continue
		}
		i++

		switch tkn.Type {
		// Write slashes to the path
		case TokenSlash:
			pathBuilder.WriteString(tkn.Value)

		// Write token identifiers to the path
		case TokenIdentifier, TokenLogicalOperator, TokenComparisonOperator, TokenCollectionOperator:
			pathBuilder.WriteString(tkn.Value)

		// For all other token types exit
		default:
			break outer
		}

		nxt := t.Peek()
		if nxt.Type != TokenIdentifier &&
			nxt.Type != TokenLogicalOperator &&
			nxt.Type != TokenComparisonOperator &&
			nxt.Type != TokenCollectionOperator &&
			nxt.Type != TokenSlash {
			break
		}

		// Guard against consuming valid collection operators
		if nxt.Type == TokenSlash &&
			t.PeekN(2).Type == TokenCollectionOperator &&
			t.PeekN(3).Type == TokenParenL {
			//consume the slash without writing to the path and break
			_ = t.Next()
			break
		}
	}
	return pathBuilder.String(), nil
}
