// Package query executes resource queries by coordinating parsing, planning,
// SQL generation, data access, result materialization, and pagination.
package querystack

import (
	"context"
	"fmt"

	"github.com/turnerbenjamin/querystack/paginationtokens"
	qstore "github.com/turnerbenjamin/querystack/querydatastore"
	qerr "github.com/turnerbenjamin/querystack/queryerror"
	"github.com/turnerbenjamin/querystack/querymodel"
	mdl "github.com/turnerbenjamin/querystack/querymodel"
	"github.com/turnerbenjamin/querystack/queryparser"
	qplan "github.com/turnerbenjamin/querystack/queryplanner"
	azSqlWriter "github.com/turnerbenjamin/querystack/querywriter/azsqlwriter"
	valuebuilder "github.com/turnerbenjamin/querystack/valuebuilder"
)

// QueryWriterGetter creates a query writer for a planned query data store.
type QueryWriterGetter func(s qstore.QueryDataStore) mdl.QueryWriter

// QueryParserInitialiser creates a new query parser.
type QueryParserInitialiser func() qplan.QueryParser

// queryExecutor coordinates query parsing, planning, execution, and result
// materialization.
type QueryExecutor struct {
	repository             mdl.Repository
	queryConfig            mdl.QueryConfig
	schema                 mdl.Schema
	pagingTokenBuilder     qplan.PagingTokenBuilder
	queryWriterGetter      QueryWriterGetter
	queryParserInitialiser QueryParserInitialiser
}

// NewQueryExecutorFactory creates a query executor from the supplied
// configuration.
func NewQueryExecutor(
	repository mdl.Repository,
	schema mdl.Schema,
	paginationTokenSigner mdl.PayloadSigner,
	paginationTokenSecret []byte,
	sqlFlavor querymodel.SqlFlavour,
	queryConfig mdl.QueryConfig,
) (*QueryExecutor, error) {
	if repository == nil {
		return nil, qerr.InternalErr(
			"unable to build query executor factory: repository is nil",
		)
	}

	if schema == nil {
		return nil, qerr.InternalErr(
			"unable to build query executor factory: schema token secret is nil",
		)
	}

	if paginationTokenSigner == nil {
		return nil, qerr.InternalErr(
			"unable to build query executor factory: pagination token signer is nil",
		)
	}

	if paginationTokenSecret == nil {
		return nil, qerr.InternalErr(
			"unable to build query executor factory: pagination token secret is nil",
		)
	}

	pagingTokenBuilder, err := paginationtokens.NewPagingTokenBuilder(
		paginationTokenSigner,
		paginationTokenSecret,
	)
	if err != nil {
		return nil, err
	}

	sqlWriterGetter, err := getSqlWriter(sqlFlavor)

	if err != nil {
		return nil, err
	}

	return &QueryExecutor{
		repository:             repository,
		schema:                 schema,
		pagingTokenBuilder:     pagingTokenBuilder,
		queryWriterGetter:      sqlWriterGetter,
		queryParserInitialiser: queryparser.NewQueryParser,
		queryConfig:            mdl.QueryConfigWithDefaults(queryConfig),
	}, err
}

// Execute parses, plans, executes, and materializes a query for the named
// resource.
func (executor *QueryExecutor) Execute(
	ctx context.Context,
	resourceName string,
	accessPolicy mdl.AccessPolicy,
	queryString string,
) (*mdl.ExecuteResult, error) {
	resource, exists := executor.schema.GetResource(resourceName)
	if !exists {
		return nil, qerr.BindingErr("the table %s does not exist in the schema", resourceName)
	}

	typedResult, err := execute(
		executor,
		resource,
		ctx,
		accessPolicy,
		queryString,
		func(
			jsonData []byte,
			projection *mdl.ProjectionNode,
		) ([]mdl.TableModel, error) {
			return resource.SliceFromJSON(jsonData, projection)
		},
	)
	if err != nil {
		return nil, err
	}

	// return the results
	return &mdl.ExecuteResult{
		Count:         typedResult.Count,
		NextPageToken: typedResult.NextPageToken,
		Data:          typedResult.Data,
	}, nil
}

func Execute[T mdl.TableModel](
	executor *QueryExecutor,
	resource mdl.TypedResource[T],
	ctx context.Context,
	accessPolicy mdl.AccessPolicy,
	queryString string,
) (*mdl.TypedExecuteResult[T], error) {
	return execute(
		executor,
		mdl.Resource(resource),
		ctx,
		accessPolicy,
		queryString,
		func(
			jsonData []byte,
			projection *mdl.ProjectionNode,
		) ([]T, error) {
			return resource.TypedSliceFromJSON(jsonData, projection)
		},
	)
}

func execute[T mdl.TableModel](
	executor *QueryExecutor,
	resource mdl.Resource,
	ctx context.Context,
	accessPolicy mdl.AccessPolicy,
	queryString string,
	materialize func(
		[]byte,
		*mdl.ProjectionNode,
	) ([]T, error),
) (*mdl.TypedExecuteResult[T], error) {

	queryParser := executor.queryParserInitialiser()
	valueBuilder := valuebuilder.NewValueBuilder()

	// Plan query
	plannedQuery, err := qplan.PlanQuery(
		executor.queryConfig,
		queryString,
		queryParser,
		executor.pagingTokenBuilder,
		valueBuilder,
		resource,
		accessPolicy,
	)
	if err != nil {
		return nil, err
	}

	// Write and execute the query
	w := executor.queryWriterGetter(plannedQuery)
	json, count, err := executor.executeQueryStatement(ctx, plannedQuery, w)

	queryResults, err := materialize(
		json,
		plannedQuery.ProjectionNode(),
	)
	if err != nil {
		return nil, err
	}

	nextPageToken, err := getNextPageToken(executor, plannedQuery, &queryResults)
	if err != nil {
		return nil, err
	}

	// return the results
	return &mdl.TypedExecuteResult[T]{
		Count:         count,
		NextPageToken: nextPageToken,
		Data:          queryResults,
	}, nil

}

// getSqlWriter returns a query writer factory for the requested SQL dialect.
func getSqlWriter(flavour querymodel.SqlFlavour) (QueryWriterGetter, error) {
	switch flavour {
	case querymodel.SqlFlavorAzureSql:
		return azSqlWriter.NewQueryWriter, nil
	default:
		return nil, fmt.Errorf("unsupported sql flavour: %s", flavour)
	}
}

// getNextPageToken builds a pagination token when the query returned more
// records than the configured page limit, and trims the results to that limit.
func getNextPageToken[T mdl.TableModel](
	executor *QueryExecutor,
	s qstore.QueryDataStore,
	queryResults *[]T,
) (string, error) {
	results := *queryResults
	limit := s.Limit()
	isNextRecord := len(results) > int(limit)

	if !isNextRecord {
		return "", nil
	}
	lastRecord := results[limit-1]
	*queryResults = results[0:limit]

	return executor.pagingTokenBuilder.BuildToken(
		s,
		lastRecord,
	)
}

// executeQueryStatement generates and executes the query SQL, optionally
// executing a count query when requested.
func (executor *QueryExecutor) executeQueryStatement(
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

		json, count, err = executor.repository.ExecuteJsonRequestWithCount(
			ctx,
			queryStatement,
			countStatement,
		)
	} else {
		json, err = executor.repository.ExecuteJsonRequest(ctx, queryStatement)
	}
	if err != nil {
		return nil, nil, qerr.InternalErr("query executor failed: %v", err)
	}
	return json, count, nil
}
