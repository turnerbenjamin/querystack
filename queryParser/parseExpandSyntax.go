// Package queryParser contains query string parsing functionality
//
// This file contains functionality for parsing expand operations
package queryParser

import (
	qstore "github.com/turnerbenjamin/querystack/queryDataStore"
)

// parseExpandOperation parses expand syntax and adds the operations into the
// query data store
func parseExpandOperation(
	s qstore.QueryDataStore,
	t *Tokeniser,
	operationSeparator tokenType,
	operationTerminator tokenType,
) error {
	for {
		tkn := t.Next()
		if tkn.Type != TokenIdentifier {
			return t.TknErr(tkn, "expected a column identifier but received '%s'", tkn.Value)
		}

		expandOperations, err := s.AddExpand(tkn.Value)
		if err != nil {
			return err
		}

		nxt := t.Peek()
		// If next is an opening parenthesis, parse the nested operations
		if nxt.Type == TokenParenL {
			// consume the opening parenthesis
			_ = t.Next()

			// parse operations returns operations and consumes closing parenthesis
			_, err := parseOperations(
				expandOperations,
				t,
				TokenSemiColon,
				TokenParenR,
			)
			if err != nil {
				return err
			}

			// set nxt again
			nxt = t.Peek()
		}

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
