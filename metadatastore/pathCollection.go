// Package metadatastore is responsible for binding column/path/relationship
// identifiers to schema metadata
//
// This file contains pathCollection which is used in the package to persist
// resolved paths to reduce the number of objects created at runtime
package metadatastore

import (
	"sync"

	mdl "github.com/turnerbenjamin/querystack/querymodel"
)

// pathCollection is a thread safe mapping of path identifiers to resolved paths
type pathCollection struct {
	store map[string]mdl.ResolvedPath
	mu    *sync.RWMutex
}

// readPath returns a path from the collection or an empty string if not found
// and a bool indicating whether the path exists in the store
func (c *pathCollection) readPath(pathId string) (mdl.ResolvedPath, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	p, ok := c.store[pathId]
	return p, ok
}

// write path to the collection
func (c *pathCollection) writePath(pathId string, path mdl.ResolvedPath) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store[pathId] = path
}
