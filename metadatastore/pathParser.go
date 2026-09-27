// Package metadatastore is responsible for binding column/path/relationship
// identifiers to schema metadata
//
// This file contains pathParser which is used to split path components into a
// string slice - A shared slice is used to reduce memory allocations in a given
// request thread
package metadatastore

import (
	qerr "github.com/turnerbenjamin/querystack/queryerror"
)

// pathParser parses path strings into their individual path segments.
//
// A pathParser is intended for sequential, non-concurrent use. The backing
// array for the returned segments is reused between calls to parsePath.
type pathParser struct {
	segments []string
}

// newPathParser creates a pathParser
func newPathParser() *pathParser {
	return &pathParser{
		segments: make([]string, 0, 8),
	}
}

// parsePath parses pathString into its individual path segments.
//
// Empty path segments and trailing slashes are rejected. The returned slice
// references storage owned by the parser and must not be retained after the
// parser is reused.
func (pp *pathParser) parsePath(pathString string) ([]string, error) {
	if pp.segments == nil {
		pp.segments = make([]string, 0, 8)
	}

	pp.segments = pp.segments[:0]

	wordStart := 0
	for i := 0; i < len(pathString); i++ {
		if pathString[i] != '/' {
			continue
		}

		if i == wordStart {
			return nil, qerr.SyntaxErr("invalid path: empty path segment")
		}

		if i == len(pathString)-1 {
			return nil, qerr.SyntaxErr("invalid path: trailing slashes are not permitted")
		}

		pp.segments = append(pp.segments, pathString[wordStart:i])
		wordStart = i + 1
	}

	if wordStart < len(pathString) {
		pp.segments = append(pp.segments, pathString[wordStart:])
	}

	return pp.segments, nil
}
