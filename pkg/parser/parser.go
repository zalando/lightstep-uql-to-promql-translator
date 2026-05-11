package parser

import (
	"sort"

	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model/ast"
)

type OperationPriority int

const (
	PARSER_MAX_ITERATIONS = 1000

	_ OperationPriority = iota
	LOWEST
	LOGICAL // &&, ||
	EQUALS  // ==, !=, ...
	COMPARE // >, <, >=, <=, ...
	SUM     // +, -
	PRODUCT // *, /
	PREFIX  // prefix -, prefix !
)

type Parser struct {
	input []model.Token
	index int
}

func New(input []model.Token) *Parser {
	return &Parser{
		input: input,
		index: 0,
	}
}

func (parser *Parser) isCurrentTokenEof() bool {
	return parser.index >= len(parser.input)
}

func (parser *Parser) currentToken() (*model.Token, bool) {
	if parser.isCurrentTokenEof() {
		return nil, true
	}
	return &parser.input[parser.index], false
}

func (parser *Parser) move() {
	if parser.isCurrentTokenEof() {
		return
	}
	parser.index += 1
}

func (parser *Parser) parseFetchStageLogs() (*ast.FetchStageLogs, *model.Error) {
	stage, _ := parser.currentToken()
	parser.move()
	token, isEof := parser.currentToken()
	if isEndOfFilter(token, isEof) {
		return &ast.FetchStageLogs{
			FetchType: "",
			Metadata:  getSingleTokenMetadata(stage),
		}, nil
	}
	if token.Type != model.TypeKeyword && token.Value != "count" {
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	parser.move()
	return &ast.FetchStageLogs{
		FetchType: ast.LogsFetchTypeCount,
		Metadata:  getMultipleTokensMetadata(stage, token),
	}, nil
}

func (parser *Parser) parseFetchStageMetric() (*ast.FetchStageMetric, *model.Error) {
	stage, _ := parser.currentToken()
	parser.move()
	token, isEof := parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(stage)
	}
	if token.Type != model.TypeIdentifier && token.Type != model.TypeTemplateVariable {
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	parser.move()
	return &ast.FetchStageMetric{
		MetricName: token.Value,
		Metadata:   getMultipleTokensMetadata(stage, token),
	}, nil
}

func (parser *Parser) parseFetchStageSpans() (*ast.FetchStageSpans, *model.Error) {
	stage, _ := parser.currentToken()
	parser.move()
	token, isEof := parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(stage)
	}
	if token.Type != model.TypeKeyword && token.Type != model.TypeIdentifier {
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	parser.move()
	return &ast.FetchStageSpans{
		FetchType: token.Value,
		Metadata:  getMultipleTokensMetadata(stage, token),
	}, nil
}

func (parser *Parser) parseFetchStageConstant() (*ast.FetchStageConstant, *model.Error) {
	stage, _ := parser.currentToken()
	parser.move()
	token, isEof := parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(stage)
	}
	literalValue, isLiteralValue := mapLiteralValue(token)
	if !isLiteralValue {
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	parser.move()
	return &ast.FetchStageConstant{
		Value:    literalValue,
		Metadata: getMultipleTokensMetadata(stage, token),
	}, nil
}

func (parser *Parser) parseFetchStage() (ast.FetchStage, *model.Error) {
	token, _ := parser.currentToken()
	switch token.Value {
	case "logs":
		return parser.parseFetchStageLogs()
	case "metric", "fetch":
		return parser.parseFetchStageMetric()
	case "spans":
		return parser.parseFetchStageSpans()
	case "constant":
		return parser.parseFetchStageConstant()
	default:
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
}

func (parser *Parser) parseAlignerDelta() (*ast.AlignerStageDelta, *model.Error) {
	stage, _ := parser.currentToken()
	parser.move()

	tokenA, isEof := parser.currentToken()
	if isEndOfFilter(tokenA, isEof) {
		return &ast.AlignerStageDelta{
			Metadata: getSingleTokenMetadata(stage),
		}, nil
	}
	if tokenA.Type != model.TypeDuration && tokenA.Type != model.TypeTemplateVariable {
		return nil, newParserError_unexpected_token_X_of_type_Y(tokenA)
	}
	parser.move()

	tokenB, isEof := parser.currentToken()
	if isEndOfFilter(tokenB, isEof) {
		return &ast.AlignerStageDelta{
			InputWindow: &ast.DurationLiteral{Value: tokenA.Value},
			Metadata:    getMultipleTokensMetadata(stage, tokenA),
		}, nil
	}
	if tokenB.Type != model.TypeComma {
		return nil, newParserError_unexpected_token_X_of_type_Y(tokenB)
	}
	parser.move()

	tokenC, isEof := parser.currentToken()
	if isEndOfFilter(tokenC, isEof) {
		return nil, newParserError_unexpected_EOF(tokenB)
	}
	if tokenC.Type != model.TypeDuration && tokenC.Type != model.TypeTemplateVariable {
		return nil, newParserError_unexpected_token_X_of_type_Y(tokenC)
	}
	parser.move()

	return &ast.AlignerStageDelta{
		InputWindow:  &ast.DurationLiteral{Value: tokenA.Value},
		OutputPeriod: &ast.DurationLiteral{Value: tokenC.Value},
		Metadata:     getMultipleTokensMetadata(stage, tokenC),
	}, nil
}

func (parser *Parser) parseAlignerRate() (*ast.AlignerStageRate, *model.Error) {
	stage, _ := parser.currentToken()
	parser.move()

	tokenA, isEof := parser.currentToken()
	if isEndOfFilter(tokenA, isEof) {
		return &ast.AlignerStageRate{
			Metadata: getSingleTokenMetadata(stage),
		}, nil
	}
	if tokenA.Type != model.TypeDuration && tokenA.Type != model.TypeTemplateVariable {
		return nil, newParserError_unexpected_token_X_of_type_Y(tokenA)
	}
	parser.move()

	tokenB, isEof := parser.currentToken()
	if isEndOfFilter(tokenB, isEof) {
		return &ast.AlignerStageRate{
			InputWindow: &ast.DurationLiteral{Value: tokenA.Value},
			Metadata:    getMultipleTokensMetadata(stage, tokenA),
		}, nil
	}
	if tokenB.Type != model.TypeComma {
		return nil, newParserError_unexpected_token_X_of_type_Y(tokenB)
	}
	parser.move()

	tokenC, isEof := parser.currentToken()
	if isEndOfFilter(tokenC, isEof) {
		return nil, newParserError_unexpected_EOF(tokenB)
	}
	if tokenC.Type != model.TypeDuration && tokenC.Type != model.TypeTemplateVariable {
		return nil, newParserError_unexpected_token_X_of_type_Y(tokenC)
	}
	parser.move()

	return &ast.AlignerStageRate{
		InputWindow:  &ast.DurationLiteral{Value: tokenA.Value},
		OutputPeriod: &ast.DurationLiteral{Value: tokenC.Value},
		Metadata:     getMultipleTokensMetadata(stage, tokenC),
	}, nil
}

func (parser *Parser) parseAlignerLatest() (*ast.AlignerStageLatest, *model.Error) {
	stage, _ := parser.currentToken()
	parser.move()

	tokenA, isEof := parser.currentToken()
	if isEndOfFilter(tokenA, isEof) {
		return &ast.AlignerStageLatest{
			Metadata: getSingleTokenMetadata(stage),
		}, nil
	}
	if tokenA.Type != model.TypeDuration && tokenA.Type != model.TypeTemplateVariable {
		return nil, newParserError_unexpected_token_X_of_type_Y(tokenA)
	}
	parser.move()

	tokenB, isEof := parser.currentToken()
	if isEndOfFilter(tokenB, isEof) {
		return &ast.AlignerStageLatest{
			InputWindow: &ast.DurationLiteral{Value: tokenA.Value},
			Metadata:    getMultipleTokensMetadata(stage, tokenA),
		}, nil
	}
	if tokenB.Type != model.TypeComma {
		return nil, newParserError_unexpected_token_X_of_type_Y(tokenB)
	}
	parser.move()

	tokenC, isEof := parser.currentToken()
	if isEndOfFilter(tokenC, isEof) {
		return nil, newParserError_unexpected_EOF(tokenB)
	}
	if tokenC.Type != model.TypeDuration && tokenC.Type != model.TypeTemplateVariable {
		return nil, newParserError_unexpected_token_X_of_type_Y(tokenC)
	}
	parser.move()

	return &ast.AlignerStageLatest{
		InputWindow:  &ast.DurationLiteral{Value: tokenA.Value},
		OutputPeriod: &ast.DurationLiteral{Value: tokenC.Value},
		Metadata:     getMultipleTokensMetadata(stage, tokenC),
	}, nil
}

func (parser *Parser) parseAlignerReduce() (*ast.AlignerStageReduce, *model.Error) {
	stage, _ := parser.currentToken()
	parser.move()

	tokenA, isEof := parser.currentToken()
	if isEndOfFilter(tokenA, isEof) {
		return nil, newParserError_unexpected_EOF(stage)
	}
	if tokenA.Type != model.TypeDuration && tokenA.Type != model.TypeKeyword && tokenA.Type != model.TypeTemplateVariable {
		return nil, newParserError_unexpected_token_X_of_type_Y(tokenA)
	}
	parser.move()
	if tokenA.Type == model.TypeKeyword {
		return &ast.AlignerStageReduce{
			Reducer:  tokenA.Value,
			Metadata: getMultipleTokensMetadata(stage, tokenA),
		}, nil
	}

	tokenB, isEof := parser.currentToken()
	if isEndOfFilter(tokenB, isEof) {
		return nil, newParserError_unexpected_EOF(tokenA)
	}
	if tokenB.Type != model.TypeComma {
		return nil, newParserError_unexpected_token_X_of_type_Y(tokenB)
	}
	parser.move()

	tokenC, isEof := parser.currentToken()
	if isEndOfFilter(tokenC, isEof) {
		return nil, newParserError_unexpected_EOF(tokenB)
	}
	if tokenC.Type != model.TypeDuration && tokenC.Type != model.TypeKeyword && tokenC.Type != model.TypeTemplateVariable {
		return nil, newParserError_unexpected_token_X_of_type_Y(tokenC)
	}
	parser.move()
	if tokenC.Type == model.TypeKeyword {
		return &ast.AlignerStageReduce{
			InputWindow: &ast.DurationLiteral{Value: tokenA.Value},
			Reducer:     tokenC.Value,
			Metadata:    getMultipleTokensMetadata(stage, tokenC),
		}, nil
	}

	tokenD, isEof := parser.currentToken()
	if isEndOfFilter(tokenD, isEof) {
		return nil, newParserError_unexpected_EOF(tokenC)
	}
	if tokenD.Type != model.TypeComma {
		return nil, newParserError_unexpected_token_X_of_type_Y(tokenD)
	}
	parser.move()

	tokenE, isEof := parser.currentToken()
	if isEndOfFilter(tokenE, isEof) {
		return nil, newParserError_unexpected_EOF(tokenD)
	}
	if tokenE.Type != model.TypeKeyword {
		return nil, newParserError_unexpected_token_X_of_type_Y(tokenE)
	}
	parser.move()

	return &ast.AlignerStageReduce{
		InputWindow:  &ast.DurationLiteral{Value: tokenA.Value},
		OutputPeriod: &ast.DurationLiteral{Value: tokenC.Value},
		Reducer:      tokenE.Value,
		Metadata:     getMultipleTokensMetadata(stage, tokenE),
	}, nil
}

func (parser *Parser) parseAligner() (ast.AlignerStage, *model.Error) {
	token, _ := parser.currentToken()
	switch token.Value {
	case "delta":
		return parser.parseAlignerDelta()
	case "rate":
		return parser.parseAlignerRate()
	case "latest", "align":
		return parser.parseAlignerLatest()
	case "reduce":
		return parser.parseAlignerReduce()
	default:
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
}

func (parser *Parser) parseModifierFill() (*ast.ModifierStageFill, *model.Error) {
	stage, _ := parser.currentToken()
	parser.move()

	token, isEof := parser.currentToken()
	if isEndOfFilter(token, isEof) {
		return nil, newParserError_unexpected_EOF(stage)
	}
	var minusSign bool = false
	if token.Type == model.TypeDiff {
		minusSign = true
		parser.move()
		token, isEof = parser.currentToken()
		if isEndOfFilter(token, isEof) {
			return nil, newParserError_unexpected_EOF(stage)
		}
	}

	var numberLiteral ast.NumberLiteral
	switch token.Type {
	case model.TypeFloat:
		numberLiteral = &ast.FloatLiteral{
			Value:    minusOrEmpty(minusSign) + token.Value,
			Metadata: getSingleTokenMetadata(token),
		}
	case model.TypeInteger:
		numberLiteral = &ast.IntegerLiteral{
			Value:    minusOrEmpty(minusSign) + token.Value,
			Metadata: getSingleTokenMetadata(token),
		}
	default:
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	parser.move()

	return &ast.ModifierStageFill{
		Number:   numberLiteral,
		Metadata: getMultipleTokensMetadata(stage, token),
	}, nil
}

func (parser *Parser) parseModifierFilter() (*ast.ModifierStageFilter, *model.Error) {
	stage, _ := parser.currentToken()
	parser.move()

	expr, exprLastToken, err := parser.parseExpression(stage, LOWEST)
	if err != nil {
		return nil, err
	}

	return &ast.ModifierStageFilter{
		Expr:     expr,
		Metadata: getMultipleTokensMetadata(stage, exprLastToken),
	}, nil
}

func (parser *Parser) parseLabels() ([]ast.Identifier, *model.Error) {
	brToken, _ := parser.currentToken()

	if brToken.Type != model.TypeSquareBracketLeft {
		return nil, newParserError_unexpected_token_X_of_type_Y(brToken)
	}

	parser.move()
	var labels []ast.Identifier

	for range PARSER_MAX_ITERATIONS {
		token, isEof := parser.currentToken()
		if isEof {
			return nil, newParserError_unexpected_EOF(brToken)
		}
		if token.Type == model.TypeSquareBracketRight {
			parser.move()
			break
		}
		if token.Type == model.TypeComma {
			parser.move()
			continue
		}
		if token.Type != model.TypeIdentifier && token.Type != model.TypeString && token.Type != model.TypeKeyword && token.Type != model.TypeTemplateVariable {
			return nil, newParserError_unexpected_token_X_of_type_Y(token)
		}
		labels = append(labels, ast.Identifier{Value: token.Value, Metadata: getSingleTokenMetadata(token)})
		parser.move()
	}

	return labels, nil
}

func (parser *Parser) parseModifierGroupBy() (*ast.ModifierStageGroupBy, *model.Error) {
	stage, _ := parser.currentToken()
	parser.move()

	tokenA, isEof := parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(stage)
	}
	if tokenA.Type != model.TypeSquareBracketLeft {
		return nil, newParserError_unexpected_token_X_of_type_Y(tokenA)
	}

	labels, err := parser.parseLabels()
	if err != nil {
		return nil, err
	}

	tokenB, isEof := parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(stage)
	}
	if tokenB.Type != model.TypeComma {
		return nil, newParserError_unexpected_token_X_of_type_Y(tokenB)
	}
	parser.move()

	tokenC, isEof := parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(tokenB)
	}
	if tokenC.Type != model.TypeKeyword {
		return nil, newParserError_unexpected_token_X_of_type_Y(tokenC)
	}
	parser.move()

	return &ast.ModifierStageGroupBy{
		Labels:   labels,
		Reducer:  tokenC.Value,
		Metadata: getMultipleTokensMetadata(stage, tokenC),
	}, nil
}

func (parser *Parser) parseModifierTimeShift() (*ast.ModifierStageTimeShift, *model.Error) {
	stage, _ := parser.currentToken()
	parser.move()

	tokenA, isEof := parser.currentToken()
	if isEndOfFilter(tokenA, isEof) {
		return nil, newParserError_unexpected_EOF(stage)
	}
	if tokenA.Type != model.TypeDuration && tokenA.Type != model.TypeTemplateVariable {
		return nil, newParserError_unexpected_token_X_of_type_Y(tokenA)
	}
	parser.move()

	return &ast.ModifierStageTimeShift{
		ShiftDuration: ast.DurationLiteral{Value: tokenA.Value, Metadata: getSingleTokenMetadata(tokenA)},
		Metadata:      getMultipleTokensMetadata(stage, tokenA),
	}, nil
}

func (parser *Parser) parseModifierPoint() (*ast.ModifierStagePoint, *model.Error) {
	stage, _ := parser.currentToken()
	parser.move()

	expr, lastToken, err := parser.parseExpression(stage, LOWEST)
	if err != nil {
		return nil, err
	}

	var expressions []ast.Expression = []ast.Expression{expr}

	for range PARSER_MAX_ITERATIONS {
		token, isEof := parser.currentToken()
		if isEndOfFilter(token, isEof) {
			break
		}
		if token.Type != model.TypeComma {
			break
		}
		parser.move()

		expr, lastToken, err = parser.parseExpression(stage, LOWEST)
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, expr)
	}

	return &ast.ModifierStagePoint{
		Expressions: expressions,
		Metadata:    getMultipleTokensMetadata(stage, lastToken),
	}, nil
}

func (parser *Parser) parseModifierPointFilter() (*ast.ModifierStagePointFilter, *model.Error) {
	stage, _ := parser.currentToken()
	parser.move()

	expr, lastToken, err := parser.parseExpression(stage, LOWEST)
	if err != nil {
		return nil, err
	}

	return &ast.ModifierStagePointFilter{
		Expr:     expr,
		Metadata: getMultipleTokensMetadata(stage, lastToken),
	}, nil
}

func (parser *Parser) parseModifierTop() (*ast.ModifierStageTop, *model.Error) {
	stage, _ := parser.currentToken()
	parser.move()

	token, isEof := parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(stage)
	}

	var err *model.Error = nil
	var labels []ast.Identifier

	if token.Type == model.TypeSquareBracketLeft {
		labels, err = parser.parseLabels()
		if err != nil {
			return nil, err
		}
		token, isEof := parser.currentToken()
		if isEof {
			return nil, newParserError_unexpected_EOF(stage)
		}
		if token.Type != model.TypeComma {
			return nil, newParserError_unexpected_token_X_of_type_Y(token)
		}
		parser.move()
	}

	token, isEof = parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(stage)
	}
	if token.Type != model.TypeInteger {
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	amount := ast.IntegerLiteral{Value: token.Value, Metadata: getSingleTokenMetadata(token)}
	parser.move()

	token, isEof = parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(stage)
	}
	if token.Type != model.TypeComma {
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	parser.move()

	token, isEof = parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(stage)
	}
	if token.Type != model.TypeKeyword {
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	reducer := token.Value
	reducerToken := token
	parser.move()

	token, isEof = parser.currentToken()
	if isEndOfFilter(token, isEof) {
		return &ast.ModifierStageTop{
			Labels:   labels,
			Amount:   amount,
			Reducer:  reducer,
			Window:   nil,
			Metadata: getMultipleTokensMetadata(stage, reducerToken),
		}, nil
	}
	if token.Type != model.TypeComma {
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	parser.move()

	token, isEof = parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(stage)
	}
	if token.Type != model.TypeDuration {
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	window := ast.DurationLiteral{Value: token.Value, Metadata: getSingleTokenMetadata(token)}
	parser.move()

	return &ast.ModifierStageTop{
		Labels:   labels,
		Amount:   amount,
		Reducer:  reducer,
		Window:   &window,
		Metadata: getMultipleTokensMetadata(stage, token),
	}, nil
}

func (parser *Parser) parseModifierBottom() (*ast.ModifierStageBottom, *model.Error) {
	stage, _ := parser.currentToken()
	parser.move()

	token, isEof := parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(stage)
	}

	var err *model.Error = nil
	var labels []ast.Identifier

	if token.Type == model.TypeSquareBracketLeft {
		labels, err = parser.parseLabels()
		if err != nil {
			return nil, err
		}
		token, isEof := parser.currentToken()
		if isEof {
			return nil, newParserError_unexpected_EOF(stage)
		}
		if token.Type != model.TypeComma {
			return nil, newParserError_unexpected_token_X_of_type_Y(token)
		}
		parser.move()
	}

	token, isEof = parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(stage)
	}
	if token.Type != model.TypeInteger {
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	amount := ast.IntegerLiteral{Value: token.Value, Metadata: getSingleTokenMetadata(token)}
	parser.move()

	token, isEof = parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(stage)
	}
	if token.Type != model.TypeComma {
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	parser.move()

	token, isEof = parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(stage)
	}
	if token.Type != model.TypeKeyword {
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	reducer := token.Value
	reducerToken := token
	parser.move()

	token, isEof = parser.currentToken()
	if isEndOfFilter(token, isEof) {
		return &ast.ModifierStageBottom{
			Labels:   labels,
			Amount:   amount,
			Reducer:  reducer,
			Window:   nil,
			Metadata: getMultipleTokensMetadata(stage, reducerToken),
		}, nil
	}
	if token.Type != model.TypeComma {
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	parser.move()

	token, isEof = parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(stage)
	}
	if token.Type != model.TypeDuration {
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	window := ast.DurationLiteral{Value: token.Value, Metadata: getSingleTokenMetadata(token)}
	parser.move()

	return &ast.ModifierStageBottom{
		Labels:   labels,
		Amount:   amount,
		Reducer:  reducer,
		Window:   &window,
		Metadata: getMultipleTokensMetadata(stage, token),
	}, nil
}

func (parser *Parser) parseModifierJoin() (*ast.ModifierStageJoin, *model.Error) {
	stage, _ := parser.currentToken()
	parser.move()

	expr, lastToken, err := parser.parseExpression(stage, LOWEST)
	if err != nil {
		return nil, err
	}

	var leftDefault, rightDefault ast.NumberLiteral = nil, nil

	token, isEof := parser.currentToken()
	if isEndOfFilter(token, isEof) {
		return &ast.ModifierStageJoin{
			Expr:         expr,
			LeftDefault:  nil,
			RightDefault: nil,
			Metadata:     getMultipleTokensMetadata(stage, lastToken),
		}, nil
	}
	if token.Type != model.TypeComma {
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	parser.move()
	token, isEof = parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(stage)
	}
	switch token.Type {
	case model.TypeFloat:
		leftDefault = &ast.FloatLiteral{Value: token.Value, Metadata: getSingleTokenMetadata(token)}
		lastToken = token
		parser.move()
	case model.TypeInteger:
		leftDefault = &ast.IntegerLiteral{Value: token.Value, Metadata: getSingleTokenMetadata(token)}
		lastToken = token
		parser.move()
	default:
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}

	token, isEof = parser.currentToken()
	if isEndOfFilter(token, isEof) {
		return &ast.ModifierStageJoin{
			Expr:         expr,
			LeftDefault:  leftDefault,
			RightDefault: nil,
			Metadata:     getMultipleTokensMetadata(stage, lastToken),
		}, nil
	}
	if token.Type != model.TypeComma {
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	parser.move()
	token, isEof = parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(stage)
	}
	switch token.Type {
	case model.TypeFloat:
		rightDefault = &ast.FloatLiteral{Value: token.Value, Metadata: getSingleTokenMetadata(token)}
		lastToken = token
		parser.move()
	case model.TypeInteger:
		rightDefault = &ast.IntegerLiteral{Value: token.Value, Metadata: getSingleTokenMetadata(token)}
		lastToken = token
		parser.move()
	default:
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}

	return &ast.ModifierStageJoin{
		Expr:         expr,
		LeftDefault:  leftDefault,
		RightDefault: rightDefault,
		Metadata:     getMultipleTokensMetadata(stage, lastToken),
	}, nil
}

func (parser *Parser) parseModifier() (ast.ModifierStage, *model.Error) {
	token, _ := parser.currentToken()
	switch token.Value {
	case "fill":
		return parser.parseModifierFill()
	case "filter":
		return parser.parseModifierFilter()
	case "group_by":
		return parser.parseModifierGroupBy()
	case "join":
		return parser.parseModifierJoin()
	case "top":
		return parser.parseModifierTop()
	case "bottom":
		return parser.parseModifierBottom()
	case "point":
		return parser.parseModifierPoint()
	case "point_filter":
		return parser.parseModifierPointFilter()
	case "time_shift":
		return parser.parseModifierTimeShift()
	default:
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
}

func (parser *Parser) parseSingleArgumentFunction(parentToken *model.Token) (ast.Expression, *model.Token, *model.Error) {
	operation, _ := parser.currentToken()
	parser.move()

	token, isEof := parser.currentToken()
	if isEof {
		return nil, nil, newParserError_unexpected_EOF(parentToken)
	}
	if token.Type != model.TypeRoundBracketLeft {
		return nil, nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	parser.move()

	expr, _, err := parser.parseExpression(parentToken, LOWEST)
	if err != nil {
		return nil, nil, err
	}

	token, isEof = parser.currentToken()
	if isEof {
		return nil, nil, newParserError_unexpected_EOF(parentToken)
	}
	if token.Type != model.TypeRoundBracketRight {
		return nil, nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	parser.move()

	return &ast.PrefixExpression{
		Operation: string(operation.Value),
		Expr:      expr,
		Metadata:  getMultipleTokensMetadata(operation, token),
	}, token, nil
}

func (parser *Parser) parseDoubleArgumentFunction(parentToken *model.Token) (ast.Expression, *model.Token, *model.Error) {
	operation, _ := parser.currentToken()
	parser.move()

	token, isEof := parser.currentToken()
	if isEof {
		return nil, nil, newParserError_unexpected_EOF(parentToken)
	}
	if token.Type != model.TypeRoundBracketLeft {
		return nil, nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	parser.move()

	leftExpr, _, err := parser.parseExpression(parentToken, LOWEST)
	if err != nil {
		return nil, nil, err
	}

	token, isEof = parser.currentToken()
	if isEof {
		return nil, nil, newParserError_unexpected_EOF(parentToken)
	}
	if token.Type != model.TypeComma {
		return nil, nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	parser.move()

	rightExpr, _, err := parser.parseExpression(parentToken, LOWEST)
	if err != nil {
		return nil, nil, err
	}

	token, isEof = parser.currentToken()
	if isEof {
		return nil, nil, newParserError_unexpected_EOF(parentToken)
	}
	if token.Type != model.TypeRoundBracketRight {
		return nil, nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	parser.move()

	return &ast.InfixExpression{
		Operation: string(operation.Value),
		LeftExpr:  leftExpr,
		RightExpr: rightExpr,
		Metadata:  getMultipleTokensMetadata(operation, token),
	}, token, nil
}

func (parser *Parser) parsePrefixExpression(parentToken *model.Token) (ast.Expression, *model.Token, *model.Error) {
	token, _ := parser.currentToken()
	switch token.Type {
	case model.TypeDiff:
		parser.move()
		expr, exprLastToken, err := parser.parseExpression(token, PREFIX)
		if err != nil {
			return nil, nil, err
		}
		switch typedExpr := expr.(type) {
		case *ast.IntegerLiteral:
			return &ast.IntegerLiteral{
				Value:    "-" + typedExpr.Value,
				Metadata: getMultipleTokensMetadata(token, exprLastToken),
			}, exprLastToken, nil
		case *ast.FloatLiteral:
			return &ast.FloatLiteral{
				Value:    "-" + typedExpr.Value,
				Metadata: getMultipleTokensMetadata(token, exprLastToken),
			}, exprLastToken, nil
		default:
			return &ast.PrefixExpression{
				Operation: string(token.Type),
				Expr:      expr,
				Metadata:  getMultipleTokensMetadata(token, exprLastToken),
			}, exprLastToken, nil
		}
	case model.TypeLogicalNot:
		parser.move()
		expr, exprLastToken, err := parser.parseExpression(token, PREFIX)
		if err != nil {
			return nil, nil, err
		}
		return &ast.PrefixExpression{
			Operation: string(token.Type),
			Expr:      expr,
			Metadata:  getMultipleTokensMetadata(token, exprLastToken),
		}, exprLastToken, nil
	case model.TypeRoundBracketLeft:
		parser.move()
		expr, exprLastToken, err := parser.parseExpression(token, LOWEST)
		if err != nil {
			return nil, nil, err
		}
		token, isEof := parser.currentToken()
		if isEof {
			return nil, nil, newParserError_unexpected_EOF(parentToken)
		}
		if token.Type != model.TypeRoundBracketRight {
			return nil, nil, newParserError_unexpected_token_X_of_type_Y(token)
		}
		parser.move()
		return expr, exprLastToken, nil
	case model.TypeKeyword:
		if isSingleArgumentFunction(token.Value) {
			expr, exprLastToken, err := parser.parseSingleArgumentFunction(token)
			if err != nil {
				return nil, nil, err
			}
			return expr, exprLastToken, nil
		}
		if isDoubleArgumentFunction(token.Value) {
			expr, exprLastToken, err := parser.parseDoubleArgumentFunction(token)
			if err != nil {
				return nil, nil, err
			}
			return expr, exprLastToken, nil
		}
		parser.move()
		return &ast.Identifier{
			Value:    token.Value,
			Metadata: getSingleTokenMetadata(token),
		}, token, nil
	case model.TypeIdentifier:
		parser.move()
		return &ast.Identifier{Value: token.Value, Metadata: getSingleTokenMetadata(token)}, token, nil
	case model.TypeTemplateVariable:
		parser.move()
		return &ast.TemplateVariable{Value: token.Value, Metadata: getSingleTokenMetadata(token)}, token, nil
	case model.TypeString:
		parser.move()
		return &ast.StringLiteral{Value: token.Value, Metadata: getSingleTokenMetadata(token)}, token, nil
	case model.TypeInteger:
		parser.move()
		return &ast.IntegerLiteral{Value: token.Value, Metadata: getSingleTokenMetadata(token)}, token, nil
	case model.TypeFloat:
		parser.move()
		return &ast.FloatLiteral{Value: token.Value, Metadata: getSingleTokenMetadata(token)}, token, nil
	case model.TypeBoolean:
		parser.move()
		return &ast.BooleanLiteral{Value: token.Value, Metadata: getSingleTokenMetadata(token)}, token, nil
	case model.TypeDuration:
		parser.move()
		return &ast.DurationLiteral{Value: token.Value, Metadata: getSingleTokenMetadata(token)}, token, nil
	default:
		return nil, nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
}

func (parser *Parser) parseInfixExpression(
	parentToken *model.Token,
	leftExpr ast.Expression,
	leftExprFirstToken *model.Token,
) (ast.Expression, *model.Token, *model.Error) {
	token, isEof := parser.currentToken()
	if isEof {
		return nil, nil, newParserError_unexpected_EOF(parentToken)
	}
	switch {
	case isInfixOperation(token.Type):
		priority, _ := mapInfixOperationToPriority(token.Type)
		parser.move()
		rightExpr, rightExprLastToken, err := parser.parseExpression(token, priority)
		if err != nil {
			return nil, nil, err
		}
		fixedOperation := tryFixOperationType(token.Value)
		return &ast.InfixExpression{
			Operation: fixedOperation,
			LeftExpr:  leftExpr,
			RightExpr: rightExpr,
			Metadata:  getMultipleTokensMetadata(leftExprFirstToken, rightExprLastToken),
		}, rightExprLastToken, nil
	default:
		return nil, nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
}

func (parser *Parser) parseExpression(parentToken *model.Token, priority OperationPriority) (ast.Expression, *model.Token, *model.Error) {
	startToken, isEof := parser.currentToken()
	if isEof {
		return nil, nil, newParserError_unexpected_EOF(parentToken)
	}

	prefixExpr, lastToken, err := parser.parsePrefixExpression(parentToken)
	if err != nil {
		return nil, nil, err
	}

	for range PARSER_MAX_ITERATIONS {
		currentToken, isEof := parser.currentToken()
		if isEndOfFilter(currentToken, isEof) {
			return prefixExpr, lastToken, nil
		}
		if currentToken.Type == model.TypeComma || currentToken.Type == model.TypeRoundBracketRight {
			return prefixExpr, lastToken, nil
		}
		currentPriority, hasPriority := mapInfixOperationToPriority(currentToken.Type)
		if !hasPriority {
			return prefixExpr, lastToken, nil
		}
		if priority >= currentPriority {
			return prefixExpr, lastToken, nil
		}
		prefixExpr, lastToken, err = parser.parseInfixExpression(currentToken, prefixExpr, startToken)
		if err != nil {
			return nil, nil, err
		}
	}

	return nil, nil, newParserError_max_iterations_limit_reached(lastToken)
}

func (parser *Parser) parsePipeline() ([]ast.Stage, *model.Error) {
	var stages []ast.Stage
	var token *model.Token
	var isEof bool
	for range PARSER_MAX_ITERATIONS {
		token, isEof = parser.currentToken()
		if isEof {
			return stages, nil
		}
		switch {
		case token.Type == model.TypeSemicolon || token.Type == model.TypeRoundBracketRight:
			return stages, nil
		case token.Type == model.TypeSeparator:
			parser.move()
		case token.Type == model.TypeKeyword && isFetchStage(token.Value):
			stage, err := parser.parseFetchStage()
			if err != nil {
				return nil, err
			}
			stages = append(stages, stage)
		case token.Type == model.TypeKeyword && isAligner(token.Value):
			stage, err := parser.parseAligner()
			if err != nil {
				return nil, err
			}
			stages = append(stages, stage)
		case token.Type == model.TypeKeyword && isModifier(token.Value):
			stage, err := parser.parseModifier()
			if err != nil {
				return nil, err
			}
			stages = append(stages, stage)
		default:
			return nil, newParserError_unexpected_token_X_of_type_Y(token)
		}
	}
	return nil, newParserError_max_iterations_limit_reached(token)
}

func (parser *Parser) parseUnnamedJoin() (*ast.Query, *model.Error) {
	unnamedJoin, _ := parser.currentToken()
	parser.move()

	queryLeft, err := parser.parseQuery()
	if err != nil {
		return nil, err
	}

	token, isEof := parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(unnamedJoin)
	}
	if token.Type != model.TypeSemicolon {
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	parser.move()

	queryRight, err := parser.parseQuery()
	if err != nil {
		return nil, err
	}

	token, isEof = parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(unnamedJoin)
	}
	switch token.Type {
	case model.TypeRoundBracketRight:
		parser.move()
	case model.TypeSemicolon:
		parser.move()
		token, isEof = parser.currentToken()
		if isEof {
			return nil, newParserError_unexpected_EOF(unnamedJoin)
		}
		if token.Type != model.TypeRoundBracketRight {
			return nil, newParserError_unexpected_token_X_of_type_Y(token)
		}
		parser.move()
	default:
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}

	remainingStages, err := parser.parsePipeline()
	if err != nil {
		return nil, err
	}

	if len(remainingStages) == 0 {
		return nil, model.NewErrorForMultipleTokens("join stage is required for unnamed joins", unnamedJoin, token)
	}

	lastMetadata := remainingStages[len(remainingStages)-1].GetMetadata()

	return &ast.Query{
		Type: ast.QueryTypeUnnamedJoin,
		UnnamedJoin: &ast.UnnamedJoin{
			Left:   queryLeft,
			Right:  queryRight,
			Stages: remainingStages,
		},
		Metadata: ast.Metadata{
			SourceIndex:  unnamedJoin.Index,
			SourceLength: lastMetadata.SourceIndex - unnamedJoin.Index + lastMetadata.SourceLength,
		},
	}, nil
}

func (parser *Parser) parseNamedJoin() (*ast.Query, *model.Error) {
	namedJoin, _ := parser.currentToken()
	parser.move()

	queries := make(map[string]*ast.Query)

	for range PARSER_MAX_ITERATIONS {
		tokenA, isEof := parser.currentToken()
		if isEof {
			return nil, newParserError_unexpected_EOF(namedJoin)
		}
		if tokenA.Type != model.TypeIdentifier && tokenA.Type != model.TypeKeyword {
			break
		}
		if isJoinKeyword(tokenA) {
			break
		}
		parser.move()

		tokenB, isEof := parser.currentToken()
		if isEof {
			return nil, newParserError_unexpected_EOF(namedJoin)
		}
		if tokenB.Type != model.TypeAssign {
			return nil, newParserError_unexpected_token_X_of_type_Y(tokenB)
		}
		parser.move()

		innerQuery, err := parser.parseQuery()
		if err != nil {
			return nil, err
		}

		tokenC, isEof := parser.currentToken()
		if isEof {
			return nil, newParserError_unexpected_EOF(namedJoin)
		}
		if tokenC.Type != model.TypeSemicolon {
			return nil, newParserError_unexpected_token_X_of_type_Y(tokenC)
		}
		parser.move()

		queries[tokenA.Value] = innerQuery
	}

	token, isEof := parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(namedJoin)
	}
	if token.Type != model.TypeKeyword && token.Value != "join" {
		return nil, newParserError_unexpected_token_X_of_type_Y(token)
	}
	parser.move()

	joinExpr, lastToken, err := parser.parseExpression(token, LOWEST)
	if err != nil {
		return nil, err
	}

	defaults := make(map[string]ast.NumberLiteral)

	for range PARSER_MAX_ITERATIONS {
		tokenA, isEof := parser.currentToken()
		if isEof {
			break
		}
		if tokenA.Type == model.TypeSeparator || tokenA.Type == model.TypeSemicolon {
			break
		}
		if tokenA.Type != model.TypeComma {
			return nil, newParserError_unexpected_token_X_of_type_Y(tokenA)
		}
		parser.move()

		tokenB, isEof := parser.currentToken()
		if isEof {
			return nil, newParserError_unexpected_EOF(tokenA)
		}
		if tokenB.Type != model.TypeIdentifier && tokenB.Type != model.TypeKeyword {
			return nil, newParserError_unexpected_token_X_of_type_Y(tokenB)
		}
		parser.move()

		tokenC, isEof := parser.currentToken()
		if isEof {
			return nil, newParserError_unexpected_EOF(tokenB)
		}
		if tokenC.Type != model.TypeAssign {
			return nil, newParserError_unexpected_token_X_of_type_Y(tokenC)
		}
		parser.move()

		tokenD, isEof := parser.currentToken()
		if isEof {
			return nil, newParserError_unexpected_EOF(tokenC)
		}
		if tokenD.Type != model.TypeFloat && tokenD.Type != model.TypeInteger {
			return nil, newParserError_unexpected_token_X_of_type_Y(tokenD)
		}
		lastToken = tokenD
		parser.move()

		if tokenD.Type == model.TypeFloat {
			defaults[tokenB.Value] = &ast.FloatLiteral{Value: tokenD.Value, Metadata: getSingleTokenMetadata(tokenD)}
		} else {
			defaults[tokenB.Value] = &ast.IntegerLiteral{Value: tokenD.Value, Metadata: getSingleTokenMetadata(tokenD)}
		}
	}

	var remainingStages []ast.Stage
	token, isEof = parser.currentToken()

	if !isEof && token.Type == model.TypeSeparator {
		parser.move()
		remainingStages, err = parser.parsePipeline()
		if err != nil {
			return nil, err
		}
	}

	resultQueries := make([]ast.NamedJoinPipeline, 0, len(queries))

	for name, query := range queries {
		var defaultValue ast.NumberLiteral = nil

		if _, exists := defaults[name]; exists {
			defaultValue = defaults[name]
		}

		resultQueries = append(resultQueries, ast.NamedJoinPipeline{
			Name:    name,
			Query:   query,
			Default: defaultValue,
		})
	}

	sort.Slice(resultQueries, func(i, j int) bool {
		return resultQueries[i].Name < resultQueries[j].Name
	})

	metadata := getMultipleTokensMetadata(namedJoin, lastToken)

	if len(remainingStages) > 0 {
		lastMetadata := remainingStages[len(remainingStages)-1].GetMetadata()
		metadata = ast.Metadata{
			SourceIndex:  namedJoin.Index,
			SourceLength: lastMetadata.SourceIndex - namedJoin.Index + lastMetadata.SourceLength,
		}
	}

	return &ast.Query{
		Type: ast.QueryTypeNamedJoin,
		NamedJoin: &ast.NamedJoin{
			Queries:  resultQueries,
			JoinExpr: joinExpr,
			Stages:   remainingStages,
		},
		Metadata: metadata,
	}, nil
}

func (parser *Parser) parseDefaultQuery() (*ast.Query, *model.Error) {
	pipeline, err := parser.parsePipeline()
	if err != nil {
		return nil, err
	}
	firstMetadata := pipeline[0].GetMetadata()
	lastMetadata := pipeline[len(pipeline)-1].GetMetadata()
	return &ast.Query{
		Type:     ast.QueryTypeDefault,
		Pipeline: pipeline,
		Metadata: ast.Metadata{
			SourceIndex:  firstMetadata.SourceIndex,
			SourceLength: lastMetadata.SourceIndex - firstMetadata.SourceIndex + lastMetadata.SourceLength,
		},
	}, nil
}

func (parser *Parser) parseQuery() (*ast.Query, *model.Error) {
	token, isEof := parser.currentToken()
	if isEof {
		return nil, newParserError_unexpected_EOF(&model.Token{Index: parser.index, Length: 1})
	}
	switch {
	case token.Type == model.TypeKeyword && isFetchStage(token.Value):
		query, err := parser.parseDefaultQuery()
		if err != nil {
			return nil, err
		}
		return query, nil
	case token.Type == model.TypeKeyword && token.Value == "with":
		query, err := parser.parseNamedJoin()
		if err != nil {
			return nil, err
		}
		return query, nil
	case token.Type == model.TypeRoundBracketLeft:
		query, err := parser.parseUnnamedJoin()
		if err != nil {
			return nil, err
		}
		return query, nil
	}
	return nil, newParserError_unexpected_token_X_of_type_Y(token)
}

func (parser *Parser) Parse() (*ast.Query, *model.Error) {
	query, err := parser.parseQuery()
	if err != nil {
		return nil, err
	}
	return query, nil
}
