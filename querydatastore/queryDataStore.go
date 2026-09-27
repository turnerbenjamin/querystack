// Package querydatastore is responsible for maintaining query data in a central
// store. In earilier iterations, a pipeline approach was taken where the query
// data was enriched at each stage of the pipeline; this resulted in a clear
// separation of responsibilities but it created complications when accessing
// the data which may or may not have been fully enriched.
//
// This package takes the alternative approach of calling the relevant
// metadataBinder and relationshipPlanner methods as operations are added to the
// store. This allows consuming packages to access the data with the assurance
// that all metadata and relationship data will always be populated
//
// This file contains the QueryDataStore which is the repository for query data
package querydatastore

import (
	"iter"
	"math"

	"github.com/turnerbenjamin/querystack/metadatastore"
	qerr "github.com/turnerbenjamin/querystack/queryerror"
	mdl "github.com/turnerbenjamin/querystack/querymodel"
	"github.com/turnerbenjamin/querystack/relationships"
)

// MetadataBinder is used to bind a given Column, Path or Relationship to schema
// metadata and validate access against the access policy
type MetadataBinder interface {
	ResolveColumn(columnPath string) (mdl.ResolvedColumn, error)
	ResolvePath(pathString string) (mdl.ResolvedPath, error)
	ResolveRelationship(resource mdl.TableMetadata, relationshipName string) (mdl.TraversalStep, error)
}

// FilterExpressionBuilder provides methods to build FilterExpressions and
// ensures that columns, paths and relationships are bound to metadata and that
// the relationship planner is updated as required, on creation of those
// expressions
type FilterExpressionBuilder interface {

	// ValueBuilder returns the Value builder instance for the
	// filterExpressionBuilder
	ValueBuilder() mdl.ValueBuilder

	// NewLogicalExpression builds a new logical expression
	NewLogicalExpression(
		left mdl.FilterExpression,
		operator mdl.LogicalOperator,
		right mdl.FilterExpression,
	) (*mdl.LogicalExpression, error)

	// NewComparisonExpression builds a new comparison expression. It ensures
	// that the columnPath is bound to schema metadata, invokes validation
	// functions that use this metadata and ensures that the relationshipPlanner
	// processes the column
	NewComparisonExpression(
		columnPath string,
		operator mdl.ComparisonOperator,
		value mdl.Value,
	) (*mdl.ComparisonExpression, error)

	// NewComparisonExpressionFromResolvedColumn is used to build a new
	// comparison expression where the column has been already been resolved -
	// For instance, when creating an expression for a cursor filter where the
	// column has already been resolved when builing the associated orderbyrule
	NewComparisonExpressionFromResolvedColumn(
		column mdl.ResolvedColumn,
		operator mdl.ComparisonOperator,
		value mdl.Value,
	) (*mdl.ComparisonExpression, error)

	// NewCollectionExpression builds a new collection expression. It ensures
	// that the path is bound to metadata and processed by the relationship
	// planner
	NewCollectionExpression(
		resourcePath string,
		operator mdl.CollectionOperator,
		filterExpression mdl.FilterExpression,
	) (*mdl.CollectionExpression, error)
}

// Expansion stores the an expansion query
type Expansion struct {

	// QueryData is a nested QueryDataStore for the expand query
	QueryData QueryDataStore

	// TraversalStep is the link from the parent query to the expand query
	TraversalStep mdl.TraversalStep
}

// QueryDataStore is the primary repository for all query data. It acts as a
// central controller, ensuring that operations are bound and processed by the
// relationship planner as they are added to the store
type QueryDataStore interface {
	// RootResource is the root resource for the query store
	RootResource() mdl.Resource

	// RootResourceMetadata is the metadata for the root resource
	RootResourceMetadata() mdl.TableMetadata

	// RootResourceAccessPolicy is the table access policy for the root resource
	RootResourceAccessPolicy() mdl.TableAccessPolicy

	// QueryString is the raw query string associated with the query
	QueryString() string

	// IsTopLevelQuery returns true if the query is the top-level query - For
	// nested queries in expand statements it will return false
	IsTopLevelQuery() bool

	// Depth returns the depth of the query - A top-level query will have a
	// depth of 0
	Depth() uint8

	// Alias returns the alias for the query
	Alias() string

	// Parent returns the parent query data store, or nil if there is no parent
	Parent() *queryDataStore

	// LinkFromParent returns the traversal step from the parent to the current
	// store or nil if no parent exists
	LinkFromParent() *mdl.TraversalStep

	// FilterExpressionBuilder returns the filter expression builder for the
	// QueryDataStore
	FilterExpressionBuilder() FilterExpressionBuilder

	// GetAliasStore returns the AliasStore, this can be used to access the
	// alias assigned to the root resource and any joins
	GetAliasStore() relationships.AliasStore

	// JoinCollection returns the join collection compiled for the query. This
	// can be used, for instance, to access planned joins for orderby queries
	JoinCollection() relationships.JoinCollection

	// ProjectionNode returns the projection node for the query. This is used to
	// set the projection on models and any nested models
	ProjectionNode() *mdl.ProjectionNode

	// AddSelect adds a named column to the query's select operation. It binds the
	// column to metadata at the point of addition and will return an error if
	// binding fails or if access to the column is not permitted under the access
	// policy. The column will be added to the query projection
	AddSelect(columnName string) error

	// AddSelect adds a named column to the query's select operation. It binds the
	// column to metadata at the point of addition and will return an error if
	// binding fails or if access to the column is not permitted under the access
	// policy. The column will not be added to the query projection
	AddSystemSelect(columnName string) error

	// Selects iterates over all columns in the query select operation
	Selects() iter.Seq2[string, mdl.ResolvedColumn]

	// AddExpand adds a new expand by relationshipId and returns a nested
	// QueryDataStore for that expand. It will return an error if an expansion has
	// already been added for the relationship, if the relationship id cannot be
	// bound to metadata or if the traversal is not permitted under the access
	// policy. The relationship column will be added to the projection.
	AddExpand(relationshipId string) (QueryDataStore, error)

	// AddExpand adds a new expand by relationshipId and returns a nested
	// QueryDataStore for that expand. It will return an error if an expansion has
	// already been added for the relationship, if the relationship id cannot be
	// bound to metadata or if the traversal is not permitted under the access
	// policy. The relationship column will not be added to the projection
	AddSystemExpand(relationshipId string) (QueryDataStore, error)

	// ExpandsLen returns the total count of expands statements
	ExpandsLen() int

	// Expands iterates over all expansions in the data store
	Expands() iter.Seq2[string, Expansion]

	// GetExpansionByRelationshipId returns the Expansion associated with a given
	// relationship id and a bool indicating whether the expansion exists in the
	// store or not
	GetExpansionByRelationshipId(relationshipId string) (Expansion, bool)

	// SetFilterExpression sets a filter expression on the store
	SetFilterExpression(ex mdl.FilterExpression)

	// SetOrAppendFilter will add the filter as a top-level filter to the query
	// store, either as a lone filter, if one has not previously been set, or part
	// of an 'and' group if an existing filter exists
	SetOrAppendFilter(ex mdl.FilterExpression)

	// FilterExpression returns the filter expression for the data store or nil if
	// no filter has been set
	FilterExpression() mdl.FilterExpression

	// AddOrderBy adds a new sorting rule to the query store. It ensures that
	// metadata is bound to the column being sorted and that any required joins are
	// added to the query join collection. An error will be returned if the column
	// cannot be bound, access to the column is not permitted under the access
	// policy or if the column type is not comparable, e.g. a point
	AddOrderBy(columnPath string, dir mdl.SortDirectionOperator) error

	// OrderByLen returns the total count of sorting rules in the data store order
	// by operation
	OrderByLen() int

	// OrderBy iterates over the sorting rules in the data store order by operation
	OrderBy() iter.Seq[mdl.SortingRule]

	// SetLimit sets the limit of records returned
	SetLimit(limit uint32)

	// Limit returns the user specified limit
	Limit() uint32

	// SetSystemLimit sets the system limit - This can be used to modify the number
	// of records actually retrieved in the query without mutating the user
	// specified limit
	SetSystemLimit(limit uint64)

	// SystemLimit returns the system limit
	SystemLimit() uint64

	// SetDoCount sets whether the query should include the total count of records
	SetDoCount(doCount bool)

	// DoCount returns whether count has been set to true in the data store
	DoCount() bool

	// SetPagingToken sets the paging token
	SetPagingToken(string)

	// PagingToken returns the paging token and a bool indicating whether a paging
	// token has been set on the store
	PagingToken() (string, bool)
}

// queryDataStore is the primary store for all query data. When any operation is
// added, paths are immediately bound to metadata and the relationship plan is
// updated as required. This provides a guarantee to consumers that at any point
// in the process, all metadata is set and the relationship plan is current.
type queryDataStore struct {
	// overview data
	queryString          string
	rootResource         mdl.Resource
	rootResourceMetadata mdl.TableMetadata
	depth                uint8
	projectionNode       *mdl.ProjectionNode

	// parent
	parent       *queryDataStore
	linkToParent *mdl.TraversalStep

	// metadata and relationship binding deps
	metadataBinder          MetadataBinder
	relationshipPlanner     *relationships.RelationshipPlanner
	filterExpressionBuilder filterExpressionBuilder
	valueBuilder            mdl.ValueBuilder

	// access policies
	accessPolicy             mdl.AccessPolicy
	rootResourceAccessPolicy mdl.TableAccessPolicy

	// operations
	selects     map[string]mdl.ResolvedColumn
	expands     map[string]Expansion
	filter      mdl.FilterExpression
	orderBy     []mdl.SortingRule
	doCount     bool
	limit       uint32
	systemLimit uint64
	pagingToken string
}

// NewQueryDataStore returns a new top-level query data store
func NewQueryDataStore(
	queryString string,
	rootResource mdl.Resource,
	accessPolicy mdl.AccessPolicy,
	valueBuilder mdl.ValueBuilder,
) (QueryDataStore, error) {
	// init parentage and depth for top-level query
	depth := uint8(0)
	var parent *queryDataStore = nil
	var linkToParent *mdl.TraversalStep = nil

	// init the root projectionNode for the query
	projectionNode := &mdl.ProjectionNode{
		Projection: rootResource.InitProjection(),
	}

	return newQueryDataStore(
		queryString,
		rootResource,
		accessPolicy,
		valueBuilder,
		depth,
		parent,
		linkToParent,
		projectionNode,
	)
}

// newQueryDataStore returns a new query data store
func newQueryDataStore(
	queryString string,
	rootResource mdl.Resource,
	accessPolicy mdl.AccessPolicy,
	valueBuilder mdl.ValueBuilder,
	depth uint8,
	parent *queryDataStore,
	linkToParent *mdl.TraversalStep,
	projectionNode *mdl.ProjectionNode,
) (QueryDataStore, error) {
	rootResourceMetadata := rootResource.GetMetadata()

	metadataBinder, err := metadatastore.NewMetadataBinder(
		rootResourceMetadata,
		accessPolicy,
	)
	if err != nil {
		return nil, err
	}

	relationshipPlanner, err := relationships.NewRelationshipPlanner(
		rootResourceMetadata,
	)
	if err != nil {
		return nil, err
	}

	rootResourceAccessPolicy, exists := accessPolicy.GetTableAccessPolicy(
		rootResourceMetadata.Name,
	)
	if !exists {
		return nil, qerr.InternalErr(
			"unable to create new data store: root resource access policy is nil",
		)
	}

	return &queryDataStore{
		queryString:          queryString,
		rootResource:         rootResource,
		rootResourceMetadata: rootResourceMetadata,
		depth:                depth,
		projectionNode:       projectionNode,

		parent:       parent,
		linkToParent: linkToParent,

		metadataBinder:      metadataBinder,
		relationshipPlanner: relationshipPlanner,
		filterExpressionBuilder: filterExpressionBuilder{
			rootResourceMetadata: rootResourceMetadata,
			metadataBinder:       metadataBinder,
			relationshipPlanner:  relationshipPlanner,
			valueBuilder:         valueBuilder,
		},
		valueBuilder: valueBuilder,

		accessPolicy:             accessPolicy,
		rootResourceAccessPolicy: rootResourceAccessPolicy,
	}, nil
}

// RootResource is the root resource for the query store
func (qd *queryDataStore) RootResource() mdl.Resource {
	return qd.rootResource
}

// RootResourceMetadata is the metadata for the root resource
func (qd *queryDataStore) RootResourceMetadata() mdl.TableMetadata {
	return qd.rootResourceMetadata
}

// RootResourceAccessPolicy is the table access policy for the root resource
func (qd *queryDataStore) RootResourceAccessPolicy() mdl.TableAccessPolicy {
	return qd.rootResourceAccessPolicy
}

// QueryString is the raw query string associated with the query
func (qd *queryDataStore) QueryString() string {
	return qd.queryString
}

// IsTopLevelQuery returns true if the query is the top-level query - For
// nested queries in expand statements it will return false
func (qd *queryDataStore) IsTopLevelQuery() bool {
	return qd.depth == 0
}

// Depth returns the depth of the query - A top-level query will have a
// depth of 0
func (qd *queryDataStore) Depth() uint8 {
	return qd.depth
}

// Alias returns the alias for the query
func (qd *queryDataStore) Alias() string {
	return qd.relationshipPlanner.Aliases.GetRootAlias()
}

// Parent returns the parent query data store, or nil if there is no parent
func (qd *queryDataStore) Parent() *queryDataStore {
	return qd.parent
}

// LinkFromParent returns the traversal step from the parent to the current
// store or nil if no parent exists
func (qd *queryDataStore) LinkFromParent() *mdl.TraversalStep {
	return qd.linkToParent
}

// FilterExpressionBuilder returns the filter expression builder for the
// QueryDataStore
func (qd *queryDataStore) FilterExpressionBuilder() FilterExpressionBuilder {
	return qd.filterExpressionBuilder
}

// GetAliasStore returns the AliasStore, this can be used to access the
// alias assigned to the root resource and any joins
func (qd *queryDataStore) GetAliasStore() relationships.AliasStore {
	return qd.relationshipPlanner.Aliases
}

// JoinCollection returns the join collection compiled for the query. This
// can be used, for instance, to access planned joins for orderby queries
func (qd *queryDataStore) JoinCollection() relationships.JoinCollection {
	return qd.relationshipPlanner.JoinStore
}

// ProjectionNode returns the projection node for the query. This is used to
// set the projection on models and any nested models
func (qd *queryDataStore) ProjectionNode() *mdl.ProjectionNode {
	return qd.projectionNode
}

// AddSelect adds a named column to the query's select operation. It binds the
// column to metadata at the point of addition and will return an error if
// binding fails or if access to the column is not permitted under the access
// policy. The column will be added to the query projection
func (qd *queryDataStore) AddSelect(columnName string) error {
	return qd.addSelect(columnName, true)
}

// AddSelect adds a named column to the query's select operation. It binds the
// column to metadata at the point of addition and will return an error if
// binding fails or if access to the column is not permitted under the access
// policy. The column will not be added to the query projection
func (qd *queryDataStore) AddSystemSelect(columnName string) error {
	return qd.addSelect(columnName, false)
}

// AddSelect adds a named column to the query's select operation. It binds the
// column to metadata at the point of addition and will return an error if
// binding fails or if access to the column is not permitted under the access
// policy.
func (qd *queryDataStore) addSelect(columnName string, doProject bool) error {
	// Get the metadata
	c, err := qd.metadataBinder.ResolveColumn(columnName)
	if err != nil {
		return err
	}

	// Ensure select initialised
	if qd.selects == nil {
		qd.selects = make(map[string]mdl.ResolvedColumn, 1)
	}

	// Set the selects
	qd.selects[columnName] = c

	// Set the projection if required
	if doProject {
		if err := qd.projectionNode.Projection.Add(columnName); err != nil {
			return err
		}
	}
	return nil
}

// Selects iterates over all columns in the query select operation
func (qd *queryDataStore) Selects() iter.Seq2[string, mdl.ResolvedColumn] {
	return func(yield func(string, mdl.ResolvedColumn) bool) {
		for k, v := range qd.selects {
			if !yield(k, v) {
				return
			}
		}
	}
}

// AddExpand adds a new expand by relationshipId and returns a nested
// QueryDataStore for that expand. It will return an error if an expansion has
// already been added for the relationship, if the relationship id cannot be
// bound to metadata or if the traversal is not permitted under the access
// policy. The relationship column will be added to the projection.
func (qd *queryDataStore) AddExpand(relationshipId string) (QueryDataStore, error) {
	return qd.addExpand(relationshipId, true)
}

// AddExpand adds a new expand by relationshipId and returns a nested
// QueryDataStore for that expand. It will return an error if an expansion has
// already been added for the relationship, if the relationship id cannot be
// bound to metadata or if the traversal is not permitted under the access
// policy. The relationship column will not be added to the projection
func (qd *queryDataStore) AddSystemExpand(relationshipId string) (QueryDataStore, error) {
	return qd.addExpand(relationshipId, false)
}

// AddExpand adds a new expand by relationshipId and returns a nested
// QueryDataStore for that expand. It will return an error if an expansion has
// already been added for the relationship, if the relationship id cannot be
// bound to metadata or if the traversal is not permitted under the access
// policy.
func (qd *queryDataStore) addExpand(relationshipId string, doProject bool) (QueryDataStore, error) {
	// Check that the expand is not being duplicated
	if qd.expands != nil {
		if _, exists := qd.expands[relationshipId]; exists {
			msg := "an expand operations has already been declared for %s"
			if doProject {
				return nil, qerr.SyntaxErr(msg, relationshipId)
			} else {
				return nil, qerr.InternalErr(msg, relationshipId)
			}
		}
	}

	// Ensure that the expands slice is initialised
	if qd.expands == nil {
		qd.expands = make(map[string]Expansion, 4)
	}

	// Map the relationship to a traversal step
	traversalStep, err := qd.metadataBinder.ResolveRelationship(
		qd.rootResourceMetadata,
		relationshipId,
	)
	if err != nil {
		return nil, err
	}

	// Compute the expansion depth - Max depth is the responsibility of the
	// query planner so checking range only
	if qd.depth == math.MaxUint8 {
		return nil, qerr.InternalErr(
			"unable to add expand as it will cause depth overflow",
		)
	}
	expansionDepth := qd.depth + 1

	// Init projection node for the expanded resource
	expandedResourceData := traversalStep.Relationship.To
	expandedProjectionNode := &mdl.ProjectionNode{
		Projection: expandedResourceData.InitProjection(),
	}

	// Add expanded resource projection node as a child of the current query's
	// projection node
	if qd.projectionNode.Children == nil {
		qd.projectionNode.Children = map[string]*mdl.ProjectionNode{}
	}
	qd.projectionNode.Children[traversalStep.Relationship.ExpansionColumnName] = expandedProjectionNode

	// Build expanded query
	expandedQuery, err := newQueryDataStore(
		qd.queryString,
		expandedResourceData,
		qd.accessPolicy,
		qd.valueBuilder,
		expansionDepth,
		qd,
		&traversalStep,
		expandedProjectionNode,
	)
	if err != nil {
		return nil, err
	}

	expansion := Expansion{
		QueryData:     expandedQuery,
		TraversalStep: traversalStep,
	}

	qd.expands[traversalStep.Relationship.Id] = expansion

	// Add expansion column to the current query's projection where required
	if doProject {
		if err := qd.projectionNode.Projection.Add(
			traversalStep.Relationship.ExpansionColumnName,
		); err != nil {
			return nil, err
		}
	}

	return expansion.QueryData, nil
}

// ExpandsLen returns the total count of expands statements
func (qd *queryDataStore) ExpandsLen() int {
	return len(qd.expands)
}

// Expands iterates over all expansions in the data store
func (qd *queryDataStore) Expands() iter.Seq2[string, Expansion] {
	return func(yield func(string, Expansion) bool) {
		for k, v := range qd.expands {
			if !yield(k, v) {
				return
			}
		}
	}
}

// GetExpansionByRelationshipId returns the Expansion associated with a given
// relationship id and a bool indicating whether the expansion exists in the
// store or not
func (qd *queryDataStore) GetExpansionByRelationshipId(relationshipId string) (Expansion, bool) {
	if qd.expands == nil {
		return Expansion{}, false
	}

	if _, exists := qd.expands[relationshipId]; !exists {
		return Expansion{}, false
	}

	return qd.expands[relationshipId], true
}

// SetFilterExpression sets a filter expression on the store
func (qd *queryDataStore) SetFilterExpression(ex mdl.FilterExpression) {
	qd.filter = ex
}

// SetOrAppendFilter will add the filter as a top-level filter to the query
// store, either as a lone filter, if one has not previously been set, or part
// of an 'and' group if an existing filter exists
func (qd *queryDataStore) SetOrAppendFilter(ex mdl.FilterExpression) {
	if qd.filter == nil {
		qd.filter = ex
	} else {
		qd.filter = &mdl.LogicalExpression{
			Left:     qd.filter,
			Operator: mdl.LogicalAnd,
			Right:    ex,
		}
	}
}

// FilterExpression returns the filter expression for the data store or nil if
// no filter has been set
func (qd *queryDataStore) FilterExpression() mdl.FilterExpression {
	return qd.filter
}

// AddOrderBy adds a new sorting rule to the query store. It ensures that
// metadata is bound to the column being sorted and that any required joins are
// added to the query join collection. An error will be returned if the column
// cannot be bound, access to the column is not permitted under the access
// policy or if the column type is not comparable, e.g. a point
func (qd *queryDataStore) AddOrderBy(columnPath string, dir mdl.SortDirectionOperator) error {
	// Get the metadata
	c, err := qd.metadataBinder.ResolveColumn(columnPath)
	if err != nil {
		return err
	}

	// Check Compatibility
	if _, isComparable := mdl.ComparableDbTypes[c.Metadata.Type]; !isComparable {
		return qerr.SyntaxErr(
			"%s cannot be used in an orderby operation as its type is not comparable",
			c.Metadata.Name,
		)
	}

	if err := qd.relationshipPlanner.ProcessJoin(c.ResolvedPath); err != nil {
		return err
	}

	// ensure order by field is created
	if qd.orderBy == nil {
		qd.orderBy = make([]mdl.SortingRule, 0, 1)
	}

	// add the rule
	qd.orderBy = append(
		qd.orderBy,
		mdl.SortingRule{
			ResolvedColumn: c,
			Direction:      dir,
		},
	)
	return nil
}

// OrderByLen returns the total count of sorting rules in the data store order
// by operation
func (qd *queryDataStore) OrderByLen() int {
	return len(qd.orderBy)
}

// OrderBy iterates over the sorting rules in the data store order by operation
func (qd *queryDataStore) OrderBy() iter.Seq[mdl.SortingRule] {
	return func(yield func(mdl.SortingRule) bool) {
		for _, rule := range qd.orderBy {
			if !yield(rule) {
				return
			}
		}
	}
}

// SetLimit sets the limit of records returned
func (qd *queryDataStore) SetLimit(limit uint32) {
	qd.limit = limit
}

// Limit returns the user specified limit
func (qd *queryDataStore) Limit() uint32 {
	return qd.limit
}

// SetSystemLimit sets the system limit - This can be used to modify the number
// of records actually retrieved in the query without mutating the user
// specified limit
func (qd *queryDataStore) SetSystemLimit(limit uint64) {
	qd.systemLimit = limit
}

// SystemLimit returns the system limit
func (qd *queryDataStore) SystemLimit() uint64 {
	return qd.systemLimit
}

// SetDoCount sets whether the query should include the total count of records
func (qd *queryDataStore) SetDoCount(doCount bool) {
	qd.doCount = doCount
}

// DoCount returns whether count has been set to true in the data store
func (qd *queryDataStore) DoCount() bool {
	return qd.doCount
}

// SetPagingToken sets the paging token
func (qd *queryDataStore) SetPagingToken(pagingToken string) {
	qd.pagingToken = pagingToken
}

// PagingToken returns the paging token and a bool indicating whether a paging
// token has been set on the store
func (qd *queryDataStore) PagingToken() (string, bool) {
	if qd.pagingToken == "" {
		return "", false
	}
	return qd.pagingToken, true
}
