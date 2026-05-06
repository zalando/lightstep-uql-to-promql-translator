package lexer

import (
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
)

func Tokenize(query string) ([]model.Token, *model.Error) {
	lexerInstance := New(query)
	tokens, err := lexerInstance.FetchAllTokensWithoutComments()
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

func TokenizeWithComments(query string) ([]model.Token, *model.Error) {
	lexerInstance := New(query)
	tokens, err := lexerInstance.FetchAllTokens()
	if err != nil {
		return nil, err
	}
	return tokens, nil
}
