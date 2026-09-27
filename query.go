// Package query executes resource queries by coordinating parsing, planning,
// SQL generation, data access, result materialization, and pagination.
package query

import (
	"context"
	"fmt"

	"github.com/turnerbenjamin/querystack/paginationtokens"
	qstore "github.com/turnerbenjamin/querystack/querydatastore"
	qerr "github.com/turnerbenjamin/querystack/queryerror"
	mdl "github.com/turnerbenjamin/querystack/querymodel"
	"github.com/turnerbenjamin/querystack/queryparser"
	qplan "github.com/turnerbenjamin/querystack/queryplanner"
	azSqlWriter "github.com/turnerbenjamin/querystack/querywriter/azsqlwriter"
	valuebuilder "github.com/turnerbenjamin/querystack/valuebuilder"
)

// sqlFlavour identifies the SQL dialect used to execute queries.
type sqlFlavour string

const SqlFlavorAzureSql sqlFlavour = "azure_sql"

// QueryWriterGetter creates a query writer for a planned query data store.
type QueryWriterGetter func(s qstore.QueryDataStore) mdl.QueryWriter

// QueryParserInitialiser creates a new query parser.
type QueryParserInitialiser func() qplan.QueryParser

// queryExecutor coordinates query parsing, planning, execution, and result
// materialization.
type queryExecutor struct {
	repository             mdl.Repository
	queryConfig            mdl.QueryConfig
	schema                 mdl.Schema
	accessPolicy           mdl.AccessPolicy
	pagingTokenBuilder     qplan.PagingTokenBuilder
	queryWriterGetter      QueryWriterGetter
	queryParserInitialiser QueryParserInitialiser
}

// QueryExecutor executes resource queries and returns their results.
type QueryExecutor interface {
	Execute(
		ctx context.Context,
		resourceName string,
		queryString string,
	) (*mdl.ExecuteResult, error)
}

// QueryExecutorConfig configures a query executor and its dependencies.
type QueryExecutorConfig struct {
	Repo                  mdl.Repository
	Schema                mdl.Schema
	AccessPolicy          mdl.AccessPolicy
	PaginationTokenSigner mdl.PayloadSigner
	PaginationTokenSecret []byte
	SqlFlavor             sqlFlavour
	queryConfig           mdl.QueryConfig
}

// NewQueryExecutorFactory creates a query executor from the supplied
// configuration.
func NewQueryExecutorFactory(config QueryExecutorConfig) (QueryExecutor, error) {
	pagingTokenBuilder, err := paginationtokens.NewPagingTokenBuilder(
		config.PaginationTokenSigner,
		config.PaginationTokenSecret,
	)
	if err != nil {
		return nil, err
	}

	sqlWriterGetter, err := getSqlWriter(config.SqlFlavor)

	if err != nil {
		return nil, err
	}

	return &queryExecutor{
		repository:             config.Repo,
		schema:                 config.Schema,
		accessPolicy:           config.AccessPolicy,
		pagingTokenBuilder:     pagingTokenBuilder,
		queryWriterGetter:      sqlWriterGetter,
		queryParserInitialiser: queryparser.NewQueryParser,
		queryConfig:            mdl.QueryConfigWithDefaults(config.queryConfig),
	}, err
}

// Execute parses, plans, executes, and materializes a query for the named
// resource.
func (qf *queryExecutor) Execute(
	ctx context.Context,
	resourceName string,
	queryString string,
) (*mdl.ExecuteResult, error) {
	rootResource, exists := qf.schema.GetResource(resourceName)
	if !exists {
		return nil, qerr.BindingErr("the table %s does not exist in the schema", resourceName)
	}

	queryParser := qf.queryParserInitialiser()
	valueBuilder := valuebuilder.NewValueBuilder()

	// Plan query
	s, err := qplan.PlanQuery(
		qf.queryConfig,
		queryString,
		queryParser,
		qf.pagingTokenBuilder,
		valueBuilder,
		rootResource,
		qf.accessPolicy,
	)
	if err != nil {
		return nil, err
	}

	// Write and execute the query
	w := qf.queryWriterGetter(s)
	json, count, err := qf.executeQueryStatement(ctx, s, w)

	// Parse the query into a table model
	queryResults, stdErr := rootResource.SliceFromJSON(
		json,
		s.ProjectionNode(),
	)
	if stdErr != nil {
		return nil, qerr.InternalErr("unable to create model slice: %v", stdErr)
	}

	// Build the next page token
	nextPageToken, err := qf.getNextPageToken(s, &queryResults)
	if err != nil {
		return nil, err
	}

	// return the results
	return &mdl.ExecuteResult{
		Count:         count,
		NextPageToken: nextPageToken,
		Data:          queryResults,
	}, nil
}

// getSqlWriter returns a query writer factory for the requested SQL dialect.
func getSqlWriter(flavour sqlFlavour) (QueryWriterGetter, error) {
	switch flavour {
	case SqlFlavorAzureSql:
		return azSqlWriter.NewQueryWriter, nil
	default:
		return nil, fmt.Errorf("unsupported sql flavour: %s", flavour)
	}
}

// getNextPageToken builds a pagination token when the query returned more
// records than the configured page limit, and trims the results to that limit.
func (e *queryExecutor) getNextPageToken(
	s qstore.QueryDataStore,
	queryResults *[]mdl.TableModel,
) (string, error) {
	results := *queryResults
	limit := s.Limit()
	isNextRecord := len(results) > int(limit)

	if !isNextRecord {
		return "", nil
	}
	lastRecord := results[limit-1]
	*queryResults = results[0:limit]

	return e.pagingTokenBuilder.BuildToken(
		s,
		lastRecord,
	)
}

// executeQueryStatement generates and executes the query SQL, optionally
// executing a count query when requested.
func (e *queryExecutor) executeQueryStatement(
	ctx context.Context,
	s qstore.QueryDataStore,
	w mdl.QueryWriter,
) ([]byte, *uint64, error) {
	queryStatement, err := w.WriteQueryStatement()
	if err != nil {
		return nil, nil, err
	}

	var json []byte = nil
	var count *uint64 = nil
	if s.DoCount() {
		countStatement, err := w.WriteCountStatement()
		if err != nil {
			return nil, nil, err
		}

		json, count, err = e.repository.ExecuteJsonRequestWithCount(
			ctx,
			queryStatement,
			countStatement,
		)
	} else {
		json, err = e.repository.ExecuteJsonRequest(ctx, queryStatement)
	}
	if err != nil {
		return nil, nil, qerr.InternalErr("query executor failed: %v", err)
	}
	return json, count, nil
}
