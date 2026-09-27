// Package paginationtokens is responsible for generating pagination tokens to
// enable cursor pagination
//
// This file contains readerBuf which is a small, internal helper struct used
// when deserialising paging tokens
package paginationtokens

import (
	"io"

	qerr "github.com/turnerbenjamin/querystack/queryerror"
)

// readerBuf is an internal helper used to read a byte buffer
type readerBuf struct {
	bytes  []byte
	offset uint64
	len    uint64
}

// newBuf initialises a new readerBuf instance
func newBuf(d []byte) *readerBuf {
	return &readerBuf{
		bytes:  d,
		offset: 0,
		len:    uint64(len(d)),
	}
}

// readByte reads the next byte from the buffer and moves the cursor forwards
func (buf *readerBuf) readByte() (byte, error) {
	if buf.offset >= buf.len {
		return 0, io.EOF
	}

	b := buf.bytes[buf.offset]
	buf.offset++
	return b, nil
}

// readBytes reads n bytes from the buffer and moves the cursor forwards
func (buf *readerBuf) readBytes(n uint64) ([]byte, error) {
	if buf.offset+n >= uint64(len(buf.bytes)) {
		return nil, qerr.InternalErr("unable to read bytes: out of range")
	}
	slice := buf.bytes[buf.offset : buf.offset+n]
	buf.offset += n
	return slice, nil
}

// hasNextByte returns true if there are additional bytes to read in the buffer
func (buf *readerBuf) hasNextByte() bool {
	return buf.offset < buf.len
}
