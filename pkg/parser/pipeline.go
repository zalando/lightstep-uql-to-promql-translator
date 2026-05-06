package parser

import (
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/lexer"
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model/ast"
)

func Parse(query string) (*ast.Query, *model.Error) {
	tokens, err := lexer.Tokenize(query)
	if err != nil {
		return nil, err
	}

	parserInstance := New(tokens)
	queryAst, err := parserInstance.Parse()
	if err != nil {
		return nil, err
	}

	return queryAst, nil
}
