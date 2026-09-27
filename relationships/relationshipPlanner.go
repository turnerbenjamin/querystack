// Package relationships plans aliases and SQL joins for traversing resource
// relationships in query paths.
package relationships

import (
	"fmt"
	"iter"

	mdl "github.com/turnerbenjamin/querystack/querymodel"
)

// aliasType identifies the kind of alias mapping being managed.
type aliasType uint8

const (
	// aliasTypeExists identifies aliases used for EXISTS expressions.
	aliasTypeExists aliasType = iota

	// aliasTypeJoin identifies aliases used for SQL joins.
	aliasTypeJoin
)

// Join describes a planned relationship join and any nested joins beneath it.
type Join struct {
	ParentAlias string
	Alias       string
	Step        mdl.TraversalStep
	SubJoins    JoinCollection
}

// JoinCollection stores planned joins indexed by traversal path Id.
type JoinCollection struct {
	joins map[string]*Join
}

// addJoin adds or replaces the join associated with the given path Id.
func (c *JoinCollection) addJoin(pathId string, join *Join) {
	if c.joins == nil {
		c.joins = map[string]*Join{}
	}
	c.joins[pathId] = join
}

// Joins returns an iterator over the planned joins and their path Ids.
func (c *JoinCollection) Joins() iter.Seq2[string, *Join] {
	return func(yield func(string, *Join) bool) {
		for k, v := range c.joins {
			if !yield(k, v) {
				return
			}
		}
	}
}

// JoinsLen returns the number of planned joins.
func (c *JoinCollection) JoinsLen() int {
	return len(c.joins)
}

// AliasStore manages SQL aliases for resource paths and relationship
// traversals.
type AliasStore struct {
	aliasCount             int
	rootAlias              string
	pathIdToExistsAliasMap map[string]string
	pathIdToJoinsAliasMap  map[string]string
}

// GetRootAlias returns the alias used for the root resource.
func (s *AliasStore) GetRootAlias() string {
	return s.rootAlias
}

// GetExistsAlias returns the EXISTS alias associated with a resolved path.
func (s *AliasStore) GetExistsAlias(path mdl.ResolvedPath) (string, bool) {
	return s.getAlias(aliasTypeExists, path)
}

// GetJoinAlias returns the JOIN alias associated with a resolved path.
func (s *AliasStore) GetJoinAlias(path mdl.ResolvedPath) (string, bool) {
	return s.getAlias(aliasTypeJoin, path)
}

// getAlias returns the alias of the requested type associated with a path.
func (s *AliasStore) getAlias(t aliasType, path mdl.ResolvedPath) (string, bool) {
	if len(path.Steps) == 0 {
		return s.rootAlias, true
	}
	pathId := path.Id

	mapping := s.getAliasMapping(t)
	if _, exists := mapping[pathId]; !exists {
		return "", false
	}

	return mapping[pathId], true
}

// nextAlias returns the existing alias for a path or creates a new one.
func (s *AliasStore) nextAlias(t aliasType, pathId string) string {
	mapping := s.getAliasMapping(t)
	if _, exists := mapping[pathId]; !exists {
		s.aliasCount++
		mapping[pathId] = fmt.Sprintf("t%d", s.aliasCount)
	}

	return mapping[pathId]
}

// getAliasMapping returns the alias mapping for the requested alias type.
func (s *AliasStore) getAliasMapping(t aliasType) map[string]string {
	switch t {
	case aliasTypeExists:
		if s.pathIdToExistsAliasMap == nil {
			s.pathIdToExistsAliasMap = make(map[string]string, 1)
		}
		return s.pathIdToExistsAliasMap
	case aliasTypeJoin:
		if s.pathIdToJoinsAliasMap == nil {
			s.pathIdToJoinsAliasMap = make(map[string]string, 1)
		}
		return s.pathIdToJoinsAliasMap
	default:
		panic("invalid alias type received")
	}
}

// RelationshipPlanner builds aliases and join structures for relationship
// paths.
type RelationshipPlanner struct {
	Aliases   AliasStore
	JoinStore JoinCollection
}

// NewRelationshipPlanner creates a relationship planner for the given root
// resource.
func NewRelationshipPlanner(rootResource mdl.TableMetadata) (*RelationshipPlanner, error) {
	planner := &RelationshipPlanner{
		Aliases:   AliasStore{rootAlias: rootResource.Name},
		JoinStore: JoinCollection{},
	}

	return planner, nil
}

// ProcessExists creates the EXISTS nodes required to traverse the given path.
func (p *RelationshipPlanner) ProcessExists(path mdl.ResolvedPath) []mdl.ExistsNode {
	existsNodes := make([]mdl.ExistsNode, len(path.Steps))

	parentAlias := p.Aliases.rootAlias
	for i, step := range path.Steps {
		nextAlias := p.Aliases.nextAlias(aliasTypeExists, step.SubPathId)

		existsNodes[i] = mdl.ExistsNode{
			Step:        step,
			Alias:       nextAlias,
			ParentAlias: parentAlias,
		}
		parentAlias = nextAlias
	}
	return existsNodes
}

// ProcessJoin adds the joins required to traverse the given path.
func (p *RelationshipPlanner) ProcessJoin(path mdl.ResolvedPath) error {
	pathLen := len(path.Steps)
	if pathLen == 0 {
		return nil
	}

	// Handle initial join
	step := path.Steps[0]
	currentJoin, exists := p.JoinStore.joins[step.SubPathId]
	if !exists {
		currentJoin = &Join{
			ParentAlias: p.Aliases.rootAlias,
			Alias:       p.Aliases.nextAlias(aliasTypeJoin, step.SubPathId),
			Step:        *step,
		}
		p.JoinStore.addJoin(step.SubPathId, currentJoin)
	}
	if pathLen < 2 {
		return nil
	}

	for _, step := range path.Steps[1:] {
		nextJoin, exists := currentJoin.SubJoins.joins[step.SubPathId]
		if !exists {
			nextJoin = &Join{
				ParentAlias: currentJoin.Alias,
				Alias:       p.Aliases.nextAlias(aliasTypeJoin, step.SubPathId),
				Step:        *step,
			}
			currentJoin.SubJoins.addJoin(step.SubPathId, nextJoin)
		}
		currentJoin = nextJoin
	}
	return nil
}
