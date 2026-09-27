// Package queryparser contains query string parsing functionality
//
// This file contains functionality for parsing select operations
package queryparser

import qstore "github.com/turnerbenjamin/querystack/querydatastore"

// parseSelectOperation is responsible for parsing select operations.
func parseSelectOperation(
	queryDataStore qstore.QueryDataStore,
	t *Tokeniser,
	operationSeparator tokenType,
	operationTerminator tokenType,
) error {
	for {
		tkn := t.Next()
		if tkn.Type != TokenIdentifier {
			return t.TknErr(tkn, "expected a column identifier but received '%s'", tkn.Value)
		}

		if err := queryDataStore.AddSelect(tkn.Value); err != nil {
			return err
		}

		nxt := t.Peek()
		switch nxt.Type {
		case operationSeparator, operationTerminator:
			return nil
		case TokenComma:
			_ = t.Next()
		default:
			return t.TknErr(nxt, "unexpected token encountered '%s'", nxt.Value)
		}
	}
}
