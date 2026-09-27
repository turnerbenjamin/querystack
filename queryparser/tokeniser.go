// Package queryparser contains query string parsing functionality
//
// This file contains Tokeniser which is used to tokenise query strings into
// tokens
package queryparser

import (
	"fmt"
	"strings"

	qerr "github.com/turnerbenjamin/querystack/queryerror"
	mdl "github.com/turnerbenjamin/querystack/querymodel"
)

// tokenType identifies the kinds of tokens recognised by the tokeniser.
type tokenType uint64

const (
	// TokenEmpty is an invalid  or uninitialised token type.
	TokenEmpty tokenType = iota

	// TokenEOF represents the end of the input.
	TokenEOF

	// TokenSpace represents one or more consecutive whitespace characters.
	TokenSpace

	// TokenLogicalOperator represents a recognised logical operator.
	TokenLogicalOperator

	// TokenComparisonOperator represents a recognised comparison operator.
	TokenComparisonOperator

	// TokenCollectionOperator represents a recognised collection operator.
	TokenCollectionOperator

	// TokenSortDirectionOperator represents a recognised sort direction
	// operator.
	TokenSortDirectionOperator

	// TokenNull represents the null literal.
	TokenNull

	// TokenNullByte represents a null byte character.
	TokenNullByte

	// TokenIdentifier represents an identifier that does not match a more
	// specific token type.
	TokenIdentifier

	// TokenBool represents a boolean literal
	TokenBool

	// TokenNumberRaw represents a numeric literal that has not yet been
	// converted to a numeric value.
	TokenNumberRaw

	// TokenStringRaw represents a string literal that has not yet been
	// validated or unquoted.
	TokenStringRaw

	// TokenParenL represents an opening parenthesis.
	TokenParenL

	// TokenParenR represents a closing parenthesis.
	TokenParenR

	// TokenAmpersand represents an ampersand.
	TokenAmpersand

	// TokenSemiColon represents a semicolon.
	TokenSemiColon

	// TokenComma represents a comma.
	TokenComma

	// TokenEquals represents an equals sign.
	TokenEquals

	// TokenSlash represents a forward slash.
	TokenSlash

	// TokenHyphen represents a hyphen.
	TokenHyphen
)

// singleCharTokenMap maps characters that terminate identifiers to their
// corresponding token types.
var singleCharTokenMap = map[byte]tokenType{
	'(':    TokenParenL,
	')':    TokenParenR,
	'&':    TokenAmpersand,
	';':    TokenSemiColon,
	',':    TokenComma,
	'=':    TokenEquals,
	'/':    TokenSlash,
	'-':    TokenHyphen,
	'\x00': TokenNullByte,
}

// token represents a single token produced by the tokeniser.
type token struct {
	endIdx int
	Type   tokenType
	Value  string
}

// Tokeniser tokenises query string input and provides sequential and lookahead
// access to the resulting tokens.
type Tokeniser struct {
	input   string
	idx     int
	buffIdx int
	buffer  []token
}

// NewTokeniser creates a Tokeniser for the supplied query string.
func NewTokeniser(input string) *Tokeniser {
	return &Tokeniser{
		input:   input,
		buffer:  make([]token, 0, 64),
		buffIdx: -1,
	}
}

// newTkn creates a token using the tokeniser's current input position as the
// token's end position.
func (t *Tokeniser) newTkn(tType tokenType, v string) token {
	return token{
		endIdx: t.idx,
		Type:   tType,
		Value:  v,
	}
}

// Peek returns the next token without advancing the tokeniser.
func (t *Tokeniser) Peek() token {
	return t.PeekN(1)
}

// PeekN returns the nth upcoming token without advancing the tokeniser.
func (t *Tokeniser) PeekN(n int) token {
	o := t.newTkn(TokenEmpty, "")
	savedBuffIdx := t.buffIdx
	for range n {
		o = t.Next()
	}
	t.buffIdx = savedBuffIdx
	return o
}

// Next returns the next token, skipping whitespace tokens.
func (t *Tokeniser) Next() token {
	return t.next(true)
}

// NextIncSpace returns the next token without skipping whitespace tokens.
func (t *Tokeniser) NextIncSpace() token {
	return t.next(false)
}

// next returns the next token, optionally skipping whitespace tokens.
func (t *Tokeniser) next(doSkipWhitespace bool) token {
	for (t.buffIdx + 1) < len(t.buffer) {
		t.buffIdx++
		if t.buffer[t.buffIdx].Type != TokenSpace || !doSkipWhitespace {
			return t.buffer[t.buffIdx]
		}
	}

	// collapse whitespace into single space character
	if t.idx < len(t.input) && isWhiteSpace(t.input[t.idx]) {
		for t.idx < len(t.input) && isWhiteSpace(t.input[t.idx]) {
			t.idx++
		}
		t.buffer = append(t.buffer, t.newTkn(TokenSpace, ""))
		t.buffIdx++

		if !doSkipWhitespace {
			return t.buffer[t.buffIdx]
		}
	}

	if t.idx >= len(t.input) {
		t.idx++

		t.buffer = append(t.buffer, t.newTkn(TokenEOF, ""))
		t.buffIdx++

		return t.buffer[t.buffIdx]
	}

	var next = token{Type: TokenEmpty}
	b := t.input[t.idx]
	if tt, exists := singleCharTokenMap[b]; exists {
		t.idx++
		next = t.newTkn(tt, string(b))
	} else {
		switch {
		case b == '\'', b == '"':
			next = t.readString()
		case b >= '0' && b <= '9':
			next = t.readNumber()
		default:
			next = t.readIdentifier()
		}
	}

	t.buffer = append(t.buffer, next)
	t.buffIdx++
	return t.buffer[t.buffIdx]
}

// readString reads a quoted string literal and returns it as a raw string
// token.
//
// The returned token includes the opening and, when present, closing
// quotation marks. Validation of the quotation marks and string contents is
// left to the consumer.
func (t *Tokeniser) readString() token {
	openingQuotationChar := t.input[t.idx]
	start := t.idx
	t.idx++

	for t.idx < len(t.input) {
		b := t.input[t.idx]
		t.idx++

		if b == openingQuotationChar {
			break
		}
	}

	return t.newTkn(TokenStringRaw, t.input[start:t.idx])
}

// readNumber reads a numeric literal containing digits and an optional decimal
// point.
//
// The tokeniser performs only basic lexical parsing; validation and numeric
// conversion are performed by the consumer. A trailing decimal point is
// normalised by appending a zero.
func (t *Tokeniser) readNumber() token {
	start := t.idx

	dpSeen := false
	for t.idx < len(t.input) {
		b := t.input[t.idx]
		if (b < '0' || b > '9') && b != '.' {
			break
		}

		if b == '.' {
			if dpSeen {
				break
			}
			dpSeen = true
		}

		t.idx++
	}

	numString := t.input[start:t.idx]

	if strings.HasSuffix(numString, ".") {
		numString = numString + "0"
	}

	return t.newTkn(TokenNumberRaw, numString)
}

// readIdentifier reads a sequence of characters that forms an identifier,
// stopping at whitespace, recognised single-character tokens, or quotation
// marks.
//
// The resulting identifier is classified into a more specific token type when
// it matches a supported operator or literal.
func (t *Tokeniser) readIdentifier() token {
	start := t.idx

	for t.idx < len(t.input) {
		b := t.input[t.idx]

		if isWhiteSpace(b) {
			break
		}

		if _, exists := singleCharTokenMap[b]; exists {
			break
		}

		if b == '\'' || b == '"' {
			break
		}

		t.idx++
	}

	value := t.input[start:t.idx]
	tokenType, formattedValue := classifyIdentifier(value)

	return t.newTkn(tokenType, formattedValue)
}

// classifyIdentifier determines the token type for an identifier by matching
// it against the supported operators and recognised literal values.
//
// Identifiers are matched case-insensitively for recognised values and
// operators. Unmatched identifiers retain their original value and are
// classified as TokenIdentifier.
func classifyIdentifier(identifier string) (tokenType, string) {
	lIdentifier := strings.ToLower(identifier)

	if _, isLogicalOperation := mdl.SupportedLogicalOperators[lIdentifier]; isLogicalOperation {
		return TokenLogicalOperator, lIdentifier
	}

	if _, isComparisonOperation := mdl.SupportedComparisonOperators[lIdentifier]; isComparisonOperation {
		return TokenComparisonOperator, lIdentifier
	}

	if _, isCollectionOperation := mdl.SupportedCollectionOperators[lIdentifier]; isCollectionOperation {
		return TokenCollectionOperator, lIdentifier
	}

	if _, isSortDirectionOperator := supportedSortDirectionOperators[lIdentifier]; isSortDirectionOperator {
		return TokenSortDirectionOperator, lIdentifier
	}

	if lIdentifier == "null" {
		return TokenNull, lIdentifier
	}

	if lIdentifier == "true" || lIdentifier == "false" {
		return TokenBool, lIdentifier
	}

	return TokenIdentifier, identifier
}

// isWhiteSpace reports whether b is a whitespace character recognised by the
// query tokeniser.
func isWhiteSpace(b byte) bool {
	switch b {
	case '\t', '\n', '\v', '\f', '\r', ' ':
		return true
	default:
		return false
	}
}

// TknErr creates a syntax error associated with a token.
//
// For ordinary tokens, the error includes a short substring of the input ending
// at the offending token to provide additional context. Empty and EOF tokens do
// not have a meaningful input position, so no context is added.
func (t Tokeniser) TknErr(tkn token, m string, a ...any) error {
	if tkn.Type == TokenEmpty || tkn.Type == TokenEOF {
		return qerr.SyntaxErr(m, a...)
	}

	em := fmt.Sprintf(m, a...)

	maxCtxLen := 50
	ctx := t.input[max(0, tkn.endIdx-maxCtxLen):min(len(t.input), tkn.endIdx)]

	return qerr.SyntaxErr("%s: __%s <--", em, ctx)
}
