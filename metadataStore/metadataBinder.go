// Package metadataStore is responsible for binding column/path/relationship
// identifiers to schema metadata
//
// This file contains metadataBinder which represents the main API for this
// package
package metadataStore

import (
	"fmt"
	"strings"
	"sync"

	qerr "github.com/turnerbenjamin/querystack/queryError"
	mdl "github.com/turnerbenjamin/querystack/queryModel"
)

// pathStore is a shared path collection used to reduce the number of
// short-lived objects in the query pipeline
var pathStore = pathCollection{
	store: make(map[string]mdl.ResolvedPath),
	mu:    &sync.RWMutex{},
}

// metadataBinder is responsible for binding columns and paths to schema
// metadata and access policies
type metadataBinder struct {
	accessPolicy         mdl.AccessPolicy
	rootResourceMetadata mdl.TableMetadata
	rootAccessPolicy     mdl.TableAccessPolicy

	pathIdBuilder pathIdBuilder
	pathParser    *pathParser
}

// NewMetadataBinder initialises a new metadata binder
func NewMetadataBinder(
	rootMetadata mdl.TableMetadata,
	accessPolicy mdl.AccessPolicy,
) (*metadataBinder, error) {
	if accessPolicy == nil {
		return nil, qerr.InternalErr(
			"unable to create new MetadataBinder: accessPolicy cannot be nil",
		)
	}

	rootAccessPolicy, err := getTableAccessPolicy(accessPolicy, rootMetadata)
	if err != nil {
		return nil, qerr.InternalErr(
			"unable to create new MetadataBinder: %w",
			err,
		)
	}

	return &metadataBinder{
		accessPolicy:         accessPolicy,
		rootResourceMetadata: rootMetadata,
		rootAccessPolicy:     rootAccessPolicy,
		pathIdBuilder: pathIdBuilder{
			root: rootMetadata.Name,
			sb:   &strings.Builder{},
		},
		pathParser: newPathParser(),
	}, nil
}

// ResolveColumn returns a resolved column for the given column path - Errors
// are returned if the column path is invalid or access is not permitted by the
// access policy
func (b *metadataBinder) ResolveColumn(columnPath string) (mdl.ResolvedColumn, error) {
	pathSegments, err := b.pathParser.parsePath(columnPath)
	if err != nil {
		return mdl.ResolvedColumn{}, err
	}

	pathLen := len(pathSegments)
	if pathLen == 0 {
		return mdl.ResolvedColumn{}, qerr.SyntaxErr(
			"unable to resolve column: invalid column path: '%s'", columnPath,
		)
	}

	columnName := pathSegments[pathLen-1]
	pathToColumn := pathSegments[0 : pathLen-1]

	resolvedPath, err := b.resolvePath(pathToColumn, b.rootResourceMetadata)
	if err != nil {
		return mdl.ResolvedColumn{}, err
	}

	resource := resolvedPath.EndResource
	columnMetadata, exists := resource.Columns[columnName]
	if !exists {
		return mdl.ResolvedColumn{}, qerr.BindingErr(
			"table %s does not include a column definition for %s",
			resource.Name,
			columnName,
		)
	}

	accessPolicy, err := getTableAccessPolicy(
		b.accessPolicy,
		resource,
	)
	if err != nil {
		return mdl.ResolvedColumn{}, qerr.InternalErr("unable to resolve column: %w", err)
	}

	if err := validateColumnAccess(
		accessPolicy,
		resolvedPath.EndResource,
		columnMetadata,
	); err != nil {
		return mdl.ResolvedColumn{}, qerr.InternalErr("unable to resolve column: %w", err)
	}

	return mdl.ResolvedColumn{
		ResolvedPath: resolvedPath,
		Metadata:     columnMetadata,
	}, nil
}

// ResolvePath returns a resolved path for a given path string. Errors are
// returned if the path is invalid or the path traversal is not permitted by the
// access policy
func (b *metadataBinder) ResolvePath(pathString string) (mdl.ResolvedPath, error) {
	pathSegments, err := b.pathParser.parsePath(pathString)
	if err != nil {
		return mdl.ResolvedPath{}, err
	}

	pathLen := len(pathSegments)
	if pathLen == 0 {
		return mdl.ResolvedPath{}, qerr.SyntaxErr("unable to resolve path: '%s'", pathString)
	}

	resolvedPath, err := b.resolvePath(pathSegments, b.rootResourceMetadata)
	if err != nil {
		return mdl.ResolvedPath{}, err
	}

	return resolvedPath, nil
}

// Resolve relationship returns a resolved relationship for a given relationship
// name - Errors are thrown for invalid relationship names or if access to
// either the from or to columns is not permitted under the access policy
func (b *metadataBinder) ResolveRelationship(resource mdl.TableMetadata, relationshipName string) (mdl.TraversalStep, error) {
	relationshipData, exists := resource.Relationships[relationshipName]
	if !exists {
		return mdl.TraversalStep{}, qerr.BindingErr(
			"table %s does not include a relationship definition for %s",
			resource.Name,
			relationshipName,
		)
	}

	b.pathIdBuilder.resetToRoot()
	b.pathIdBuilder.appendToPath(relationshipData)
	step := mdl.TraversalStep{
		SubPathId:    b.pathIdBuilder.string(),
		Relationship: relationshipData,
	}

	if err := b.validateTraversalPermissions(step); err != nil {
		return mdl.TraversalStep{}, err
	}

	return step, nil
}

// resolvePath returns a ResolvedPath for a given set of path segments. Errors
// are returned if the path is invalid or path traversal is not permitted under
// the access policy
func (b *metadataBinder) resolvePath(
	pathSegments []string,
	rootResource mdl.TableMetadata,
) (mdl.ResolvedPath, error) {
	// final pathId
	traversalPathLength := len(pathSegments)
	finalPathId := rootResource.Name
	if traversalPathLength > 0 {
		finalPathId = fmt.Sprintf(
			"%s/%s",
			rootResource.Name,
			strings.Join(pathSegments, "/"),
		)
	}

	// paths are lazy loaded into a store
	if path, exists := pathStore.readPath(finalPathId); exists {
		return path, nil
	}

	// initialise resolved path
	o := mdl.ResolvedPath{
		StartResource: rootResource,
		EndResource:   rootResource,
	}

	// if path has no length return early
	if traversalPathLength == 0 {
		o.Id = rootResource.Name
		pathStore.writePath(finalPathId, o)
		return o, nil
	}

	// prepare path id builder
	b.pathIdBuilder.resetToRoot()

	// Traverse through intermediate steps
	i := 0
	o.Steps = make([]*mdl.TraversalStep, traversalPathLength)
	for i < traversalPathLength {
		// Get relationship data
		relationshipName := pathSegments[i]
		relationshipData, exists := o.EndResource.Relationships[relationshipName]
		if !exists {
			return o, qerr.BindingErr(
				"%s is not a valid relationship on the '%s' table",
				relationshipName,
				o.EndResource.FullyQualifiedName,
			)
		}

		// validate relationship type
		relationshipType := relationshipData.Type
		isIntermediateStep := i < traversalPathLength-1
		if isIntermediateStep && relationshipType == mdl.RelationshipOneToMany {
			return o, qerr.BindingErr(
				"%s is a 1:N relationship and cannot be used as an "+
					"intermediate path step, please use a collection operator",
				relationshipData.ColumnName,
			)
		}

		// update the path id
		b.pathIdBuilder.appendToPath(relationshipData)

		// append the traversal step to the resolved path
		o.Steps[i] = &mdl.TraversalStep{
			SubPathId:    b.pathIdBuilder.string(),
			Relationship: relationshipData,
		}

		// validate user has permissions to make the traversal
		err := b.validateTraversalPermissions(*o.Steps[i])
		if err != nil {
			return o, err
		}

		// update the final end resource and relationship type
		o.EndResource = relationshipData.To.GetMetadata()
		o.Type = relationshipType
		i++
	}

	// Set the final pathid and return
	o.Id = b.pathIdBuilder.string()
	if o.Id != finalPathId {
		return o, qerr.InternalErr(
			"unable to resolve path. There is a disconnect between the " +
				"projected final path id and the built final path id",
		)
	}

	pathStore.writePath(o.Id, o)
	return o, nil
}

// validateTraversalPermissions returns validates that a given traversal is
// permitted under the access policy - A traversal is permitted if access is
// granted for both the from and to columns - An error is returned if access is
// not permitted, else nil
func (b *metadataBinder) validateTraversalPermissions(step mdl.TraversalStep) error {
	fromResourceMetadata := step.Relationship.From.GetMetadata()
	fromTableAccessPolicy, err := getTableAccessPolicy(
		b.accessPolicy,
		fromResourceMetadata,
	)
	if err != nil {
		return err
	}

	err = validateColumnAccess(
		fromTableAccessPolicy,
		fromResourceMetadata,
		step.Relationship.FromColumn,
	)
	if err != nil {
		return err
	}

	toResourceMetadata := step.Relationship.To.GetMetadata()
	toTableAccessPolicy, err := getTableAccessPolicy(
		b.accessPolicy,
		toResourceMetadata,
	)
	if err != nil {
		return err
	}
	err = validateColumnAccess(
		toTableAccessPolicy,
		toResourceMetadata,
		step.Relationship.ToColumn,
	)
	if err != nil {
		return err
	}

	return nil
}

// validateColumnAccess validates that access is permitted for a give column
// under the access policy. An error is returned if access is not permitted,
// else nil
func validateColumnAccess(
	tableAccessPolicy mdl.TableAccessPolicy,
	tableData mdl.TableMetadata,
	columnData mdl.ColumnMetadata,
) error {
	canAccessColumn, err := tableAccessPolicy.CanAccessColumn(columnData.Name)
	if err != nil {
		return qerr.InternalErr("unable to validate column access: %w", err)
	}

	if !canAccessColumn {
		return qerr.AccessErr(
			"you do not have permission to access the %s column on the %s table",
			columnData.Name,
			tableData.Name,
		)
	}
	return nil
}

// getTableAccessPolicy returns an access policy for a given table - An error is
// returned if the access policy cannot be accessed or if access to the table is
// not permitted under the access policy
func getTableAccessPolicy(
	accessPolicy mdl.AccessPolicy,
	tableData mdl.TableMetadata,
) (mdl.TableAccessPolicy, error) {
	if accessPolicy == nil {
		return nil, qerr.InternalErr("access policy cannot be nil")
	}

	tablePolicy, exists := accessPolicy.GetTableAccessPolicy(tableData.Name)
	if !exists {
		return nil, qerr.InternalErr("unable to find access policy for the %s table", tableData.Name)
	}

	if !tablePolicy.CanAccess() {
		return nil, qerr.AccessErr("you do not have permission to access the %s table", tableData.Name)
	}
	return tablePolicy, nil
}
