package model

type TokenType string

type Token struct {
	Type   TokenType
	Value  string
	Index  int
	Length int
}

const (
	// data types
	TypeKeyword          TokenType = "keyword"
	TypeIdentifier       TokenType = "identifier"
	TypeTemplateVariable TokenType = "template_variable"
	TypeString           TokenType = "string"
	TypeInteger          TokenType = "integer"
	TypeFloat            TokenType = "float"
	TypeBoolean          TokenType = "boolean"
	TypeDuration         TokenType = "duration"

	// operators
	TypeMul                TokenType = "*"
	TypeDiv                TokenType = "/"
	TypeAdd                TokenType = "+"
	TypeDiff               TokenType = "-"
	TypeRoundBracketLeft   TokenType = "("
	TypeRoundBracketRight  TokenType = ")"
	TypeSquareBracketLeft  TokenType = "["
	TypeSquareBracketRight TokenType = "]"
	TypeSemicolon          TokenType = ";"
	TypeComma              TokenType = ","
	TypeLogicalAnd         TokenType = "&&"
	TypeNotMachRegex       TokenType = "!~"
	TypeNotEquals          TokenType = "!="
	TypeLogicalNot         TokenType = "!"
	TypeLogicalOr          TokenType = "||"
	TypeSeparator          TokenType = "|"
	TypeEquals             TokenType = "=="
	TypeMatchRegex         TokenType = "=~"
	TypeAssign             TokenType = "="
	TypeLessOrEquals       TokenType = "<="
	TypeLess               TokenType = "<"
	TypeMoreOrEquals       TokenType = ">="
	TypeMore               TokenType = ">"

	// misc
	TypeComment TokenType = "comment"

	// special types
	TypeEOF     TokenType = "eof"
	TypeUnknown TokenType = "unknown"
	TypeAny     TokenType = "any"
)
