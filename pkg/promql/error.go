package promql

import (
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model/ast"
)

func newTranslatorError(status string, metadata ast.Metadata) *model.Error {
	return model.NewError(status, metadata.SourceIndex, metadata.SourceLength)
}

func newTranslatorError_unexpected_operand_in_expression(metadata ast.Metadata) *model.Error {
	return newTranslatorError("unexpacted operand in expression", metadata)
}

func newTranslatorError_unexpected_operand_in_disjunct(metadata ast.Metadata) *model.Error {
	return newTranslatorError("unexpected operand in disjunct", metadata)
}

func newTranslatorError_unexpected_operand_in_dnf(metadata ast.Metadata) *model.Error {
	return newTranslatorError("unexpected operand in disjunctive normal form", metadata)
}

func newTranslatorError_expression_is_too_long(metadata ast.Metadata) *model.Error {
	return newTranslatorError("expression is too complex", metadata)
}

func newTranslatorError_invalid_expression(metadata ast.Metadata) *model.Error {
	return newTranslatorError("invalid expression", metadata)
}
