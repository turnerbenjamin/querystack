// Package paginationTokens is responsible for generating pagination tokens to
// enable cursor pagination
//
// This file contains the top-level logic for building and parsing paging tokens
package paginationTokens

import (
	qstore "github.com/turnerbenjamin/querystack/queryDataStore"
	qerr "github.com/turnerbenjamin/querystack/queryError"
	mdl "github.com/turnerbenjamin/querystack/queryModel"
)

// pagingTokenSchemaVersion represents the schema version of a paging token for
// the purpose of decoding
var pagingTokenSchemaVersion uint32 = 1

// PagingToken represents a token using for cursor pagination
type PagingToken struct {

	// Version is the paging token schema version
	Version uint32

	// CursorValues contains the values, for each orderby field, on the last
	// record of the current query - They are used to construct a cursor based
	// pagination filter
	CursorValues []mdl.Value

	// ResourceName is the resource the query relates to
	ResourceName string

	// QueryString is the original query string for the query
	QueryString string
}

// pagingTokenBuilder is used to build query paging tokens
type pagingTokenBuilder struct {

	// payloadSigner is used by pagingTokenBuilder to sign the paging tokens
	payloadSigner mdl.PayloadSigner

	// payloadSecret is the secret used to sign paging tokens with
	payloadSecret []byte
}

// NewPagingTokenBuilder initialises a new pagingTokenBuilder
func NewPagingTokenBuilder(payloadSigner mdl.PayloadSigner, payloadSecret []byte) (*pagingTokenBuilder, error) {
	if payloadSigner == nil {
		return nil, qerr.InternalErr("unable to build next page token. payload signer cannot be nil")
	}

	if payloadSecret == nil {
		return nil, qerr.InternalErr("unable to build next page token. payload secret cannot be nil")
	}

	return &pagingTokenBuilder{
		payloadSigner: payloadSigner,
		payloadSecret: payloadSecret,
	}, nil
}

// BuildToken builds a new paging token string containing data used to fetch the
// next page of data with cursor pagination
func (b *pagingTokenBuilder) BuildToken(
	queryDataStore qstore.QueryDataStore,
	lastRecord mdl.TableModel,
) (string, error) {
	if b.payloadSigner == nil {
		return "", qerr.InternalErr("unable to build next page token. payload signer cannot be nil")
	}

	if b.payloadSecret == nil {
		return "", qerr.InternalErr("unable to build next page token. payload secret cannot be nil")
	}

	cursorValues, err := getCursorValues(queryDataStore, lastRecord)
	if err != nil {
		return "", err
	}

	payloadBytes, err := serialiseToken(
		queryDataStore,
		pagingTokenSchemaVersion,
		cursorValues,
	)
	if err != nil {
		return "", err
	}
	return b.payloadSigner.Sign(b.payloadSecret, payloadBytes), nil
}

// parses a given token string into a PagingToken for use in cursor pagination
func (b *pagingTokenBuilder) ParseToken(
	token string,
	valueBuilder mdl.ValueBuilder,
) (*PagingToken, error) {
	if b.payloadSigner == nil {
		return nil, qerr.InternalErr(
			"unable to build next page token. payload signer cannot be nil",
		)
	}

	if b.payloadSecret == nil {
		return nil, qerr.InternalErr(
			"unable to build next page token. payload secret cannot be nil",
		)
	}

	payloadBytes, ok := b.payloadSigner.Verify(b.payloadSecret, token)
	if !ok {
		return nil, qerr.PagingTokenErr(
			"the next page token is invalid",
		)
	}

	tokenPayload, err := deserialiseToken(
		payloadBytes,
		valueBuilder,
	)
	if err != nil {
		return nil, err
	}

	if tokenPayload.Version != pagingTokenSchemaVersion {
		return nil, qerr.PagingTokenErr(
			"the next page token has expired",
		)
	}

	return &tokenPayload, nil
}

// getCursorValues iterates through the order by rules of the query and accesses
// the value for the orderby field from the last record in the current page of
// results - This allows construction of a cursor filter
func getCursorValues(
	s qstore.QueryDataStore,
	lastRecord mdl.TableModel,
) ([]mdl.Value, error) {
	cursorValues := make([]mdl.Value, s.OrderByLen())

	i := 0
	for rule := range s.OrderBy() {
		nextRecordValue, err := lastRecord.GetValue(
			s.FilterExpressionBuilder().ValueBuilder(),
			rule.ResolvedColumn.ResolvedPath.Steps,
			rule.ResolvedColumn.Metadata.Name,
		)
		if err != nil {
			return nil, err
		}

		cursorValues[i] = nextRecordValue
		i++
	}
	return cursorValues, nil
}
