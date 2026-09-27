// Package metadatastore is responsible for binding column/path/relationship
// identifiers to schema metadata
//
// This file contains pathIdBuilder which is used to reuse memory allocations
// when building strings in a given request thread
package metadatastore

import (
	"strings"

	mdl "github.com/turnerbenjamin/querystack/querymodel"
)

// pathIdBuilder is used to construct path ids - This is safe for
// non-concurrent, sequential use only; the underlying strings builder
// is reused between calls to reduce memory allocations
type pathIdBuilder struct {
	root string
	sb   *strings.Builder
}

// resetToRoot resets the path builder to the root string. This should be called
// before starting a new path
func (b pathIdBuilder) resetToRoot() {
	b.sb.Reset()
	b.sb.WriteString(b.root)
}

// appendToPath appends a relationship column to the path
func (b pathIdBuilder) appendToPath(relationship mdl.RelationshipMetadata) {
	b.sb.WriteByte('/')
	b.sb.WriteString(relationship.ColumnName)
}

// string returns the pathId as a string
func (b pathIdBuilder) string() string {
	return b.sb.String()
}
