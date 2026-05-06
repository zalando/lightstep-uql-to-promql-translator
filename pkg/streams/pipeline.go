package streams

import (
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
)

func Parse(streamQuery string) ([]Filter, *model.Error) {
	lexer := NewLexer(streamQuery)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, err
	}

	parser := NewParser(tokens)
	filters, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	return filters, nil
}
