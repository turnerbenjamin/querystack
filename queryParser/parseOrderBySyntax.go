// Package queryParser contains query string parsing functionality
//
// This file contains functionality for parsing order by operations
package queryParser

import (
	qstore "github.com/turnerbenjamin/querystack/queryDataStore"
	qerr "github.com/turnerbenjamin/querystack/queryError"
	mdl "github.com/turnerbenjamin/querystack/queryModel"
)

var supportedSortDirectionOperators = map[string]mdl.SortDirectionOperator{
	"asc":  mdl.SortDirectionAsc,
	"desc": mdl.SortDirectionDesc,
}

// parseSelectOperation is responsible for parsing select operations It expects
// a simple list of comma-separated values containing at least one column
// identifier
func parseOrderByOperation(
	s qstore.QueryDataStore,
	t *Tokeniser,
	operationSeparator tokenType,
	operationTerminator tokenType,
) error {
	for {
		tkn := t.Peek()
		if tkn.Type != TokenIdentifier {
			return t.TknErr(tkn, "expected a column path but received '%s'", tkn.Value)
		}

		path, err := parsePath(t)
		if err != nil {
			return err
		}
		dir := mdl.SortDirectionAsc

		nxt := t.Peek()
		if nxt.Type == TokenSortDirectionOperator {
			sortDirectionStr := t.Next()
			sortDirection, ok := supportedSortDirectionOperators[sortDirectionStr.Value]
			if !ok {
				return qerr.InternalErr(
					"unable to match sort direction token %s to an operator",
					sortDirectionStr.Value,
				)
			}
			dir = sortDirection
		}

		if err := s.AddOrderBy(path, dir); err != nil {
			return err
		}

		nxt = t.Peek()
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
