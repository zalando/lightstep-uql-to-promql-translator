package promql

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model/ast"
)

func getProperDefaultInterval(useRateInterval bool) string {
	if useRateInterval {
		return "$__rate_interval"
	}
	return "$__interval"
}

func isLetter(char byte) bool {
	return isLetterLowercase(char) || isLetterUppercase(char)
}

func isLetterLowercase(char byte) bool {
	return 'a' <= char && char <= 'z'
}

func isLetterUppercase(char byte) bool {
	return 'A' <= char && char <= 'Z'
}

func isDigit(char byte) bool {
	return '0' <= char && char <= '9'
}

func isUnderscore(char byte) bool {
	return char == '_'
}

func isTemplateVariableChar(char byte) bool {
	return char == '$' || char == '{' || char == '}'
}

func promqlAttributeFormat(raw string) string {
	var result strings.Builder
	for _, c := range []byte(raw) {
		switch {
		case isLetter(c) || isDigit(c) || isUnderscore(c) || isTemplateVariableChar(c):
			result.WriteByte(c)
		default:
			result.WriteByte('_')
		}
	}
	return result.String()
}

func tryGetAttributeKey(expr ast.Expression) (string, *model.Error) {
	switch typedExpr := expr.(type) {
	case *ast.TemplateVariable:
		return typedExpr.Value, nil
	case *ast.Identifier:
		return promqlAttributeFormat(typedExpr.Value), nil
	case *ast.StringLiteral:
		return promqlAttributeFormat(typedExpr.Value), nil
	default:
		return "", newTranslatorError_unexpected_operand_in_disjunct(expr.GetMetadata())
	}
}

func tryGetAttributeValue(expr ast.Expression) (string, *model.Error) {
	switch typedExpr := expr.(type) {
	case *ast.TemplateVariable:
		return "$" + typedExpr.Value, nil
	case *ast.StringLiteral:
		result, err := json.Marshal(typedExpr.Value)
		if err != nil {
			return "", newTranslatorError(fmt.Sprintf("cannot marshal string literal: %s", typedExpr.Value), expr.GetMetadata())
		}
		return string(result), nil
	case *ast.Identifier:
		result, err := json.Marshal(typedExpr.Value)
		if err != nil {
			return "", newTranslatorError(fmt.Sprintf("cannot marshal identifier literal: %s", typedExpr.Value), expr.GetMetadata())
		}
		return string(result), nil
	case *ast.IntegerLiteral:
		return "\"" + typedExpr.Value + "\"", nil
	case *ast.FloatLiteral:
		return "\"" + typedExpr.Value + "\"", nil
	case *ast.BooleanLiteral:
		return "\"" + typedExpr.Value + "\"", nil
	default:
		return "", newTranslatorError_unexpected_operand_in_disjunct(expr.GetMetadata())
	}
}

func mapOperationToPromQL(uqlOperation string) (string, bool) {
	switch uqlOperation {
	case string(model.TypeEquals):
		return "=", true
	case string(model.TypeAssign):
		return "=", true
	case string(model.TypeNotEquals):
		return "!=", true
	case string(model.TypeMatchRegex):
		return "=~", true
	case string(model.TypeNotMachRegex):
		return "!~", true
	case string(model.TypeLess):
		return "<", true
	case string(model.TypeLessOrEquals):
		return "<=", true
	case string(model.TypeMore):
		return ">", true
	case string(model.TypeMoreOrEquals):
		return ">=", true
	default:
		return "", false
	}
}

func mapReducerToReducerFuncName(reducer string) (string, bool) {
	switch reducer {
	case "min", "max", "sum", "count":
		return reducer + "_over_time", true
	case "mean":
		return "avg_over_time", true
	case "std_dev":
		return "stddev_over_time", true
	case "distribution", "count_nonzero":
		return "", false
	default:
		return "", false
	}
}

func mapReducerToGroupByReducer(reducer string) (string, bool) {
	switch reducer {
	case "min", "max", "sum", "count":
		return reducer, true
	case "mean":
		return "avg", true
	case "std_dev":
		return "stddev", true
	case "count_nonzero":
		return "count", true
	case "distribution":
		return "", false
	default:
		return "", false
	}
}

func mapInfixOperationToPromqlOperation(op string) (string, bool) {
	switch op {
	case string(model.TypeLogicalAnd):
		return "and", true
	case string(model.TypeLogicalOr):
		return "or", true
	case string(model.TypeAssign):
		return "==", true
	case string(model.TypeNotEquals), string(model.TypeEquals), string(model.TypeLessOrEquals),
		string(model.TypeLess), string(model.TypeMoreOrEquals), string(model.TypeMore),
		string(model.TypeAdd), string(model.TypeDiff), string(model.TypeMul), string(model.TypeDiv):
		return op, true
	default:
		return "", false
	}
}

func stringPointersMatch(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a != nil && b != nil {
		return *a == *b
	}
	return false
}

func isSubsetOf(a []string, b []string) bool {
	aMap := map[string]any{}
	for _, aItem := range a {
		aMap[aItem] = nil
	}
	for _, bItem := range b {
		if _, ok := aMap[bItem]; !ok {
			return false
		}
	}
	return true
}

func getAllElements(a []string, b []string) []string {
	aMap := map[string]any{}
	result := []string{}
	for _, aItem := range a {
		aMap[aItem] = nil
		result = append(result, aItem)
	}
	for _, bItem := range b {
		if _, ok := aMap[bItem]; !ok {
			result = append(result, bItem)
		}
	}
	return result
}

func getCommonElements(a []string, b []string) []string {
	aMap := map[string]any{}
	result := []string{}
	for _, aItem := range a {
		aMap[aItem] = nil
	}
	for _, bItem := range b {
		if _, ok := aMap[bItem]; ok {
			result = append(result, bItem)
		}
	}
	return result
}

func shiftFractionDot(digits string, current, shift int) string {
	if shift >= current {
		numZeros := shift - current
		return "0." + strings.Repeat("0", numZeros) + digits
	}
	pos := current - shift
	if pos >= len(digits) {
		numZeros := len(digits) - pos
		return digits + strings.Repeat("0", numZeros)
	}
	return digits[:pos] + "." + digits[pos:]
}

func tryConvertPercentileToQuantile(expr promqlExpression) (promqlExpression, bool) {
	scalarValue, ok := expr.(*promqlScalarValue)
	if !ok {
		return nil, false
	}

	_, err := strconv.ParseFloat(scalarValue.value, 64)
	if err != nil {
		return nil, false
	}

	tokens := strings.Split(scalarValue.value, ".")
	switch len(tokens) {
	case 1:
		return &promqlScalarValue{value: shiftFractionDot(tokens[0], len(tokens[0]), 2)}, true
	case 2:
		return &promqlScalarValue{value: shiftFractionDot(tokens[0]+tokens[1], len(tokens[0]), 2)}, true
	default:
		return nil, false
	}
}

func getJoinTypeAndAttributes(leftLabels []string, rightLabels []string) (promqlJoinType, []string, []string) {
	if leftLabels == nil && rightLabels == nil {
		return joinTypeDefault, nil, nil
	}
	if leftLabels == nil && rightLabels != nil {
		return joinTypeGroupLeft, nil, nil
	}
	if leftLabels != nil && rightLabels == nil {
		return joinTypeGroupRight, nil, nil
	}

	if isSubsetOf(leftLabels, rightLabels) && isSubsetOf(rightLabels, leftLabels) {
		return joinTypeDefault, nil, leftLabels
	}
	if isSubsetOf(leftLabels, rightLabels) {
		return joinTypeGroupLeft, rightLabels, leftLabels
	}
	if isSubsetOf(rightLabels, leftLabels) {
		return joinTypeGroupRight, leftLabels, rightLabels
	}
	allElements := getAllElements(leftLabels, rightLabels)
	if commonElements := getCommonElements(leftLabels, rightLabels); len(commonElements) > 0 {
		leftExtra := len(leftLabels) - len(commonElements)
		rightExtra := len(rightLabels) - len(commonElements)
		if leftExtra >= rightExtra {
			return joinTypeGroupLeft, commonElements, allElements
		} else {
			return joinTypeGroupRight, commonElements, allElements
		}
	}
	return joinTypeGroupLeft, nil, allElements
}

func getDisjunctiveNormalFormConjuncts(expr ast.Expression) []ast.Expression {
	switch typedExpr := expr.(type) {
	case *ast.InfixExpression:
		switch typedExpr.Operation {
		case string(model.TypeLogicalOr):
			leftExprs := getDisjunctiveNormalFormConjuncts(typedExpr.LeftExpr)
			rightExprs := getDisjunctiveNormalFormConjuncts(typedExpr.RightExpr)
			return append(leftExprs, rightExprs...)
		default:
			return []ast.Expression{typedExpr}
		}
	default:
		return []ast.Expression{typedExpr}
	}
}

func isConjunct(expr ast.Expression) bool {
	switch typedExpr := expr.(type) {
	case *ast.InfixExpression:
		switch typedExpr.Operation {
		case string(model.TypeLogicalOr):
			return false
		default:
			left := isConjunct(typedExpr.LeftExpr)
			right := isConjunct(typedExpr.RightExpr)
			return left && right
		}
	case *ast.PrefixExpression:
		switch typedExpr.Operation {
		case string(model.TypeLogicalNot):
			return false
		default:
			return true
		}
	default:
		return true
	}
}

func splitDNFExpressionIntoConjuncts(expr ast.Expression) ([]ast.Expression, *model.Error) {
	conjuncts := getDisjunctiveNormalFormConjuncts(expr)
	for _, conjunct := range conjuncts {
		if !isConjunct(conjunct) {
			return nil, newTranslatorError_unexpected_operand_in_dnf(conjunct.GetMetadata())
		}
	}
	return conjuncts, nil
}
