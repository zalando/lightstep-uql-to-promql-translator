package streams

import (
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
)

const (
	PARSER_MAX_ITERATIONS = 1024
)

type Parser struct {
	tokens []model.Token
	index  int
}

func NewParser(tokens []model.Token) *Parser {
	return &Parser{
		tokens: tokens,
		index:  0,
	}
}

func (p *Parser) currentToken() (*model.Token, bool) {
	if p.index >= len(p.tokens) {
		return nil, true
	}
	return &p.tokens[p.index], false
}

func (p *Parser) move() {
	p.index++
}

func (p *Parser) parseArray() ([]string, *model.Error) {
	token, isEof := p.currentToken()
	if isEof {
		return nil, model.NewError("unexpected EOF", 0, 0)
	}
	if token.Type != model.TypeRoundBracketLeft {
		return nil, model.NewErrorForSingleToken("unexpected token", token)
	}
	p.move()

	var result []string = nil

	for range PARSER_MAX_ITERATIONS {
		token, isEof = p.currentToken()
		if isEof {
			return nil, model.NewError("unexpected EOF", 0, 0)
		}
		if token.Type == model.TypeRoundBracketRight {
			p.move()
			return result, nil
		}
		if token.Type != model.TypeString {
			return nil, model.NewErrorForSingleToken("unexpected token", token)
		}
		result = append(result, token.Value)
		p.move()
	}

	return nil, model.NewErrorForSingleToken("too many iterations while parsing query", token)
}

func (p *Parser) parseAnd() *model.Error {
	token, _ := p.currentToken()
	if token.Type != model.TypeString {
		return model.NewErrorForSingleToken("unexpected token", token)
	}
	if token.Value != "AND" {
		return model.NewErrorForSingleToken("unexpected token", token)
	}
	p.move()
	return nil
}

func (p *Parser) parseOperation() (string, *model.Error) {
	tokenA, isEof := p.currentToken()
	if isEof {
		return "", model.NewError("unexpected EOF", 0, 0)
	}
	if tokenA.Value != "IN" && tokenA.Value != "MATCHES" && tokenA.Value != "NOT" {
		return "", model.NewErrorForSingleToken("unexpected token", tokenA)
	}
	p.move()

	if tokenA.Value == "IN" {
		return "in", nil
	}

	tokenB, isEof := p.currentToken()
	if isEof {
		return "", model.NewError("unexpected EOF", 0, 0)
	}
	if tokenB.Value != "IN" && tokenB.Value != "REGEXP" && tokenB.Value != "MATCHES" {
		return "", model.NewErrorForSingleToken("unexpected token", tokenB)
	}
	p.move()

	if tokenB.Value == "IN" {
		return "not_in", nil
	}
	if tokenB.Value == "REGEXP" {
		return "matches_regexp", nil
	}

	tokenC, isEof := p.currentToken()
	if isEof {
		return "", model.NewError("unexpected EOF", 0, 0)
	}
	if tokenC.Value != "REGEXP" {
		return "", model.NewErrorForSingleToken("unexpected token", tokenC)
	}
	p.move()

	return "not_matches_regexp", nil
}

func (p *Parser) Parse() ([]Filter, *model.Error) {
	var result []Filter = nil
	var firstIteration bool = true

	for range PARSER_MAX_ITERATIONS {
		_, isEof := p.currentToken()
		if isEof {
			return result, nil
		}

		if !firstIteration {
			err := p.parseAnd()
			if err != nil {
				return nil, err
			}
		}

		keyToken, isEof := p.currentToken()
		if isEof {
			return result, nil
		}
		if keyToken.Type != model.TypeString {
			return nil, model.NewErrorForSingleToken("unexpected token", keyToken)
		}
		p.move()

		operation, err := p.parseOperation()
		if err != nil {
			return nil, err
		}

		values, err := p.parseArray()
		if err != nil {
			return nil, err
		}

		result = append(result, Filter{
			Key:      keyToken.Value,
			Operator: operation,
			Values:   values,
		})

		firstIteration = false
	}

	return nil, model.NewError("too many iterations while parsing query", 0, 0)
}
