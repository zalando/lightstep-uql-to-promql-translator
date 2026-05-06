package parser

import (
	"fmt"

	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
)

func newParserError_unexpected_EOF(token *model.Token) *model.Error {
	return model.NewErrorForSingleToken("unexpected EOF", token)
}

func newParserError_unexpected_token_X_of_type_Y(token *model.Token) *model.Error {
	return model.NewErrorForSingleToken(fmt.Sprintf("unexpected token %s of type %s", token.Value, token.Type), token)
}

func newParserError_max_iterations_limit_reached(token *model.Token) *model.Error {
	return model.NewErrorForSingleToken("max iterations limit reached while parsing query", token)
}
