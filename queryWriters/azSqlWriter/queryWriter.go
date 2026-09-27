// Package azSqlWriter generates Microsoft SQL Server query statements from
// planned query data stores.
//
// It translates query selections, expansions, filters, ordering, and pagination
// into parameterised SQL statements.
package azSqlWriter

import (
	"fmt"
	"time"

	querybuilder "github.com/turnerbenjamin/querystack/queryBuilder"
	qstore "github.com/turnerbenjamin/querystack/queryDataStore"
	qerr "github.com/turnerbenjamin/querystack/queryError"
	mdl "github.com/turnerbenjamin/querystack/queryModel"
	"github.com/turnerbenjamin/querystack/relationships"
)

// aliasedResource associates resource metadata with the SQL table alias used
// to reference that resource within a query.
type aliasedResource struct {
	resource mdl.TableMetadata
	alias    string
}

// queryWriter generates SQL statements for a query data store.
type queryWriter struct {
	queryDataStore qstore.QueryDataStore
	aliasStore     relationships.AliasStore
}

// NewQueryWriter creates a query writer for the supplied query data store.
func NewQueryWriter(s qstore.QueryDataStore) mdl.QueryWriter {
	return &queryWriter{
		queryDataStore: s,
		aliasStore:     s.GetAliasStore(),
	}
}

// newNestedQueryWriter creates a query writer for a nested query data store.
//
// The nested writer uses the alias store associated with the nested query so
// that generated SQL references the correct resource aliases.
func (w queryWriter) newNestedQueryWriter(s qstore.QueryDataStore) *queryWriter {
	return &queryWriter{
		queryDataStore: s,
		aliasStore:     s.GetAliasStore(),
	}
}

// WriteQueryStatement generates the SQL statement used to execute the query,
// including selected columns, expansions, joins, filters, ordering, and the
// JSON response clause.
func (w queryWriter) WriteQueryStatement() (mdl.QueryStatement, error) {
	b := &querybuilder.Builder{}

	err := w.writeQuery(b)
	if err != nil {
		return mdl.QueryStatement{}, err
	}

	return mdl.QueryStatement{
		Statement: b.String(),
		Args:      b.Args(),
	}, nil
}

// WriteCountStatement generates a SQL statement that counts the total number of
//
//	records matching the query's conditions
func (w queryWriter) WriteCountStatement() (mdl.QueryStatement, error) {
	b := &querybuilder.Builder{}

	err := w.buildCountStatement(b)
	if err != nil {
		return mdl.QueryStatement{}, err
	}

	return mdl.QueryStatement{
		Statement: b.String(),
		Args:      b.Args(),
	}, nil
}

// writeQuery writes the complete SQL query to the supplied query builder.
func (w *queryWriter) writeQuery(b *querybuilder.Builder) error {
	err := w.writeSelectAndExpandStatement(b)
	if err != nil {
		return err
	}

	w.writeFromStatement(b)

	w.writeJoins(b, w.queryDataStore.JoinCollection())

	err = w.writeFilterStatement(b)
	if err != nil {
		return err
	}

	err = w.writeOrderByStatement(b)
	if err != nil {
		return err
	}

	// request json response
	b.Write("FOR JSON PATH")
	if w.queryDataStore.IsTopLevelQuery() {
		b.Write(";")
	}

	return nil
}

// buildCountStatement writes a SQL COUNT statement for the current query to
// the supplied query builder.
func (w *queryWriter) buildCountStatement(b *querybuilder.Builder) error {
	b.Write("SELECT COUNT(*) AS total_count ")
	w.writeFromStatement(b)

	err := w.writeFilterStatement(b)
	if err != nil {
		return err
	}
	b.Write(";")
	return nil
}

// writeFromStatement writes the FROM clause for the query's root resource,
// including its SQL table alias.
func (w *queryWriter) writeFromStatement(b *querybuilder.Builder) {
	rootResourceMetadata := w.queryDataStore.RootResourceMetadata()

	rootAlias := w.aliasStore.GetRootAlias()
	b.Write("FROM %s %s ", rootResourceMetadata.FullyQualifiedName, rootAlias)
}

// writeSelectAndExpandStatement writes the SELECT clause, including the
// system limit, selected columns, and nested resource expansions.
func (w *queryWriter) writeSelectAndExpandStatement(b *querybuilder.Builder) error {
	b.Write("SELECT ")

	// Add top statement to implement limit
	w.writeTopStatement(b)

	// Write all selected columns
	i := 0
	for _, columnValue := range w.queryDataStore.Selects() {
		if i > 0 {
			b.Write(",")
		}

		b.Write(
			"%s.%s",
			w.aliasStore.GetRootAlias(),
			w.formatSelectValue(columnValue.Metadata),
		)
		i++
	}
	if i == 0 {
		return qerr.InternalErr("expected a select operation with at least one column specified")
	}

	// Write expansions
	for _, expansion := range w.queryDataStore.Expands() {
		b.Write(",")
		if err := w.writeExpandColumn(b, expansion); err != nil {
			return err
		}
	}

	b.Write(" ")
	return nil
}

// formatSelectValue returns the SQL expression used to select a column.
func (w *queryWriter) formatSelectValue(columnData mdl.ColumnMetadata) string {
	switch columnData.Type {
	case mdl.DbTypePoint:
		return fmt.Sprintf("%s.STAsText() AS %s", columnData.Name, columnData.Name)
	default:
		return columnData.Name
	}
}

// writeExpandColumn writes a nested query as a JSON-valued column in the
// enclosing SELECT statement.
//
// Many-to-one relationships are emitted without a JSON array wrapper, while
// other relationships retain the default JSON array representation.
func (w *queryWriter) writeExpandColumn(b *querybuilder.Builder, expansion qstore.Expansion) error {
	b.Write("JSON_QUERY((")

	nestedWriter := w.newNestedQueryWriter(expansion.QueryData)
	err := nestedWriter.writeQuery(b)
	if err != nil {
		return err
	}

	if expansion.TraversalStep.Relationship.Type == mdl.RelationshipManyToOne {
		b.Write(", WITHOUT_ARRAY_WRAPPER")
	}
	b.Write(")) AS %s", expansion.TraversalStep.Relationship.ExpansionColumnName)
	return nil
}

// writeJoins writes the LEFT JOIN clauses represented by the supplied join
// collection, recursively including any nested joins.
func (w *queryWriter) writeJoins(
	b *querybuilder.Builder,
	joinCollection relationships.JoinCollection,
) {
	for _, join := range joinCollection.Joins() {
		relationship := join.Step.Relationship
		toResourceMetadata := relationship.To.GetMetadata()

		b.Write(
			"LEFT JOIN %s %s on %s.%s = %s.%s",
			toResourceMetadata.FullyQualifiedName,
			join.Alias,
			join.ParentAlias,
			relationship.FromColumn.Name,
			join.Alias,
			relationship.ToColumn.Name,
		)

		if join.SubJoins.JoinsLen() > 0 {
			b.Write(" ")
			w.writeJoins(b, join.SubJoins)
		}
		b.Write(" ")
	}
}

// writeOrderByStatement writes the query's ORDER BY clause using the aliases
// resolved for each ordered column.
func (w *queryWriter) writeOrderByStatement(b *querybuilder.Builder) error {
	if w.queryDataStore.OrderByLen() == 0 {
		return qerr.InternalErr("expected an orderby operation with at least the primary column specified")
	}

	b.Write("ORDER BY ")

	i := 0
	for r := range w.queryDataStore.OrderBy() {
		tableAlias, exists := w.aliasStore.GetJoinAlias(r.ResolvedColumn.ResolvedPath)
		if !exists {
			return qerr.InternalErr(
				"unable to access join alias for path %s",
				r.ResolvedColumn.ResolvedPath.Id,
			)
		}

		if i > 0 {
			b.Write(", ")
		}

		b.Write(
			"%s.%s %s",
			tableAlias,
			r.ResolvedColumn.Metadata.Name,
			string(r.Direction),
		)
		i++
	}

	b.Write(" ")

	return nil
}

// writeTopStatement writes the TOP clause using the query's system limit.
func (w *queryWriter) writeTopStatement(b *querybuilder.Builder) error {
	limit := w.queryDataStore.SystemLimit()
	if limit == 0 {
		return qerr.InternalErr("expected either a user or system defined limit operation")
	}

	b.Write("TOP (%d)", limit)
	b.Write(" ")
	return nil
}

// writeFilterStatement writes the query's WHERE clause, including the
// relationship condition required when the query is nested within another
// query.
func (w *queryWriter) writeFilterStatement(b *querybuilder.Builder) error {
	filterExpression := w.queryDataStore.FilterExpression()

	rootResource := &aliasedResource{
		resource: w.queryDataStore.RootResourceMetadata(),
		alias:    w.aliasStore.GetRootAlias(),
	}

	parentQuery := w.queryDataStore.Parent()
	linkToParent := w.queryDataStore.LinkFromParent()

	if parentQuery != nil && linkToParent != nil {
		b.Write("WHERE ")

		if err := w.writeFilterExpressionWithLink(
			b,
			parentQuery,
			linkToParent,
			rootResource,
			filterExpression,
		); err != nil {
			return err
		}

	} else {
		if filterExpression == nil {
			return nil
		}

		b.Write("WHERE ")
		err := w.writeFilterExpression(b, rootResource, filterExpression)
		if err != nil {
			return err
		}
	}

	b.Write(" ")
	return nil
}

// writeFilterExpressionWithLink writes a filter expression together with the
// relationship condition linking a nested query to its parent query.
func (w *queryWriter) writeFilterExpressionWithLink(
	b *querybuilder.Builder,
	parentQuery qstore.QueryDataStore,
	linkToParent *mdl.TraversalStep,
	rootResource *aliasedResource,
	filterExpression mdl.FilterExpression,
) error {
	b.Write("(")

	b.Write(
		"%s.%s = %s.%s",
		w.queryDataStore.Alias(),
		linkToParent.Relationship.ToColumn.Name,
		parentQuery.Alias(),
		linkToParent.Relationship.FromColumn.Name,
	)

	if filterExpression != nil {
		b.Write(" and ")
		w.writeFilterExpression(b, rootResource, filterExpression)
	}

	b.Write(") ")

	return nil
}

// writeFilterExpression writes the SQL representation of a filter expression,
// dispatching to the appropriate writer for logical, comparison, or collection
// expressions.
func (w *queryWriter) writeFilterExpression(
	b *querybuilder.Builder,
	rootResource *aliasedResource,
	expression mdl.FilterExpression,
) error {
	switch expression := expression.(type) {
	case *mdl.LogicalExpression:
		return w.writeLogicalExpression(b, rootResource, expression)
	case *mdl.ComparisonExpression:
		return w.writeComparisonExpression(b, expression, rootResource)
	case *mdl.CollectionExpression:
		return w.writeCollectionExpression(b, expression, rootResource)
	default:
		return qerr.InternalErr("unexpected filter expression received")
	}
}

// writeLogicalExpression writes a logical filter expression using SQL AND or
// OR operators.
func (w *queryWriter) writeLogicalExpression(
	b *querybuilder.Builder,
	rootResource *aliasedResource,
	expression *mdl.LogicalExpression,
) error {
	operator := ""
	switch expression.Operator {
	case mdl.LogicalOr:
		operator = "or"
	case mdl.LogicalAnd:
		operator = "and"
	default:
		return qerr.InternalErr("unexpected logical operator received '%v'", operator)
	}
	b.Write("(")
	if err := w.writeFilterExpression(b, rootResource, expression.Left); err != nil {
		return err
	}
	b.Write(" %s ", operator)
	if err := w.writeFilterExpression(b, rootResource, expression.Right); err != nil {
		return err
	}
	b.Write(")")

	return nil
}

// writeComparisonExpression writes a comparison filter expression, resolving
// any related-resource path required to address the compared column.
func (w *queryWriter) writeComparisonExpression(
	b *querybuilder.Builder,
	ex *mdl.ComparisonExpression,
	rootResource *aliasedResource,
) error {
	if ex == nil {
		return qerr.InternalErr("comparison expression is nil")
	}

	writeExpression := func(endResource *aliasedResource) error {
		if endResource == nil {
			endResource = rootResource
		}
		return ex.Value.WriteFilterExpression(
			b,
			w,
			fmt.Sprintf("%s.%s", endResource.alias, ex.ResolvedColumn.Metadata.Name),
			ex.Operator,
		)
	}

	return w.writeExpressionWithPath(
		b,
		ex.ExistsNodes,
		writeExpression,
		false,
	)
}

// writeCollectionExpression writes a collection filter expression using SQL
// EXISTS semantics.
//
// Collection-all expressions are rewritten as negated conditions before being
// emitted so that the resulting EXISTS expression represents the required
// collection semantics.
func (w *queryWriter) writeCollectionExpression(
	b *querybuilder.Builder,
	ex *mdl.CollectionExpression,
	rootResource *aliasedResource,
) error {
	filterExpression := ex.FilterExpression
	doNegate := ex.Operator == mdl.CollectionAll
	if doNegate {
		negatedCondition, err := w.negate(ex.FilterExpression)
		if err != nil {
			return err
		}

		filterExpression = negatedCondition
	}

	writeExpression := func(endResource *aliasedResource) error {
		if endResource == nil {
			endResource = rootResource
		}
		return w.writeFilterExpression(
			b,
			endResource,
			filterExpression,
		)
	}

	return w.writeExpressionWithPath(
		b,
		ex.ExistsNodes,
		writeExpression,
		doNegate,
	)
}

// writeExpressionWithPath writes an expression while traversing the related
// resources represented by the supplied EXISTS nodes.
//
// Each relationship is emitted as a nested SQL EXISTS clause until the target
// resource is reached, where writeExpression generates the final condition.
func (w *queryWriter) writeExpressionWithPath(
	b *querybuilder.Builder,
	existsNodes []mdl.ExistsNode,
	writeExpression func(resource *aliasedResource) error,
	doNegate bool,
) error {
	if existsNodes == nil {
		return qerr.InternalErr("exists nodes should not be nil")
	}

	if len(existsNodes) == 0 {
		return writeExpression(nil)
	}

	currentNode := existsNodes[0]

	if doNegate {
		b.Write("NOT ")
	}

	relationship := currentNode.Step.Relationship
	toResourceMetadata := relationship.To.GetMetadata()

	b.Write(
		"EXISTS (SELECT 1 FROM %s %s",
		toResourceMetadata.FullyQualifiedName,
		currentNode.Alias,
	)
	b.Write(" WHERE %s.%s", currentNode.Alias, relationship.ToColumn.Name)
	b.Write(" = ")
	b.Write("%s.%s", currentNode.ParentAlias, relationship.FromColumn.Name)
	b.Write(" AND ")

	// If last node write expression and return
	if len(existsNodes) == 1 {
		if err := writeExpression(&aliasedResource{
			alias:    currentNode.Alias,
			resource: relationship.To.GetMetadata(),
		}); err != nil {
			return err
		}
	} else {
		if err := w.writeExpressionWithPath(
			b,
			existsNodes[1:],
			writeExpression,
			doNegate,
		); err != nil {
			return err
		}
	}

	b.Write(")")
	return nil
}

// negate returns the logical negation of a filter expression.
//
// Logical operators are transformed using De Morgan's laws, comparison
// operators are replaced with their opposites, and collection-any/all
// operators are swapped while recursively negating their conditions.
func (w *queryWriter) negate(expression mdl.FilterExpression) (mdl.FilterExpression, error) {
	filterExpressionBuilder := w.queryDataStore.FilterExpressionBuilder()

	switch ex := expression.(type) {
	case *mdl.LogicalExpression:
		l, err := w.negate(ex.Left)
		if err != nil {
			return nil, err
		}

		r, err := w.negate(ex.Right)
		if err != nil {
			return nil, err
		}

		switch ex.Operator {
		case mdl.LogicalAnd:
			return filterExpressionBuilder.NewLogicalExpression(l, mdl.LogicalOr, r)
		case mdl.LogicalOr:
			return filterExpressionBuilder.NewLogicalExpression(l, mdl.LogicalAnd, r)
		default:
			return nil, fmt.Errorf("unexpected logical operator received '%v'", ex.Operator)
		}

	case *mdl.ComparisonExpression:
		negatedOperator, err := negateComparisonOperator(ex.Operator)
		if err != nil {
			return nil, err
		}

		return ex.WithValue(negatedOperator, ex.Value), nil

	case *mdl.CollectionExpression:
		operator := ex.Operator
		switch ex.Operator {
		case mdl.CollectionAny:
			operator = mdl.CollectionAll
		case mdl.CollectionAll:
			operator = mdl.CollectionAny
		default:
			return nil, fmt.Errorf("unexpected collection operator received '%v'", ex.Operator)
		}

		expression, err := w.negate(ex.FilterExpression)
		if err != nil {
			return nil, err
		}

		return ex.WithValues(operator, expression), nil
	default:
		return nil, fmt.Errorf("unknown filter expression received")
	}
}

// negateComparisonOperator returns the logical inverse of a supported
// comparison operator.
func negateComparisonOperator(operator mdl.ComparisonOperator) (mdl.ComparisonOperator, error) {
	switch operator {
	case mdl.ComparisonEq:
		return mdl.ComparisonNe, nil

	case mdl.ComparisonNe:
		return mdl.ComparisonEq, nil

	case mdl.ComparisonStartsWith:
		return mdl.ComparisonNotStartsWith, nil

	case mdl.ComparisonEndsWith:
		return mdl.ComparisonNotEndsWith, nil

	case mdl.ComparisonContains:
		return mdl.ComparisonNotContains, nil

	case mdl.ComparisonIn:
		return mdl.ComparisonNotIn, nil

	case mdl.ComparisonGt:
		return mdl.ComparisonLe, nil

	case mdl.ComparisonGe:
		return mdl.ComparisonLt, nil

	case mdl.ComparisonLt:
		return mdl.ComparisonGe, nil

	case mdl.ComparisonLe:
		return mdl.ComparisonGt, nil

	default:
		return operator, fmt.Errorf("no negation defined for comparison operatior %v", operator)
	}
}

// WriteFilterExpressionNull writes a SQL comparison against a NULL value.
//
// Equality and inequality are emitted using SQL IS NULL and IS NOT NULL
// semantics respectively.
func (w *queryWriter) WriteFilterExpressionNull(
	b *querybuilder.Builder,
	fieldName string,
	op mdl.ComparisonOperator,
) error {
	switch op {
	case mdl.ComparisonEq:
		b.Write("%s IS NULL", fieldName)
	case mdl.ComparisonNe:
		b.Write("%s IS NOT NULL", fieldName)
	default:
		return fmt.Errorf("unsupported operation: %s", string(op))
	}
	return nil
}

// WriteFilterExpressionString writes a parameterised SQL comparison for a
// string value.
//
// In addition to standard comparison operators, string containment, prefix, and
// suffix comparisons are emitted using SQL LIKE expressions.
func (w *queryWriter) WriteFilterExpressionString(
	b *querybuilder.Builder,
	fieldName string,
	op mdl.ComparisonOperator,
	value string,
) error {
	switch op {
	case mdl.ComparisonEq:
		b.Write("%s = %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonNe:
		b.Write("%s != %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonGt:
		b.Write("%s > %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonGe:
		b.Write("%s >= %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonLt:
		b.Write("%s < %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonLe:
		b.Write("%s <= %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonContains, mdl.ComparisonNotContains:
		modifier := ""
		if op == mdl.ComparisonNotContains {
			modifier = "NOT "
		}

		pattern := fmt.Sprintf("%%%s%%", value)
		b.Write("%s %sLIKE %s", fieldName, modifier, b.Placeholder(pattern))

	case mdl.ComparisonStartsWith, mdl.ComparisonNotStartsWith:
		modifier := ""
		if op == mdl.ComparisonNotStartsWith {
			modifier = "NOT "
		}

		pattern := fmt.Sprintf("%s%%", value)
		b.Write("%s %sLIKE %s", fieldName, modifier, b.Placeholder(pattern))
	case mdl.ComparisonEndsWith, mdl.ComparisonNotEndsWith:
		modifier := ""
		if op == mdl.ComparisonNotEndsWith {
			modifier = "NOT "
		}

		pattern := fmt.Sprintf("%%%s", value)
		b.Write("%s %sLIKE %s", fieldName, modifier, b.Placeholder(pattern))
	default:
		return fmt.Errorf("unsupported string operation: %s", string(op))
	}
	return nil
}

// WriteFilterExpressionInt writes a parameterised SQL comparison for an
// integer value.
func (w *queryWriter) WriteFilterExpressionInt(
	b *querybuilder.Builder,
	fieldName string,
	op mdl.ComparisonOperator,
	value int64,
) error {
	switch op {
	case mdl.ComparisonEq:
		b.Write("%s = %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonNe:
		b.Write("%s != %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonGt:
		b.Write("%s > %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonGe:
		b.Write("%s >= %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonLt:
		b.Write("%s < %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonLe:
		b.Write("%s <= %s", fieldName, b.Placeholder(value))
	default:
		return fmt.Errorf("unsupported int operation: %s", string(op))
	}
	return nil
}

// WriteFilterExpressionFloat writes a parameterised SQL comparison for a
// floating-point value.
func (w *queryWriter) WriteFilterExpressionFloat(
	b *querybuilder.Builder,
	fieldName string,
	op mdl.ComparisonOperator,
	value float64,
) error {
	switch op {
	case mdl.ComparisonEq:
		b.Write("%s = %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonNe:
		b.Write("%s != %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonGt:
		b.Write("%s > %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonGe:
		b.Write("%s >= %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonLt:
		b.Write("%s < %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonLe:
		b.Write("%s <= %s", fieldName, b.Placeholder(value))
	default:
		return fmt.Errorf("unsupported float operation: %s", string(op))
	}
	return nil
}

// WriteFilterExpressionPoint reports that point values do not currently
// support filter comparison operations.
func (w *queryWriter) WriteFilterExpressionPoint(
	b *querybuilder.Builder,
	fieldName string,
	op mdl.ComparisonOperator,
	value mdl.Point,
) error {
	return fmt.Errorf("no operators are currently supported for point")
}

// WriteFilterExpressionDateTime writes a parameterised SQL comparison for a
// time value.
func (w *queryWriter) WriteFilterExpressionDateTime(
	b *querybuilder.Builder,
	fieldName string,
	op mdl.ComparisonOperator,
	value time.Time,
) error {
	switch op {
	case mdl.ComparisonEq:
		b.Write("%s = %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonNe:
		b.Write("%s != %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonGt:
		b.Write("%s > %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonGe:
		b.Write("%s >= %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonLt:
		b.Write("%s < %s", fieldName, b.Placeholder(value))
	case mdl.ComparisonLe:
		b.Write("%s <= %s", fieldName, b.Placeholder(value))
	default:
		return fmt.Errorf("unsupported date/time operation: %s", string(op))
	}
	return nil
}

// WriteFilterExpressionStringList writes a parameterised SQL IN or NOT IN
// expression for a list of string values.
func (w *queryWriter) WriteFilterExpressionStringList(
	b *querybuilder.Builder,
	fieldName string,
	op mdl.ComparisonOperator,
	value []string,
) error {
	listLen := len(value)

	return w.writeList(
		b,
		fieldName,
		listLen,
		op,
		func(i int) error {
			if i < 0 || i > listLen-1 {
				return qerr.InternalErr("unable to write list value: index out of range")
			}
			b.Write("%s", value[i])
			return nil
		},
	)
}

// WriteFilterExpressionIntList writes a SQL IN or NOT IN expression for a list
// of integer values.
func (w *queryWriter) WriteFilterExpressionIntList(
	b *querybuilder.Builder,
	fieldName string,
	op mdl.ComparisonOperator,
	value []int64,
) error {
	listLen := len(value)

	return w.writeList(
		b,
		fieldName,
		listLen,
		op,
		func(i int) error {
			if i < 0 || i > listLen-1 {
				return qerr.InternalErr(
					"unable to write list value: index out of range",
				)
			}
			b.Write("%d", value[i])
			return nil
		},
	)
}

// WriteFilterExpressionFloatList writes a SQL IN or NOT IN expression for a
// list of floating-point values.
func (w *queryWriter) WriteFilterExpressionFloatList(
	b *querybuilder.Builder,
	fieldName string,
	op mdl.ComparisonOperator,
	value []float64,
) error {
	listLen := len(value)

	return w.writeList(
		b,
		fieldName,
		listLen,
		op,
		func(i int) error {
			if i < 0 || i > listLen-1 {
				return qerr.InternalErr(
					"unable to write list value: index out of range",
				)
			}
			b.Write("%f", value[i])
			return nil
		},
	)
}

// writeList writes the SQL representation of a list comparison using either
// IN or NOT IN.
//
// writeValue is responsible for writing each individual list value to the
// query builder.
func (w *queryWriter) writeList(
	b *querybuilder.Builder,
	fieldName string,
	listLength int,
	op mdl.ComparisonOperator,
	writeValue func(i int) error,
) error {
	if op != mdl.ComparisonIn && op != mdl.ComparisonNotIn {
		return fmt.Errorf("unsupported list operation: %s", string(op))
	}

	negationModifier := ""
	if op == mdl.ComparisonNotIn {
		negationModifier = "NOT"
	}

	b.Write("%s %sIN (", fieldName, negationModifier)
	for i := range listLength {
		if i != 0 {
			b.Write(",")
		}
		writeValue(i)
	}
	b.Write(")")
	return nil
}
