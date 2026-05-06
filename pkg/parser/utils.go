package parser

import (
	"slices"

	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model/ast"
)

func isFetchStage(item string) bool {
	return slices.Contains(model.Stages, item)
}

func isAligner(item string) bool {
	return slices.Contains(model.Aligners, item)
}

func isModifier(item string) bool {
	return slices.Contains(model.Modifiers, item)
}

func isEndOfFilter(token *model.Token, isEof bool) bool {
	return isEof || token.Type == model.TypeSeparator || token.Type == model.TypeSemicolon || token.Type == model.TypeRoundBracketRight
}

func isJoinKeyword(token *model.Token) bool {
	return token.Type == model.TypeKeyword && token.Value == "join"
}

func mapLiteralValue(token *model.Token) (ast.Literal, bool) {
	switch token.Type {
	case model.TypeString:
		return &ast.StringLiteral{Value: token.Value}, true
	case model.TypeInteger:
		return &ast.IntegerLiteral{Value: token.Value}, true
	case model.TypeFloat:
		return &ast.FloatLiteral{Value: token.Value}, true
	case model.TypeBoolean:
		return &ast.BooleanLiteral{Value: token.Value}, true
	default:
		return nil, false
	}
}

func isDoubleArgumentFunction(funcName string) bool {
	switch funcName {
	case "contains", "phrase_match", "pow", "percentile", "max", "min":
		return true
	default:
		return false
	}
}

func isSingleArgumentFunction(funcName string) bool {
	switch funcName {
	case "defined", "undefined", "dist_sum", "dist_count", "abs", "timestamp", "floor", "ceil", "round":
		return true
	default:
		return false
	}
}

var infixOperationPriorities map[model.TokenType]OperationPriority = map[model.TokenType]OperationPriority{
	model.TypeLogicalAnd:   LOGICAL,
	model.TypeLogicalOr:    LOGICAL,
	model.TypeEquals:       EQUALS,
	model.TypeAssign:       EQUALS,
	model.TypeNotEquals:    EQUALS,
	model.TypeMatchRegex:   EQUALS,
	model.TypeNotMachRegex: EQUALS,
	model.TypeLess:         COMPARE,
	model.TypeLessOrEquals: COMPARE,
	model.TypeMore:         COMPARE,
	model.TypeMoreOrEquals: COMPARE,
	model.TypeMul:          PRODUCT,
	model.TypeDiv:          PRODUCT,
	model.TypeAdd:          SUM,
	model.TypeDiff:         SUM,
}

func isInfixOperation(op model.TokenType) bool {
	_, exists := infixOperationPriorities[op]
	return exists
}

func mapInfixOperationToPriority(op model.TokenType) (OperationPriority, bool) {
	value, exists := infixOperationPriorities[op]
	if !exists {
		return LOWEST, false
	}
	return value, exists
}

func tryFixOperationType(op string) string {
	switch op {
	case "=":
		return "=="
	default:
		return op
	}
}

func minusOrEmpty(minus bool) string {
	if minus {
		return "-"
	}
	return ""
}

func getSingleTokenMetadata(token *model.Token) ast.Metadata {
	return ast.Metadata{SourceIndex: token.Index, SourceLength: token.Length}
}

func getMultipleTokensMetadata(firstToken *model.Token, lastToken *model.Token) ast.Metadata {
	return ast.Metadata{SourceIndex: firstToken.Index, SourceLength: lastToken.Index - firstToken.Index + lastToken.Length}
}
