// Package queryPlanner validates and plans parsed queries before execution.
//
// It applies query defaults and limits, ensures deterministic ordering,
// configures expanded queries, and prepares the query for cursor-based
// pagination.
package queryPlanner

import (
	tkns "github.com/turnerbenjamin/querystack/paginationTokens"
	qstore "github.com/turnerbenjamin/querystack/queryDataStore"
	qerr "github.com/turnerbenjamin/querystack/queryError"
	mdl "github.com/turnerbenjamin/querystack/queryModel"
)

// PagingTokenBuilder builds and parses paging tokens used for cursor-based
// pagination.
type PagingTokenBuilder interface {
	// BuildToken generates a new pagination token
	BuildToken(
		queryDataStore qstore.QueryDataStore,
		lastRecord mdl.TableModel,
	) (string, error)

	// ParseToken parses a pagination token
	ParseToken(token string, valueBuilder mdl.ValueBuilder) (*tkns.PagingToken, error)
}

// QueryParser parses a query string into a QueryDataStore.
type QueryParser interface {
	// Parse is used to parse a query string and add its operations to a query
	// data store
	Parse(
		queryString string,
		queryDataStore qstore.QueryDataStore,
	) (uint8, error)
}

// PlanQuery parses and plans a query according to the supplied configuration.
// It initialises the query data store, processes any paging token, applies
// query planning rules, and validates the resulting query against configured
// record limits.
func PlanQuery(
	config mdl.QueryConfig,
	queryString string,
	queryParser QueryParser,
	pagingTokenBuilder PagingTokenBuilder,
	valueBuilder mdl.ValueBuilder,
	rootResource mdl.Resource,
	accessPolicy mdl.AccessPolicy,
) (qstore.QueryDataStore, error) {
	// Initialise query data store
	s, operationCount, err := initStore(
		queryString,
		queryParser,
		valueBuilder,
		rootResource,
		accessPolicy,
	)
	if err != nil {
		return nil, err
	}

	// Parse Token
	var pagingToken *tkns.PagingToken
	if tknStr, exists := s.PagingToken(); exists {
		pagingToken, err = pagingTokenBuilder.ParseToken(
			tknStr,
			valueBuilder,
		)
		if err != nil {
			return nil, err
		}

		if operationCount > 1 {
			return nil, qerr.SyntaxErr(
				"pagingToken must not be used alongside other operations",
			)
		}

		if rootResource.GetMetadata().Name != pagingToken.ResourceName {
			return nil, qerr.SyntaxErr(
				"invalid paging token: this token can only be used with %s",
				pagingToken.ResourceName,
			)
		}

		s, _, err = initStore(
			pagingToken.QueryString,
			queryParser,
			valueBuilder,
			rootResource,
			accessPolicy,
		)
		if err != nil {
			return nil, err
		}
	}

	// Configure the query
	totalRecordCount, err := planQuery(s, config, pagingToken)
	if err != nil {
		return nil, err
	}

	// Validate Query Size
	if totalRecordCount > config.MaxRecordsPerPage {
		return nil, qerr.SyntaxErr(
			"invalid query: this query could return up to %d records; the limit is %d",
			totalRecordCount,
			config.MaxRecordsPerPage,
		)
	}

	return s, err
}

// initStore creates a query data store for the supplied query and parses the
// query string into it.
//
// The returned operation count represents the number of operations parsed
// from the query.
func initStore(
	queryString string,
	queryParser QueryParser,
	valueBuilder mdl.ValueBuilder,
	rootResource mdl.Resource,
	accessPolicy mdl.AccessPolicy,
) (qstore.QueryDataStore, uint8, error) {
	s, err := qstore.NewQueryDataStore(
		queryString,
		rootResource,
		accessPolicy,
		valueBuilder,
	)
	if err != nil {
		return nil, uint8(0), err
	}

	// Parse query string into data store
	operationCount, err := queryParser.Parse(queryString, s)
	if err != nil {
		return nil, uint8(0), err
	}

	return s, operationCount, nil
}

// planQuery applies planning rules to a query data store and its nested
// expansions.
//
// It adds default operations, ensures deterministic ordering, configures
// operations required for cursor pagination, applies system limits, plans
// nested expansions, and adds a cursor filter when required. The returned
// record count represents the maximum number of records the query may
// produce.
func planQuery(
	s qstore.QueryDataStore,
	config mdl.QueryConfig,
	pagingToken *tkns.PagingToken,
) (uint32, error) {
	totalRecordCount := uint32(0)

	// Add default operations as required
	if err := addSystemDefaults(s, config); err != nil {
		return 0, err
	}
	totalRecordCount += s.Limit()

	// Ensure deterministic order
	if err := ensureDeterministicOrdering(s); err != nil {
		return 0, err
	}

	// Add any required systems selects/expands required for cursor pagination
	if err := addRequiredOperationsForCursorPagination(s); err != nil {
		return 0, err
	}

	// Set system limit for pagination record
	setSystemLimit(s)

	// Configure expanded queries
	for _, expansion := range s.Expands() {
		if expansion.QueryData.Depth() > config.MaxDepth {
			return 0, qerr.SyntaxErr(
				"invalid query: exceeds the maximum expansion depth of %d",
				config.MaxDepth,
			)
		}

		nestedRecordCount, err := planQuery(expansion.QueryData, config, nil)
		if err != nil {
			return 0, err
		}
		if expansion.TraversalStep.Relationship.Type == mdl.RelationshipManyToOne {
			totalRecordCount += 1
		} else {
			totalRecordCount *= nestedRecordCount
		}
	}

	// Add cursor filter where required
	if err := addCursorFilter(s, pagingToken); err != nil {
		return 0, err
	}

	return totalRecordCount, nil
}

// addSystemDefaults applies default query operations where they have not been
// explicitly supplied by the caller.
//
// This includes selecting accessible columns, applying the default page size,
// and ordering by the root resource's primary key.
func addSystemDefaults(s qstore.QueryDataStore, config mdl.QueryConfig) error {
	rootResourceMetadata := s.RootResourceMetadata()

	p := s.ProjectionNode().Projection
	if p.IsEmpty() {
		if err := addDefaultSelects(s); err != nil {
			return err
		}
	}

	if s.Limit() == 0 {
		s.SetLimit(config.DefaultPageSize)
	}

	if s.OrderByLen() == 0 {
		s.AddOrderBy(
			rootResourceMetadata.PrimaryKeyColumn.Name,
			mdl.SortDirectionAsc,
		)

	}

	return nil
}

// addDefaultSelects selects all columns that the query's access policy permits
// the caller to access.
//
// An error is returned if the access policy prevents access to every column on
// the root resource.
func addDefaultSelects(s qstore.QueryDataStore) error {
	// Add all columns the user can access to the table
	tableAccessPolicy := s.RootResourceAccessPolicy()
	rootResourceMetadata := s.RootResourceMetadata()

	i := 0
	for _, col := range rootResourceMetadata.Columns {
		canAccessColumn, err := tableAccessPolicy.CanAccessColumn(col.Name)
		if err != nil {
			return qerr.InternalErr("unable to validate column access: %w", err)
		}

		if !canAccessColumn {
			continue
		}

		if err := s.AddSelect(col.Name); err != nil {
			return err
		}
		i++
	}

	if i == 0 {
		return qerr.InternalErr("invalid access policy, the user does not have access to any columns")
	}
	return nil
}

// ensureDeterministicOrdering ensures that the query ordering includes the root
// resource's primary key.
//
// The primary key is appended as an ascending ordering rule when it is not
// already present, providing a deterministic ordering for pagination.
func ensureDeterministicOrdering(s qstore.QueryDataStore) error {
	primaryKeyField := s.RootResourceMetadata().PrimaryKeyColumn.Name

	// exit early if query is already sorted by the primary key field
	for rule := range s.OrderBy() {
		if len(rule.ResolvedColumn.ResolvedPath.Steps) == 0 &&
			rule.ResolvedColumn.Metadata.Name == primaryKeyField {
			return nil
		}
	}
	return s.AddOrderBy(primaryKeyField, mdl.SortDirectionAsc)
}

// addRequiredOperationsForCursorPagination adds system selects required to
// support cursor-based pagination.
//
// Root-level ordering columns are added as system selects, while ordering
// columns on related resources cause the required nested selects or expansions
// to be added.
func addRequiredOperationsForCursorPagination(s qstore.QueryDataStore) error {
	if !s.IsTopLevelQuery() {
		return nil
	}

	for rule := range s.OrderBy() {
		col := rule.ResolvedColumn
		// If column is on the root resource just add a system select
		if len(col.ResolvedPath.Steps) == 0 {
			if err := s.AddSystemSelect(col.Metadata.Name); err != nil {
				return err
			}
		} else {
			// If the column is nested add a nested select
			if err := addNestedSystemSelect(s, col); err != nil {
				return err
			}
		}

	}
	return nil
}

// addNestedSystemSelect adds a system select for a resolved column on a
// related resource.
//
// Existing expansions are reused where possible; otherwise, the required
// system expansion and nested selects are created.
func addNestedSystemSelect(s qstore.QueryDataStore, resolvedColumn mdl.ResolvedColumn) error {
	// Shift first step from the array
	nextStep := resolvedColumn.ResolvedPath.Steps[0]

	// Check for existing nested operation
	nestedOperation, exists := s.GetExpansionByRelationshipId(nextStep.Relationship.ColumnName)
	if exists {
		// bring resolved column path forward to root from expanded entity
		remainingSteps := resolvedColumn.ResolvedPath.Steps[1:]
		resolvedColumn.ResolvedPath.Steps = remainingSteps

		// If an existing nested query is found for the orderby column, add a
		// system select to that query for the column and return
		if len(remainingSteps) == 0 {
			return nestedOperation.QueryData.AddSystemSelect(resolvedColumn.Metadata.Name)
		} else {
			// If a nested query is found, but it is not the final resource, recall
			// the current function against the nested query with the remaining path
			return addNestedSystemSelect(s, resolvedColumn)
		}
	} else {
		// If a nested query is not found one, we need to construct a system
		// expand and select the required column
		return addSystemExpand(s, resolvedColumn)
	}
}

// addSystemExpand creates the nested system expansions and selects required
// to access a resolved column on a related resource.
//
// Expansions are created from the deepest relationship back towards the
// current query so that each required intermediate resource is available.
func addSystemExpand(
	s qstore.QueryDataStore,
	resolvedColumn mdl.ResolvedColumn,
) error {
	pathSteps := resolvedColumn.ResolvedPath.Steps
	totalSteps := len(pathSteps)

	currentStore := s
	for i := totalSteps - 1; i >= 0; i-- {
		step := pathSteps[i]

		nestedOperations, err := currentStore.AddSystemExpand(step.Relationship.FromColumn.Name)
		if err != nil {
			return err
		}

		nestedResource := nestedOperations.RootResourceMetadata()

		if i == totalSteps-1 {
			if err := nestedOperations.AddSelect(resolvedColumn.Metadata.Name); err != nil {
				return err
			}
		} else {
			if err := nestedOperations.AddSelect(nestedResource.Name); err != nil {
				return err
			}
		}
		currentStore = nestedOperations
	}
	return nil
}

// setSystemLimit sets the internal record limit used when executing the query.
//
// Top-level queries receive one additional record beyond the user-requested
// limit so that the planner can determine whether another page exists.
func setSystemLimit(s qstore.QueryDataStore) {
	var userLimit uint32 = s.Limit()
	systemLimit := uint64(userLimit)

	// Set system limit as limit + 1 for top-level queries so that we can
	// check if there are additional records
	if s.IsTopLevelQuery() {
		systemLimit++
	}

	s.SetSystemLimit(systemLimit)
}

// addCursorFilter adds a filter that restricts results to records occurring
// after the position represented by a paging token.
//
// The generated filter follows the query's ordering rules and is combined
// with any existing filter using a logical AND.
func addCursorFilter(
	s qstore.QueryDataStore,
	paginationToken *tkns.PagingToken,
) error {
	if paginationToken == nil {
		return nil
	}

	cursorValues := paginationToken.CursorValues

	// validate that order by set and order by can be zipped to cursor values
	if s.OrderByLen() == 0 || s.OrderByLen() != len(cursorValues) {
		return qerr.InternalErr(
			"expected at least one orderby rule with one value expression" +
				"for each rule",
		)
	}

	b := s.FilterExpressionBuilder()
	// Build the cursor filter
	var cursorFilter mdl.FilterExpression
	i := 0
	for rule := range s.OrderBy() {
		cursorValue := cursorValues[i]

		if rule.Direction == mdl.SortDirectionDesc &&
			cursorValue.Type() == mdl.ValueTypeNull {
			continue
		}

		ruleExpression, err := getCursorFilterComparisonOperator(
			rule,
			cursorValue,
			b,
		)
		if err != nil {
			return err
		}

		if ruleExpression == nil {
			return qerr.InternalErr("unable to generate cursor filter")
		}

		j := 0
		for previousRule := range s.OrderBy() {
			if j == i {
				break
			}

			previousValue := cursorValues[j]
			previousRuleFilter, err := b.NewComparisonExpressionFromResolvedColumn(
				previousRule.ResolvedColumn,
				mdl.ComparisonEq,
				previousValue,
			)
			if err != nil {
				return err
			}

			ruleExpression, err = b.NewLogicalExpression(
				ruleExpression,
				mdl.LogicalAnd,
				previousRuleFilter,
			)
			if err != nil {
				return err
			}
			j++
		}

		if cursorFilter == nil {
			cursorFilter = ruleExpression
		} else {
			cursorFilter, err = b.NewLogicalExpression(
				cursorFilter,
				mdl.LogicalOr,
				ruleExpression,
			)
			if err != nil {
				return err
			}
		}
		i++
	}

	if s.FilterExpression() == nil {
		s.SetFilterExpression(cursorFilter)
		return nil
	}

	finalFilter, err := b.NewLogicalExpression(
		s.FilterExpression(),
		mdl.LogicalAnd,
		cursorFilter,
	)
	if err != nil {
		return err
	}
	s.SetFilterExpression(finalFilter)

	return nil
}

// getCursorFilterComparisonOperator creates the comparison expression used
// to advance a cursor according to its ordering direction and cursor value.
//
// The comparison accounts for the ordering direction and the position of
// null values within the ordering.
func getCursorFilterComparisonOperator(
	rule mdl.SortingRule,
	value mdl.Value,
	b qstore.FilterExpressionBuilder,
) (mdl.FilterExpression, error) {
	if rule.Direction == mdl.SortDirectionAsc {
		if value.Type() == mdl.ValueTypeNull {
			// Ascending logic for null value
			return b.NewComparisonExpressionFromResolvedColumn(
				rule.ResolvedColumn,
				mdl.ComparisonNe,
				value,
			)
		} else {
			// Ascending logic for non-null value
			return b.NewComparisonExpressionFromResolvedColumn(
				rule.ResolvedColumn,
				mdl.ComparisonGt,
				value,
			)
		}
	} else {
		if value.Type() == mdl.ValueTypeNull {
			// Descending logic for null value, null is already the last value
			// so do not add a filter
			return nil, nil
		}
		l, err := b.NewComparisonExpressionFromResolvedColumn(
			rule.ResolvedColumn,
			mdl.ComparisonLt,
			value,
		)
		if err != nil {
			return nil, err
		}

		r, err := b.NewComparisonExpressionFromResolvedColumn(
			rule.ResolvedColumn,
			mdl.ComparisonEq,
			b.ValueBuilder().Null(),
		)
		if err != nil {
			return nil, err
		}
		return b.NewLogicalExpression(l, mdl.LogicalOr, r)
	}
}
