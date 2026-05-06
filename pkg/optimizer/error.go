package optimizer

import (
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model/ast"
)

func newOptimizeError(status string, metadata ast.Metadata) *model.Error {
	return model.NewError(status, metadata.SourceIndex, metadata.SourceLength)
}

func newOptimizeError_unexpected_operand_in_expression(metadata ast.Metadata) *model.Error {
	return newOptimizeError("unexpected operand in expression", metadata)
}

func newOptimizeError_expression_is_too_long(metadata ast.Metadata) *model.Error {
	return newOptimizeError("expression is too complex", metadata)
}
