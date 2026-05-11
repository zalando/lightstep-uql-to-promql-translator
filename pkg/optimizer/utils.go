package optimizer

import (
	"regexp"
	"strings"

	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model/ast"
)

func convertSingleAttributeConjunctionToRegexp(expr ast.Expression) (ast.Expression, bool, string, []string) {
	switch typedExpr := expr.(type) {
	case *ast.InfixExpression:
		switch typedExpr.Operation {
		case string(model.TypeEquals), string(model.TypeAssign):
			attributeName := ""
			switch typedLeftExpr := typedExpr.LeftExpr.(type) {
			case *ast.StringLiteral:
				attributeName = typedLeftExpr.Value
			case *ast.Identifier:
				attributeName = typedLeftExpr.Value
			default:
				return expr, false, "", nil
			}
			attributeValue := ""
			switch typedRightExpr := typedExpr.RightExpr.(type) {
			case *ast.StringLiteral:
				attributeValue = typedRightExpr.Value
			case *ast.TemplateVariable:
				attributeValue = typedRightExpr.Value
			case *ast.Identifier:
				attributeValue = typedRightExpr.Value
			case *ast.IntegerLiteral:
				attributeValue = typedRightExpr.Value
			case *ast.FloatLiteral:
				attributeValue = typedRightExpr.Value
			case *ast.BooleanLiteral:
				attributeValue = typedRightExpr.Value
			case *ast.DurationLiteral:
				attributeValue = typedRightExpr.Value
			default:
				return expr, false, "", nil
			}
			return expr, true, attributeName, []string{attributeValue}
		default:
			leftExpr, leftIsOneOf, leftAttrName, leftAttrValues := convertSingleAttributeConjunctionToRegexp(typedExpr.LeftExpr)
			rightExpr, rightIsOneOf, rightAttrName, rightAttrValues := convertSingleAttributeConjunctionToRegexp(typedExpr.RightExpr)
			if leftIsOneOf && rightIsOneOf && (leftAttrName == rightAttrName) && (typedExpr.Operation == string(model.TypeLogicalOr)) {
				joinedAttrValues := append(leftAttrValues, rightAttrValues...)
				escapedValues := make([]string, 0, len(joinedAttrValues))
				for _, attrValue := range joinedAttrValues {
					escapedValues = append(escapedValues, regexp.QuoteMeta(attrValue))
				}
				return &ast.InfixExpression{
					Operation: string(model.TypeMatchRegex),
					LeftExpr:  &ast.StringLiteral{Value: leftAttrName},
					RightExpr: &ast.StringLiteral{Value: "^(" + strings.Join(escapedValues, "|") + ")$"},
				}, true, leftAttrName, joinedAttrValues
			}
			return &ast.InfixExpression{
				Operation: typedExpr.Operation,
				LeftExpr:  leftExpr,
				RightExpr: rightExpr,
			}, false, "", nil
		}
	case *ast.PrefixExpression:
		expr, _, _, _ := convertSingleAttributeConjunctionToRegexp(typedExpr.Expr)
		return &ast.PrefixExpression{
			Operation: typedExpr.Operation,
			Expr:      expr,
		}, false, "", nil
	default:
		return expr, false, "", nil
	}
}

var infixOperationInverse map[string]string = map[string]string{
	"defined":                      "undefined",
	"undefined":                    "defined",
	"contains":                     "not_contains",
	"not_contains":                 "contains",
	"phrase_match":                 "not_phrase_match",
	"not_phrase_match":             "phrase_match",
	string(model.TypeLogicalAnd):   string(model.TypeLogicalOr),
	string(model.TypeLogicalOr):    string(model.TypeLogicalAnd),
	string(model.TypeEquals):       string(model.TypeNotEquals),
	string(model.TypeAssign):       string(model.TypeNotEquals),
	string(model.TypeNotEquals):    string(model.TypeEquals),
	string(model.TypeMatchRegex):   string(model.TypeNotMachRegex),
	string(model.TypeNotMachRegex): string(model.TypeMatchRegex),
	string(model.TypeLess):         string(model.TypeMoreOrEquals),
	string(model.TypeMore):         string(model.TypeLessOrEquals),
	string(model.TypeLessOrEquals): string(model.TypeMore),
	string(model.TypeMoreOrEquals): string(model.TypeLess),
}

func inverseLogicalOperation(op string, inverse bool) string {
	if inverse {
		return infixOperationInverse[op]
	}
	return op
}

func pushLogicalNegationsDownExpressionTree(expr ast.Expression, inverse bool) (ast.Expression, *model.Error) {
	switch typedExpr := expr.(type) {
	case *ast.InfixExpression:
		switch typedExpr.Operation {
		case string(model.TypeLogicalAnd), string(model.TypeLogicalOr):
			left, err := pushLogicalNegationsDownExpressionTree(typedExpr.LeftExpr, inverse)
			if err != nil {
				return nil, err
			}
			right, err := pushLogicalNegationsDownExpressionTree(typedExpr.RightExpr, inverse)
			if err != nil {
				return nil, err
			}
			return &ast.InfixExpression{
				Operation: inverseLogicalOperation(typedExpr.Operation, inverse),
				LeftExpr:  left,
				RightExpr: right,
			}, nil
		case string(model.TypeEquals), string(model.TypeAssign), string(model.TypeNotEquals), string(model.TypeMatchRegex),
			string(model.TypeNotMachRegex), string(model.TypeLess), string(model.TypeLessOrEquals), string(model.TypeMore),
			string(model.TypeMoreOrEquals), "contains", "not_contains", "phrase_match", "not_phrase_match":
			return &ast.InfixExpression{
				Operation: inverseLogicalOperation(typedExpr.Operation, inverse),
				LeftExpr:  typedExpr.LeftExpr,
				RightExpr: typedExpr.RightExpr,
			}, nil
		default:
			return nil, newOptimizeError_unexpected_operand_in_expression(typedExpr.Metadata)
		}
	case *ast.PrefixExpression:
		switch typedExpr.Operation {
		case string(model.TypeLogicalNot):
			return pushLogicalNegationsDownExpressionTree(typedExpr.Expr, !inverse)
		case "defined", "undefined":
			return &ast.PrefixExpression{
				Operation: inverseLogicalOperation(typedExpr.Operation, inverse),
				Expr:      typedExpr.Expr,
			}, nil
		default:
			return nil, newOptimizeError_unexpected_operand_in_expression(typedExpr.Metadata)
		}
	default:
		return nil, newOptimizeError_unexpected_operand_in_expression(typedExpr.GetMetadata())
	}
}

func convertExpressionToDisjunctiveNormalForm(expr ast.Expression, limit int) ([]ast.Expression, *model.Error) {
	switch typedExpr := expr.(type) {
	case *ast.InfixExpression:
		switch typedExpr.Operation {
		case string(model.TypeLogicalAnd):
			leftItems, err := convertExpressionToDisjunctiveNormalForm(typedExpr.LeftExpr, limit)
			if err != nil {
				return nil, err
			}
			rightItems, err := convertExpressionToDisjunctiveNormalForm(typedExpr.RightExpr, limit)
			if err != nil {
				return nil, err
			}
			return mulExprSlices(leftItems, rightItems, limit, typedExpr.Metadata)
		case string(model.TypeLogicalOr):
			leftItems, err := convertExpressionToDisjunctiveNormalForm(typedExpr.LeftExpr, limit)
			if err != nil {
				return nil, err
			}
			rightItems, err := convertExpressionToDisjunctiveNormalForm(typedExpr.RightExpr, limit)
			if err != nil {
				return nil, err
			}
			return concatExprSlices(leftItems, rightItems, limit, typedExpr.Metadata)
		default:
			return []ast.Expression{typedExpr}, nil
		}
	default:
		return []ast.Expression{expr}, nil
	}
}

func mulExprSlices(left, right []ast.Expression, limit int, metadata ast.Metadata) ([]ast.Expression, *model.Error) {
	if len(left)*len(right) > limit {
		return nil, newOptimizeError_expression_is_too_long(metadata)
	}
	result := make([]ast.Expression, 0, len(left)*len(right))
	for _, a := range left {
		for _, b := range right {
			result = append(result, &ast.InfixExpression{
				Operation: string(model.TypeLogicalAnd),
				LeftExpr:  a,
				RightExpr: b,
			})
		}
	}
	return result, nil
}

func concatExprSlices(left, right []ast.Expression, limit int, metadata ast.Metadata) ([]ast.Expression, *model.Error) {
	if len(left)+len(right) > limit {
		return nil, newOptimizeError_expression_is_too_long(metadata)
	}
	result := make([]ast.Expression, 0, len(left)+len(right))
	result = append(result, left...)
	result = append(result, right...)
	return result, nil
}

func convertMultipleExpressionsToSingle(exprs []ast.Expression) ast.Expression {
	if len(exprs) <= 0 {
		return nil
	}
	var result ast.Expression = exprs[0]
	for idx := 1; idx < len(exprs); idx++ {
		result = &ast.InfixExpression{
			Operation: "||",
			LeftExpr:  result,
			RightExpr: exprs[idx],
		}
	}
	return result
}

func convertContainsOperationToRegexp(expr ast.Expression) (ast.Expression, *model.Error) {
	switch typedExpr := expr.(type) {
	case *ast.InfixExpression:
		switch typedExpr.Operation {
		case "contains":
			return convertContainsOperationToAST(typedExpr, false)
		case "not_contains":
			return convertContainsOperationToAST(typedExpr, true)
		default:
			left, err := convertContainsOperationToRegexp(typedExpr.LeftExpr)
			if err != nil {
				return nil, err
			}
			right, err := convertContainsOperationToRegexp(typedExpr.RightExpr)
			if err != nil {
				return nil, err
			}
			typedExpr.LeftExpr = left
			typedExpr.RightExpr = right
			return typedExpr, nil
		}
	case *ast.PrefixExpression:
		expr, err := convertContainsOperationToRegexp(typedExpr.Expr)
		if err != nil {
			return nil, err
		}
		typedExpr.Expr = expr
		return typedExpr, nil
	default:
		return typedExpr, nil
	}
}

func convertPhraseMatchOperationToRegexp(expr ast.Expression) (ast.Expression, *model.Error) {
	switch typedExpr := expr.(type) {
	case *ast.InfixExpression:
		switch typedExpr.Operation {
		case "phrase_match":
			return convertPhraseMatchOperationAST(typedExpr, false)
		case "not_phrase_match":
			return convertPhraseMatchOperationAST(typedExpr, true)
		default:
			left, err := convertPhraseMatchOperationToRegexp(typedExpr.LeftExpr)
			if err != nil {
				return nil, err
			}
			right, err := convertPhraseMatchOperationToRegexp(typedExpr.RightExpr)
			if err != nil {
				return nil, err
			}
			typedExpr.LeftExpr = left
			typedExpr.RightExpr = right
			return typedExpr, nil
		}
	case *ast.PrefixExpression:
		expr, err := convertPhraseMatchOperationToRegexp(typedExpr.Expr)
		if err != nil {
			return nil, err
		}
		typedExpr.Expr = expr
		return typedExpr, nil
	default:
		return typedExpr, nil
	}
}

func convertContainsOperationToAST(expr *ast.InfixExpression, inverse bool) (*ast.InfixExpression, *model.Error) {
	arg, ok := expr.RightExpr.(*ast.StringLiteral)
	if !ok {
		return nil, newOptimizeError("contains operator second argument invalid - must be string", expr.RightExpr.GetMetadata())
	}

	operation := "=~"
	if inverse {
		operation = "!~"
	}

	return &ast.InfixExpression{
		Operation: operation,
		LeftExpr:  expr.LeftExpr,
		RightExpr: &ast.StringLiteral{
			Value: ".*" + regexp.QuoteMeta(arg.Value) + ".*",
		},
	}, nil
}

func convertPhraseMatchOperationAST(expr *ast.InfixExpression, inverse bool) (*ast.InfixExpression, *model.Error) {
	body, ok := expr.LeftExpr.(*ast.Identifier)
	if !ok {
		return nil, newOptimizeError("phrase_match operator first argument invalid - must be identifier", expr.LeftExpr.GetMetadata())
	}
	if body.Value != "body" {
		return nil, newOptimizeError("phrase_match operator first argument invalid - must be `body`", expr.LeftExpr.GetMetadata())
	}

	arg, ok := expr.RightExpr.(*ast.StringLiteral)
	if !ok {
		return nil, newOptimizeError("phrase_match operator second argument invalid - must be string", expr.RightExpr.GetMetadata())
	}

	operation := "=~"
	if inverse {
		operation = "!~"
	}

	return &ast.InfixExpression{
		Operation: operation,
		LeftExpr:  &ast.StringLiteral{Value: "otel_log_body"},
		RightExpr: &ast.StringLiteral{
			Value: ".*" + regexp.QuoteMeta(arg.Value) + ".*",
		},
	}, nil
}

func convertTemplateVariablesToString(expr ast.Expression) ast.Expression {
	switch typedExpr := expr.(type) {
	case *ast.InfixExpression:
		typedExpr.LeftExpr = convertTemplateVariablesToString(typedExpr.LeftExpr)
		typedExpr.RightExpr = convertTemplateVariablesToString(typedExpr.RightExpr)
	case *ast.PrefixExpression:
		typedExpr.Expr = convertTemplateVariablesToString(typedExpr.Expr)
	case *ast.TemplateVariable:
		return &ast.StringLiteral{
			Value:    typedExpr.Value,
			Metadata: typedExpr.Metadata,
		}
	}
	return expr
}
