// Package queryparser contains query string parsing functionality
//
// This file contains functionality for parsing pagination operations
package queryparser

import (
	"strconv"
	"strings"

	qstore "github.com/turnerbenjamin/querystack/querydatastore"
	qerr "github.com/turnerbenjamin/querystack/queryerror"
)

// parseSelectOperation is responsible for parsing limit operations
func parseLimitOperation(s qstore.QueryDataStore, t *Tokeniser) error {
	limitValue := t.Next()
	if limitValue.Type != TokenNumberRaw {
		return t.TknErr(limitValue, "expected a positive integer but received '%s'", limitValue.Value)
	}

	if strings.Contains(limitValue.Value, ".") {
		return t.TknErr(limitValue, "expected a positive integer but received '%s'", limitValue.Value)
	}

	limitInt, err := strconv.ParseUint(limitValue.Value, 10, 32)
	if err != nil {
		return qerr.InternalErr("unable to parse string as int: %v", err)
	}

	s.SetLimit(uint32(limitInt))
	return nil
}

// parseCountOperation is responsible for parsing count operations
func parseCountOperation(s qstore.QueryDataStore, t *Tokeniser) error {

	countValue := t.Next()
	if countValue.Type != TokenBool {
		return t.TknErr(countValue, "expected true/false but received '%s'", countValue.Value)
	}

	s.SetDoCount(countValue.Value == "true")
	return nil
}

// parsePagingTokenOperation is responsible for parsing paging tokens
func parsePagingTokenOperation(
	s qstore.QueryDataStore,
	t *Tokeniser,
	operationSeparator tokenType,
	endOfOperationsSentinal tokenType,
) error {
	tknBuilder := strings.Builder{}
	for {
		nxt := t.Peek()
		if nxt.Type == operationSeparator ||
			nxt.Type == endOfOperationsSentinal ||
			nxt.Type == TokenEOF {
			break
		}
		tkn := t.Next()
		_, err := tknBuilder.WriteString(tkn.Value)
		if err != nil {
			return qerr.InternalErr("unable to add token to token builder: %v", err)
		}

	}

	s.SetPagingToken(tknBuilder.String())
	if _, exists := s.PagingToken(); !exists {
		return qerr.SyntaxErr("pagingToken must be provided")
	}
	return nil
}
