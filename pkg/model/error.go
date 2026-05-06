package model

import "fmt"

type Error struct {
	Status       string
	SourceIndex  int
	SourceLength int
}

func NewError(status string, sourceIndex, sourceLength int) *Error {
	return &Error{Status: status, SourceIndex: sourceIndex, SourceLength: sourceLength}
}

func NewErrorForSingleToken(status string, token *Token) *Error {
	return &Error{Status: status, SourceIndex: token.Index, SourceLength: token.Length}
}

func NewErrorForMultipleTokens(status string, firstToken, lastToken *Token) *Error {
	return &Error{Status: status, SourceIndex: firstToken.Index, SourceLength: lastToken.Index - firstToken.Index + lastToken.Length}
}

func (err *Error) Error() string {
	return fmt.Sprintf("%s at position %d (length: %d)", err.Status, err.SourceIndex, err.SourceLength)
}
