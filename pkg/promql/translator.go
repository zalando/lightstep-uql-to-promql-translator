package promql

import (
	"fmt"

	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model/ast"
)

const (
	MAX_DISJUNCTS_IN_FILTER_EXPRESSION = 128
)

type Translator struct {
	metricNameToType map[string]MetricType
	metricConfig     SpecialMetricConfig
}

func New(metricNameToType map[string]MetricType, metricConfig SpecialMetricConfig) *Translator {
	return &Translator{metricNameToType: metricNameToType, metricConfig: metricConfig}
}

func (tr *Translator) isValidFilterExpression(expr ast.Expression) bool {
	switch typedExpr := expr.(type) {
	case *ast.InfixExpression:
		switch typedExpr.Operation {
		case string(model.TypeLogicalAnd), string(model.TypeLogicalOr), string(model.TypeEquals), string(model.TypeAssign),
			string(model.TypeNotEquals), string(model.TypeMatchRegex), string(model.TypeNotMachRegex), string(model.TypeLess),
			string(model.TypeLessOrEquals), string(model.TypeMore), string(model.TypeMoreOrEquals),
			"contains", "not_contains", "phrase_match", "not_phrase_match":
			leftOk := tr.isValidFilterExpression(typedExpr.LeftExpr)
			rightOk := tr.isValidFilterExpression(typedExpr.RightExpr)
			return leftOk && rightOk
		default:
			return false
		}
	case *ast.PrefixExpression:
		switch typedExpr.Operation {
		case string(model.TypeLogicalNot), "defined", "undefined":
			return tr.isValidFilterExpression(typedExpr.Expr)
		default:
			return false
		}
	case *ast.TemplateVariable, *ast.Identifier, *ast.StringLiteral, *ast.IntegerLiteral, *ast.FloatLiteral, *ast.BooleanLiteral:
		return true
	default:
		return false
	}
}

func (tr *Translator) disjunctToLabelMatcher(expr ast.Expression) ([]promqlLabelMatcher, *model.Error) {
	switch typedExpr := expr.(type) {
	case *ast.InfixExpression:
		switch typedExpr.Operation {
		case string(model.TypeLogicalAnd):
			leftAttrs, err := tr.disjunctToLabelMatcher(typedExpr.LeftExpr)
			if err != nil {
				return nil, err
			}
			rightAttrs, err := tr.disjunctToLabelMatcher(typedExpr.RightExpr)
			if err != nil {
				return nil, err
			}
			return append(leftAttrs, rightAttrs...), nil
		case string(model.TypeLogicalOr):
			return nil, newTranslatorError_unexpected_operand_in_disjunct(typedExpr.Metadata)
		case string(model.TypeEquals), string(model.TypeAssign), string(model.TypeNotEquals),
			string(model.TypeMatchRegex), string(model.TypeNotMachRegex), string(model.TypeLess),
			string(model.TypeLessOrEquals), string(model.TypeMore), string(model.TypeMoreOrEquals):
			attributeName, err := tryGetAttributeKey(typedExpr.LeftExpr)
			if err != nil {
				return nil, err
			}
			attributeValue, err := tryGetAttributeValue(typedExpr.RightExpr)
			if err != nil {
				return nil, err
			}
			attributeOperation, opExists := mapOperationToPromQL(typedExpr.Operation)
			if !opExists {
				return nil, newTranslatorError_unexpected_operand_in_disjunct(typedExpr.Metadata)
			}
			return []promqlLabelMatcher{{
				operation: attributeOperation,
				key:       attributeName,
				value:     attributeValue,
			}}, nil
		default:
			return nil, newTranslatorError_unexpected_operand_in_disjunct(typedExpr.Metadata)
		}
	case *ast.PrefixExpression:
		switch typedExpr.Operation {
		case "defined":
			attributeName, err := tryGetAttributeKey(typedExpr.Expr)
			if err != nil {
				return nil, err
			}
			return []promqlLabelMatcher{{
				operation: "!=",
				key:       attributeName,
				value:     "\"\"",
			}}, nil
		case "undefined":
			attributeName, err := tryGetAttributeKey(typedExpr.Expr)
			if err != nil {
				return nil, err
			}
			return []promqlLabelMatcher{{
				operation: "!~",
				key:       attributeName,
				value:     "\".+\"",
			}}, nil
		default:
			return nil, newTranslatorError_unexpected_operand_in_disjunct(typedExpr.Metadata)
		}
	default:
		return nil, newTranslatorError_unexpected_operand_in_disjunct(typedExpr.GetMetadata())
	}
}

func (tr *Translator) applyModifierFilter(query *PromqlQuery, stage *ast.ModifierStageFilter, limit int) *model.Error {
	if !tr.isValidFilterExpression(stage.Expr) {
		return newTranslatorError_invalid_expression(stage.Expr.GetMetadata())
	}

	disjuncts, err := splitDNFExpressionIntoConjuncts(stage.Expr)
	if err != nil {
		return err
	}

	if len(disjuncts)*len(query.subqueries) > limit {
		return newTranslatorError_expression_is_too_long(stage.Expr.GetMetadata())
	}

	result := make([]promqlExpression, 0, len(disjuncts)*len(query.subqueries))
	for _, sq := range query.subqueries {
		vectorSelector, isVectorSelector := sq.(*promqlVectorSelector)
		if !isVectorSelector {
			return newTranslatorError("filter modifier can be applied only to vector selector", stage.Metadata)
		}
		for _, disjunct := range disjuncts {
			labelMatcher, err := tr.disjunctToLabelMatcher(disjunct)
			if err != nil {
				return err
			}
			result = append(result, &promqlVectorSelector{
				name:          vectorSelector.name,
				rangeSelector: vectorSelector.rangeSelector,
				offset:        vectorSelector.offset,
				matchers:      labelMatcher,
			})
		}
	}

	query.subqueries = result
	return nil
}

func (tr *Translator) applyAligner(
	query *PromqlQuery, funcName string, inputWindow *ast.DurationLiteral,
	outputPeriod *ast.DurationLiteral, useRateInterval bool,
) *model.Error {
	defaultInterval := getProperDefaultInterval(useRateInterval)
	var currentInputWindow string = defaultInterval
	var currentOutputPeriod *string = nil
	var nonNilCurrentOutputPeriod string = defaultInterval

	if inputWindow != nil {
		currentInputWindow = inputWindow.Value
	}
	if outputPeriod != nil {
		nonNilCurrentOutputPeriod = outputPeriod.Value
		currentOutputPeriod = &outputPeriod.Value
	}

	var result []promqlExpression = nil

	for _, sq := range query.subqueries {
		switch typedQuery := sq.(type) {
		case *promqlVectorSelector:
			typedQuery.rangeSelector = &currentInputWindow
			typedQuery.resolution = currentOutputPeriod
			result = append(result, &promqlFunctionCall{
				fname: funcName,
				args:  []promqlExpression{sq},
			})
		default:
			result = append(result, &promqlFunctionCall{
				fname: funcName,
				args: []promqlExpression{&promqlSubqueryExpression{
					expr:          &promqlParenthesis{expr: sq},
					rangeSelector: currentInputWindow,
					resolution:    &nonNilCurrentOutputPeriod,
				}},
			})
		}
	}

	query.subqueries = result
	query.outputPeriod = currentOutputPeriod
	return nil
}

func (tr *Translator) applyModifierTimeShift(query *PromqlQuery, stage *ast.ModifierStageTimeShift) *model.Error {
	var result []promqlExpression = nil

	for _, sq := range query.subqueries {
		switch typedQuery := sq.(type) {
		case *promqlVectorSelector:
			typedQuery.offset = &stage.ShiftDuration.Value
			result = append(result, typedQuery)
		default:
			return newTranslatorError("time_shift stage must always be placed before the first aligner", stage.Metadata)
		}
	}

	query.subqueries = result
	return nil
}

func (tr *Translator) applyModifierGroupBy(query *PromqlQuery, stage *ast.ModifierStageGroupBy) *model.Error {
	var result []promqlExpression = nil

	reducer, exists := mapReducerToGroupByReducer(stage.Reducer)
	if !exists {
		return newTranslatorError(fmt.Sprintf("unsupported group_by reducer %s", stage.Reducer), stage.Metadata)
	}

	labels := []string{}
	rawLabels := []string{}

	for _, label := range stage.Labels {
		labels = append(labels, promqlAttributeFormat(label.Value))
		rawLabels = append(rawLabels, label.Value)
	}

	for _, sq := range query.subqueries {
		resultExpr := sq
		if stage.Reducer == "count_nonzero" {
			resultExpr = &promqlBinaryExpression{
				operation: "!=",
				leftExpr:  resultExpr,
				rightExpr: &promqlScalarValue{value: "0"},
			}
		}
		result = append(result, &promqlGroupByExpression{
			operation: reducer,
			expr:      resultExpr,
			labels:    labels,
		})
	}

	query.subqueries = result
	query.outputLabels = rawLabels
	return nil
}

func (tr *Translator) applyModifierTopOrBottom(
	query *PromqlQuery, operation string, amount string,
	labels []ast.Identifier, reducer string, window *ast.DurationLiteral,
	metadata ast.Metadata,
) *model.Error {
	var result []promqlExpression = nil

	reducerFunc, exists := mapReducerToReducerFuncName(reducer)
	if !exists {
		return newTranslatorError(fmt.Sprintf("unsupported group_by reducer %s", reducer), metadata)
	}

	strLabels := []string{}
	for _, label := range labels {
		strLabels = append(strLabels, promqlAttributeFormat(label.Value))
	}

	aggRange := "$__range"
	if window != nil {
		aggRange = window.Value
	}

	for _, sq := range query.subqueries {
		emptyString := ""
		result = append(result, &promqlTopOrBottomExpression{
			operation: operation,
			labels:    strLabels,
			amount:    amount,
			mainExpr:  sq,
			reduceExpr: &promqlFunctionCall{
				fname: reducerFunc,
				args: []promqlExpression{
					&promqlSubqueryExpression{
						expr:          sq,
						rangeSelector: aggRange,
						resolution:    &emptyString,
					},
				},
			},
		})
	}

	query.subqueries = result
	return nil
}

func (tr *Translator) applyModifierFill(query *PromqlQuery, stage *ast.ModifierStageFill) *model.Error {
	var result []promqlExpression = nil

	for _, sq := range query.subqueries {
		result = append(result, &promqlParenthesis{
			expr: &promqlBinaryExpression{
				operation: "or",
				leftExpr:  sq,
				rightExpr: &promqlBinaryExpression{
					operation: "*",
					leftExpr:  &promqlScalarValue{value: stage.Number.String()},
					rightExpr: &promqlFunctionCall{
						fname: "absent",
						args:  []promqlExpression{sq},
					},
				},
			},
		})
	}

	query.subqueries = result
	return nil
}

func (tr *Translator) pointFilterExprToPromqlExpr(expr ast.Expression, valueExpression promqlExpression) (promqlExpression, *model.Error) {
	switch typedExpr := expr.(type) {
	case *ast.InfixExpression:
		leftExpr, err := tr.pointFilterExprToPromqlExpr(typedExpr.LeftExpr, valueExpression)
		if err != nil {
			return nil, err
		}
		rightExpr, err := tr.pointFilterExprToPromqlExpr(typedExpr.RightExpr, valueExpression)
		if err != nil {
			return nil, err
		}
		promqlOperation, found := mapInfixOperationToPromqlOperation(typedExpr.Operation)
		if !found {
			return nil, newTranslatorError_invalid_expression(expr.GetMetadata())
		}
		return &promqlBinaryExpression{
			operation: promqlOperation,
			leftExpr:  leftExpr,
			rightExpr: rightExpr,
		}, nil
	case *ast.IntegerLiteral:
		return &promqlScalarValue{value: typedExpr.Value}, nil
	case *ast.FloatLiteral:
		return &promqlScalarValue{value: typedExpr.Value}, nil
	case *ast.Identifier:
		if typedExpr.Value != "value" {
			return nil, newTranslatorError("`value` is the only allowed identifier in point_filter expressions", expr.GetMetadata())
		}
		return valueExpression, nil
	default:
		return nil, newTranslatorError_unexpected_operand_in_expression(expr.GetMetadata())
	}
}

func (tr *Translator) isValidPointFilterExpression(expr ast.Expression) bool {
	switch typedExpr := expr.(type) {
	case *ast.InfixExpression:
		switch typedExpr.Operation {
		case string(model.TypeLogicalAnd), string(model.TypeLogicalOr), string(model.TypeEquals), string(model.TypeAssign),
			string(model.TypeNotEquals), string(model.TypeMatchRegex), string(model.TypeNotMachRegex), string(model.TypeLess),
			string(model.TypeLessOrEquals), string(model.TypeMore), string(model.TypeMoreOrEquals), string(model.TypeAdd),
			string(model.TypeDiff), string(model.TypeMul), string(model.TypeDiv), "contains", "not_contains":
			leftOk := tr.isValidPointFilterExpression(typedExpr.LeftExpr)
			rightOk := tr.isValidPointFilterExpression(typedExpr.RightExpr)
			return leftOk && rightOk
		default:
			return false
		}
	case *ast.PrefixExpression:
		switch typedExpr.Operation {
		case string(model.TypeLogicalNot), "defined", "undefined":
			return tr.isValidPointFilterExpression(typedExpr.Expr)
		default:
			return false
		}
	case *ast.TemplateVariable, *ast.Identifier, *ast.StringLiteral, *ast.IntegerLiteral, *ast.FloatLiteral, *ast.BooleanLiteral:
		return true
	default:
		return false
	}
}

func (tr *Translator) applyModifierPointFilter(query *PromqlQuery, stage *ast.ModifierStagePointFilter, limit int) *model.Error {
	if !tr.isValidPointFilterExpression(stage.Expr) {
		return newTranslatorError_invalid_expression(stage.Metadata)
	}

	disjuncts, err := splitDNFExpressionIntoConjuncts(stage.Expr)
	if err != nil {
		return err
	}

	if len(disjuncts)*len(query.subqueries) > limit {
		return newTranslatorError_expression_is_too_long(stage.Expr.GetMetadata())
	}

	result := make([]promqlExpression, 0, len(disjuncts)*len(query.subqueries))

	for _, sq := range query.subqueries {
		for _, disjunct := range disjuncts {
			promExpr, err := tr.pointFilterExprToPromqlExpr(disjunct, sq)
			if err != nil {
				return err
			}
			result = append(result, &promqlParenthesis{expr: promExpr})
		}
	}

	query.subqueries = result
	return nil
}

func (tr *Translator) isValidValueExpression(expr ast.Expression) bool {
	switch typedExpr := expr.(type) {
	case *ast.InfixExpression:
		switch typedExpr.Operation {
		case string(model.TypeAdd), string(model.TypeMul), string(model.TypeDiff), string(model.TypeDiv), "percentile", "pow", "max", "min":
			leftOk := tr.isValidValueExpression(typedExpr.LeftExpr)
			rightOk := tr.isValidValueExpression(typedExpr.RightExpr)
			return leftOk && rightOk
		default:
			return false
		}
	case *ast.PrefixExpression:
		switch typedExpr.Operation {
		case string(model.TypeDiff), "abs", "timestamp", "floor", "ceil", "round", "dist_sum", "dist_count":
			return tr.isValidValueExpression(typedExpr.Expr)
		default:
			return false
		}
	case *ast.TemplateVariable, *ast.Identifier, *ast.IntegerLiteral, *ast.FloatLiteral:
		return true
	default:
		return false
	}
}

func (tr *Translator) pointExprToPromqlExpr(expr ast.Expression, values map[string]promqlExpression) (promqlExpression, *model.Error) {
	switch typedExpr := expr.(type) {
	case *ast.InfixExpression:
		switch typedExpr.Operation {
		case string(model.TypeAdd), string(model.TypeMul), string(model.TypeDiff), string(model.TypeDiv):
			leftExpr, err := tr.pointExprToPromqlExpr(typedExpr.LeftExpr, values)
			if err != nil {
				return nil, err
			}
			rightExpr, err := tr.pointExprToPromqlExpr(typedExpr.RightExpr, values)
			if err != nil {
				return nil, err
			}
			return &promqlParenthesis{
				expr: &promqlBinaryExpression{
					operation: typedExpr.Operation,
					leftExpr:  leftExpr,
					rightExpr: rightExpr,
				},
			}, nil
		case "pow":
			leftExpr, err := tr.pointExprToPromqlExpr(typedExpr.LeftExpr, values)
			if err != nil {
				return nil, err
			}
			rightExpr, err := tr.pointExprToPromqlExpr(typedExpr.RightExpr, values)
			if err != nil {
				return nil, err
			}
			return &promqlParenthesis{
				expr: &promqlBinaryExpression{
					operation: "^",
					leftExpr:  leftExpr,
					rightExpr: rightExpr,
				},
			}, nil
		case "min":
			leftExpr, err := tr.pointExprToPromqlExpr(typedExpr.LeftExpr, values)
			if err != nil {
				return nil, err
			}
			rightExpr, err := tr.pointExprToPromqlExpr(typedExpr.RightExpr, values)
			if err != nil {
				return nil, err
			}
			return &promqlFunctionCall{
				fname: "clamp_min",
				args:  []promqlExpression{leftExpr, rightExpr},
			}, nil
		case "max":
			leftExpr, err := tr.pointExprToPromqlExpr(typedExpr.LeftExpr, values)
			if err != nil {
				return nil, err
			}
			rightExpr, err := tr.pointExprToPromqlExpr(typedExpr.RightExpr, values)
			if err != nil {
				return nil, err
			}
			return &promqlFunctionCall{
				fname: "clamp_max",
				args:  []promqlExpression{leftExpr, rightExpr},
			}, nil
		case "percentile":
			value, err := tr.pointExprToPromqlExpr(typedExpr.LeftExpr, values)
			if err != nil {
				return nil, err
			}
			perc, err := tr.pointExprToPromqlExpr(typedExpr.RightExpr, values)
			if err != nil {
				return nil, err
			}
			if quantile, ok := tryConvertPercentileToQuantile(perc); ok {
				perc = quantile
			} else {
				perc = &promqlBinaryExpression{operation: "/", leftExpr: perc, rightExpr: &promqlScalarValue{value: "100"}}
			}
			return &promqlFunctionCall{
				fname: "histogram_quantile",
				args:  []promqlExpression{perc, value},
			}, nil
		default:
			return nil, newTranslatorError_invalid_expression(expr.GetMetadata())
		}
	case *ast.PrefixExpression:
		switch typedExpr.Operation {
		case string(model.TypeDiff):
			innerExpr, err := tr.pointExprToPromqlExpr(typedExpr.Expr, values)
			if err != nil {
				return nil, err
			}
			return &promqlUnaryExpression{
				operation: typedExpr.Operation,
				expr:      &promqlParenthesis{expr: innerExpr},
			}, nil
		case "abs", "timestamp", "floor", "ceil", "round":
			innerExpr, err := tr.pointExprToPromqlExpr(typedExpr.Expr, values)
			if err != nil {
				return nil, err
			}
			return &promqlFunctionCall{
				fname: typedExpr.Operation,
				args:  []promqlExpression{innerExpr},
			}, nil
		case "dist_sum":
			value, err := tr.pointExprToPromqlExpr(typedExpr.Expr, values)
			if err != nil {
				return nil, err
			}
			return &promqlFunctionCall{
				fname: "histogram_sum",
				args:  []promqlExpression{value},
			}, nil
		case "dist_count":
			value, err := tr.pointExprToPromqlExpr(typedExpr.Expr, values)
			if err != nil {
				return nil, err
			}
			return &promqlFunctionCall{
				fname: "histogram_count",
				args:  []promqlExpression{value},
			}, nil
		default:
			return nil, newTranslatorError_invalid_expression(expr.GetMetadata())
		}
	case *ast.TemplateVariable:
		return &promqlScalarValue{value: typedExpr.Value}, nil
	case *ast.Identifier:
		if value, ok := values[typedExpr.Value]; ok {
			return value, nil
		}
		return nil, newTranslatorError_invalid_expression(expr.GetMetadata())
	case *ast.IntegerLiteral:
		return &promqlScalarValue{value: typedExpr.Value}, nil
	case *ast.FloatLiteral:
		return &promqlScalarValue{value: typedExpr.Value}, nil
	default:
		return nil, newTranslatorError_invalid_expression(expr.GetMetadata())
	}
}

func (tr *Translator) applyModifierPoint(query *PromqlQuery, stage *ast.ModifierStagePoint, limit int, pointId int) *model.Error {
	for _, valueExpression := range stage.Expressions {
		if !tr.isValidValueExpression(valueExpression) {
			return newTranslatorError_invalid_expression(valueExpression.GetMetadata())
		}
	}

	multipleExpressions := len(stage.Expressions) > 1
	pointLabel := fmt.Sprintf("uql_point_%d", pointId)

	if len(stage.Expressions)*len(query.subqueries) > limit {
		return newTranslatorError_expression_is_too_long(stage.Metadata)
	}

	result := make([]promqlExpression, 0, len(stage.Expressions)*len(query.subqueries))

	for _, sq := range query.subqueries {
		for idx, vExpr := range stage.Expressions {
			promExpr, err := tr.pointExprToPromqlExpr(vExpr, map[string]promqlExpression{"value": sq})
			if err != nil {
				return err
			}
			if multipleExpressions {
				promExpr = &promqlFunctionCall{
					fname: "label_replace",
					args: []promqlExpression{
						promExpr,
						&promqlScalarValue{value: "\"" + pointLabel + "\""},
						&promqlScalarValue{value: fmt.Sprintf("\"%d\"", idx)},
						&promqlScalarValue{value: "\"\""},
						&promqlScalarValue{value: "\"\""},
					},
				}
			}
			result = append(result, promExpr)
		}
	}

	query.subqueries = result
	if multipleExpressions {
		query.outputLabels = append(query.outputLabels, pointLabel)
	}
	return nil
}

func (tr *Translator) removeAlignerStagesBeforeGroupByCount(pipeline []ast.Stage) []ast.Stage {
	var result []ast.Stage = nil
	groupByCountIdx := -1

	for idx, stage := range pipeline {
		if typedStage, isGroupBy := stage.(*ast.ModifierStageGroupBy); isGroupBy {
			if typedStage.Reducer == "count" || typedStage.Reducer == "count_nonzero" {
				groupByCountIdx = idx
			}
		}
	}

	for idx, stage := range pipeline {
		switch stage.(type) {
		case *ast.AlignerStageDelta, *ast.AlignerStageRate, *ast.AlignerStageReduce, *ast.AlignerStageLatest:
			if idx > groupByCountIdx {
				result = append(result, stage)
			}
		default:
			result = append(result, stage)
		}
	}

	return result
}

func (tr *Translator) addDeltaBeforeFirstReduceAligner(pipeline []ast.Stage) []ast.Stage {
	result := []ast.Stage{}
	isFirstAlignerStage := true
	for _, stage := range pipeline {
		switch stage.(type) {
		case *ast.AlignerStageDelta, *ast.AlignerStageRate, *ast.AlignerStageLatest:
			isFirstAlignerStage = false
			result = append(result, stage)
		case *ast.AlignerStageReduce:
			if isFirstAlignerStage {
				result = append(result, &ast.AlignerStageDelta{
					InputWindow:  nil,
					OutputPeriod: nil,
				})
			}
			isFirstAlignerStage = false
			result = append(result, stage)
		default:
			result = append(result, stage)
		}
	}
	return result
}

func (tr *Translator) applyStages(query *PromqlQuery, pipeline []ast.Stage, metricType MetricType) *model.Error {
	pipeline = tr.removeAlignerStagesBeforeGroupByCount(pipeline)
	if metricType == METRIC_TYPE_SUM {
		pipeline = tr.addDeltaBeforeFirstReduceAligner(pipeline)
	}
	pointId := 0

	for _, pipelineStage := range pipeline {
		switch stage := pipelineStage.(type) {
		case *ast.ModifierStageFilter:
			err := tr.applyModifierFilter(query, stage, MAX_DISJUNCTS_IN_FILTER_EXPRESSION)
			if err != nil {
				return err
			}
		case *ast.ModifierStageTimeShift:
			err := tr.applyModifierTimeShift(query, stage)
			if err != nil {
				return err
			}
		case *ast.ModifierStageGroupBy:
			err := tr.applyModifierGroupBy(query, stage)
			if err != nil {
				return err
			}
		case *ast.ModifierStageTop:
			err := tr.applyModifierTopOrBottom(query, "topk", stage.Amount.Value, stage.Labels, stage.Reducer, stage.Window, stage.Metadata)
			if err != nil {
				return err
			}
		case *ast.ModifierStageBottom:
			err := tr.applyModifierTopOrBottom(query, "bottomk", stage.Amount.Value, stage.Labels, stage.Reducer, stage.Window, stage.Metadata)
			if err != nil {
				return err
			}
		case *ast.ModifierStageFill:
			err := tr.applyModifierFill(query, stage)
			if err != nil {
				return err
			}
		case *ast.ModifierStagePointFilter:
			err := tr.applyModifierPointFilter(query, stage, MAX_DISJUNCTS_IN_FILTER_EXPRESSION)
			if err != nil {
				return err
			}
		case *ast.ModifierStagePoint:
			err := tr.applyModifierPoint(query, stage, MAX_DISJUNCTS_IN_FILTER_EXPRESSION, pointId)
			if err != nil {
				return err
			}
			pointId++
		case *ast.AlignerStageDelta:
			var funcName string
			switch metricType {
			case METRIC_TYPE_GAUGE:
				funcName = "delta"
			default:
				funcName = "increase"
			}
			err := tr.applyAligner(query, funcName, stage.InputWindow, stage.OutputPeriod, true)
			if err != nil {
				return err
			}
		case *ast.AlignerStageRate:
			var funcName string
			switch metricType {
			case METRIC_TYPE_GAUGE:
				funcName = "deriv"
			default:
				funcName = "rate"
			}
			err := tr.applyAligner(query, funcName, stage.InputWindow, stage.OutputPeriod, true)
			if err != nil {
				return err
			}
		case *ast.AlignerStageLatest:
			err := tr.applyAligner(query, "last_over_time", stage.InputWindow, stage.OutputPeriod, false)
			if err != nil {
				return err
			}
		case *ast.AlignerStageReduce:
			funcName, exists := mapReducerToReducerFuncName(stage.Reducer)
			if !exists {
				return newTranslatorError(fmt.Sprintf("unsupported reduce aligner reducer %s", stage.Reducer), stage.Metadata)
			}
			err := tr.applyAligner(query, funcName, stage.InputWindow, stage.OutputPeriod, false)
			if err != nil {
				return err
			}
		default:
			return newTranslatorError("unknown pipeline stage", stage.GetMetadata())
		}
	}
	return nil
}

func (tr *Translator) translatePipeline(query *ast.Query) (*PromqlQuery, *model.Error) {
	if len(query.Pipeline) == 0 {
		return nil, newTranslatorError("empty query", query.Metadata)
	}
	if _, ok := query.Pipeline[0].(ast.FetchStage); !ok {
		return nil, newTranslatorError("pipeline first stage must be fetch stage", query.Pipeline[0].GetMetadata())
	}
	for idx, stage := range query.Pipeline {
		if fetchStage, ok := stage.(ast.FetchStage); ok {
			if idx != 0 {
				return nil, newTranslatorError("multiple fetch stages are forbidden", fetchStage.GetMetadata())
			}
		}
	}

	fetchStage := query.Pipeline[0].(ast.FetchStage)
	otherStages := query.Pipeline[1:len(query.Pipeline)]

	switch stage := fetchStage.(type) {
	case *ast.FetchStageMetric:
		query := &PromqlQuery{
			subqueries: []promqlExpression{
				&promqlVectorSelector{
					name:          stage.MetricName,
					matchers:      nil,
					rangeSelector: nil,
					offset:        nil,
				},
			},
			outputPeriod: nil,
			outputLabels: nil,
		}
		metricType, metricTypeFound := tr.metricNameToType[stage.MetricName]
		if !metricTypeFound {
			metricType = METRIC_TYPE_GAUGE
		}
		err := tr.applyStages(query, otherStages, metricType)
		if err != nil {
			return nil, err
		}
		return query, nil
	case *ast.FetchStageConstant:
		query := &PromqlQuery{
			subqueries: []promqlExpression{
				&promqlConstantVector{value: stage.Value.String()},
			},
			outputPeriod: nil,
			outputLabels: nil,
		}
		return query, nil
	case *ast.FetchStageSpans:
		var metricName string
		var metricType MetricType

		switch stage.FetchType {
		case "count":
			metricName = tr.metricConfig.SpansCount
			metricType = METRIC_TYPE_SUM
		case "count_unadjusted":
			metricName = tr.metricConfig.SpansCountUnadjusted
			metricType = METRIC_TYPE_SUM
		case "latency":
			metricName = tr.metricConfig.SpansLatency
			metricType = METRIC_TYPE_HISTOGRAM
		default:
			return nil, newTranslatorError("unsupported spans query type", stage.Metadata)
		}

		query := &PromqlQuery{
			subqueries: []promqlExpression{
				&promqlVectorSelector{
					name:          metricName,
					matchers:      nil,
					rangeSelector: nil,
					offset:        nil,
				},
			},
			outputPeriod: nil,
			outputLabels: nil,
		}
		err := tr.applyStages(query, otherStages, metricType)
		if err != nil {
			return nil, err
		}
		return query, nil
	case *ast.FetchStageLogs:
		if stage.FetchType != "count" {
			return nil, newTranslatorError("only `logs count` queries are supported", stage.Metadata)
		}
		query := &PromqlQuery{
			subqueries: []promqlExpression{
				&promqlVectorSelector{
					name:          tr.metricConfig.LogsCount,
					matchers:      nil,
					rangeSelector: nil,
					offset:        nil,
				},
			},
			outputPeriod: nil,
			outputLabels: nil,
		}
		err := tr.applyStages(query, otherStages, METRIC_TYPE_SUM)
		if err != nil {
			return nil, err
		}
		return query, nil
	default:
		return nil, newTranslatorError("unknown fetch stage", fetchStage.GetMetadata())
	}
}

func (tr *Translator) applyDefaultValue(query *ast.Query, defaultValue ast.NumberLiteral) *model.Error {
	switch query.Type {
	case ast.QueryTypeDefault:
		query.Pipeline = append(query.Pipeline, &ast.ModifierStageFill{Number: defaultValue})
	case ast.QueryTypeNamedJoin:
		query.NamedJoin.Stages = append(query.NamedJoin.Stages, &ast.ModifierStageFill{Number: defaultValue})
	case ast.QueryTypeUnnamedJoin:
		query.UnnamedJoin.Stages = append(query.UnnamedJoin.Stages, &ast.ModifierStageFill{Number: defaultValue})
	default:
		return newTranslatorError("unknown query type", query.Metadata)
	}
	return nil
}

func (tr *Translator) applyCommonStages(query *ast.Query, commonStages []ast.Stage) *model.Error {
	switch query.Type {
	case ast.QueryTypeDefault:
		query.Pipeline = append(query.Pipeline, commonStages...)
	case ast.QueryTypeNamedJoin:
		query.NamedJoin.Stages = append(query.NamedJoin.Stages, commonStages...)
	case ast.QueryTypeUnnamedJoin:
		query.UnnamedJoin.Stages = append(query.UnnamedJoin.Stages, commonStages...)
	default:
		return newTranslatorError("unknown query type", query.Metadata)
	}
	return nil
}

func (tr *Translator) joinSubqueriesInPromqlQuery(query *PromqlQuery, astQuery *ast.Query) (*promqlSingleExpressionQuery, *model.Error) {
	if len(query.subqueries) == 0 {
		return nil, newTranslatorError("join query doesn't have subqueries", astQuery.Metadata)
	}
	if len(query.subqueries) == 1 {
		return &promqlSingleExpressionQuery{
			expr:         query.subqueries[0],
			outputPeriod: query.outputPeriod,
			outputLabels: query.outputLabels,
		}, nil
	}
	var result promqlExpression = query.subqueries[0]
	for _, subquery := range query.subqueries {
		result = &promqlBinaryExpression{
			operation: "or",
			leftExpr:  result,
			rightExpr: subquery,
		}
	}
	return &promqlSingleExpressionQuery{
		expr:         result,
		outputPeriod: query.outputPeriod,
		outputLabels: query.outputLabels,
	}, nil
}

func (tr *Translator) joinExprToPromqlQuery(expr ast.Expression, values map[string]*promqlSingleExpressionQuery) (*promqlSingleExpressionQuery, bool, *model.Error) {
	switch typedExpr := expr.(type) {
	case *ast.InfixExpression:
		switch typedExpr.Operation {
		case string(model.TypeAdd), string(model.TypeMul), string(model.TypeDiff), string(model.TypeDiv), string(model.TypeLess), string(model.TypeMore):
			leftQuery, isLeftScalar, err := tr.joinExprToPromqlQuery(typedExpr.LeftExpr, values)
			if err != nil {
				return nil, false, err
			}
			rightQuery, isRightScalar, err := tr.joinExprToPromqlQuery(typedExpr.RightExpr, values)
			if err != nil {
				return nil, false, err
			}
			if isLeftScalar && isRightScalar {
				return &promqlSingleExpressionQuery{
					expr: &promqlParenthesis{expr: &promqlBinaryExpression{
						operation: typedExpr.Operation,
						leftExpr:  leftQuery.expr,
						rightExpr: rightQuery.expr,
					}},
					outputPeriod: nil,
					outputLabels: nil,
				}, true, nil
			}
			if isLeftScalar {
				return &promqlSingleExpressionQuery{
					expr: &promqlParenthesis{expr: &promqlBinaryExpression{
						operation: typedExpr.Operation,
						leftExpr:  leftQuery.expr,
						rightExpr: rightQuery.expr,
					}},
					outputPeriod: rightQuery.outputPeriod,
					outputLabels: rightQuery.outputLabels,
				}, false, nil
			}
			if isRightScalar {
				return &promqlSingleExpressionQuery{
					expr: &promqlParenthesis{expr: &promqlBinaryExpression{
						operation: typedExpr.Operation,
						leftExpr:  leftQuery.expr,
						rightExpr: rightQuery.expr,
					}},
					outputPeriod: leftQuery.outputPeriod,
					outputLabels: leftQuery.outputLabels,
				}, false, nil
			}
			if !stringPointersMatch(leftQuery.outputPeriod, rightQuery.outputPeriod) {
				return nil, false, newTranslatorError("output periods of join subqueries don't match", expr.GetMetadata())
			}
			joinType, joinAttributes, outAttributes := getJoinTypeAndAttributes(leftQuery.outputLabels, rightQuery.outputLabels)
			return &promqlSingleExpressionQuery{
				expr: &promqlParenthesis{expr: &promqlJoinBinaryExpression{
					joinType:       joinType,
					joinAttributes: joinAttributes,
					operation:      typedExpr.Operation,
					leftExpr:       leftQuery.expr,
					rightExpr:      rightQuery.expr,
				}},
				outputPeriod: leftQuery.outputPeriod,
				outputLabels: outAttributes,
			}, false, nil
		case "min", "max":
			return nil, false, newTranslatorError("min and max functions in join expressions are not translatable to promql", expr.GetMetadata())
		default:
			return nil, false, newTranslatorError_invalid_expression(expr.GetMetadata())
		}
	case *ast.PrefixExpression:
		switch typedExpr.Operation {
		case string(model.TypeDiff):
			innerQuery, isScalar, err := tr.joinExprToPromqlQuery(typedExpr.Expr, values)
			if err != nil {
				return nil, false, err
			}
			return &promqlSingleExpressionQuery{
				expr: &promqlUnaryExpression{
					operation: typedExpr.Operation,
					expr:      &promqlParenthesis{expr: innerQuery.expr},
				},
				outputPeriod: innerQuery.outputPeriod,
				outputLabels: innerQuery.outputLabels,
			}, isScalar, nil
		case "abs", "timestamp", "floor", "ceil", "round":
			innerQuery, isScalar, err := tr.joinExprToPromqlQuery(typedExpr.Expr, values)
			if err != nil {
				return nil, false, err
			}
			return &promqlSingleExpressionQuery{
				expr: &promqlFunctionCall{
					fname: typedExpr.Operation,
					args:  []promqlExpression{innerQuery.expr},
				},
				outputPeriod: innerQuery.outputPeriod,
				outputLabels: innerQuery.outputLabels,
			}, isScalar, nil
		default:
			return nil, false, newTranslatorError_invalid_expression(expr.GetMetadata())
		}
	case *ast.Identifier:
		if value, ok := values[typedExpr.Value]; ok {
			if constant, isConstant := value.expr.(*promqlConstantVector); isConstant {
				return &promqlSingleExpressionQuery{
					expr: &promqlScalarValue{value: constant.value},
				}, true, nil
			}
			return value, false, nil
		}
		return nil, false, newTranslatorError_invalid_expression(expr.GetMetadata())
	case *ast.IntegerLiteral:
		return &promqlSingleExpressionQuery{
			expr: &promqlScalarValue{value: typedExpr.Value},
		}, true, nil
	case *ast.FloatLiteral:
		return &promqlSingleExpressionQuery{
			expr: &promqlScalarValue{value: typedExpr.Value},
		}, true, nil
	case *ast.TemplateVariable:
		return &promqlSingleExpressionQuery{
			expr: &promqlScalarValue{value: typedExpr.Value},
		}, true, nil
	default:
		return nil, false, newTranslatorError_invalid_expression(expr.GetMetadata())
	}
}

func (tr *Translator) translateUnnamedJoin(query *ast.Query) (*PromqlQuery, *model.Error) {
	var joinStage *ast.ModifierStageJoin = nil
	var commonStages []ast.Stage = nil
	var remainingStages []ast.Stage = nil

	for _, stage := range query.UnnamedJoin.Stages {
		if typedStage, ok := stage.(*ast.ModifierStageJoin); ok {
			if joinStage == nil {
				joinStage = typedStage
			} else {
				return nil, newTranslatorError("unnamed joins must only have one join stage, it must be the first in the chain", typedStage.Metadata)
			}
		} else {
			if joinStage == nil {
				commonStages = append(commonStages, stage)
			} else {
				remainingStages = append(remainingStages, stage)
			}
		}
	}

	if joinStage == nil {
		return nil, newTranslatorError("unnamed join must have a join stage", query.Metadata)
	}

	var leftQuery *ast.Query = query.UnnamedJoin.Left
	var rightQuery *ast.Query = query.UnnamedJoin.Right
	var err *model.Error = nil

	err = tr.applyCommonStages(leftQuery, commonStages)
	if err != nil {
		return nil, err
	}
	err = tr.applyCommonStages(rightQuery, commonStages)
	if err != nil {
		return nil, err
	}

	if joinStage.LeftDefault != nil {
		err = tr.applyDefaultValue(leftQuery, joinStage.LeftDefault)
		if err != nil {
			return nil, err
		}
	}
	if joinStage.RightDefault != nil {
		err = tr.applyDefaultValue(rightQuery, joinStage.RightDefault)
		if err != nil {
			return nil, err
		}
	}

	leftPromqlQuery, err := tr.translateQuery(leftQuery)
	if err != nil {
		return nil, err
	}
	rightPromqlQuery, err := tr.translateQuery(rightQuery)
	if err != nil {
		return nil, err
	}

	leftSingleExprQuery, err := tr.joinSubqueriesInPromqlQuery(leftPromqlQuery, leftQuery)
	if err != nil {
		return nil, err
	}
	rightSingleExprQuery, err := tr.joinSubqueriesInPromqlQuery(rightPromqlQuery, rightQuery)
	if err != nil {
		return nil, err
	}

	joinedQuery, _, err := tr.joinExprToPromqlQuery(joinStage.Expr, map[string]*promqlSingleExpressionQuery{
		"left":  leftSingleExprQuery,
		"right": rightSingleExprQuery,
	})
	if err != nil {
		return nil, err
	}

	outputQuery := &PromqlQuery{
		subqueries:   []promqlExpression{joinedQuery.expr},
		outputPeriod: joinedQuery.outputPeriod,
		outputLabels: joinedQuery.outputLabels,
	}

	err = tr.applyStages(outputQuery, remainingStages, METRIC_TYPE_GAUGE)
	if err != nil {
		return nil, err
	}

	return outputQuery, nil
}

func (tr *Translator) translateNamedJoin(query *ast.Query) (*PromqlQuery, *model.Error) {
	promqlQueries := map[string]*promqlSingleExpressionQuery{}

	for _, subquery := range query.NamedJoin.Queries {
		if subquery.Default != nil {
			err := tr.applyDefaultValue(subquery.Query, subquery.Default)
			if err != nil {
				return nil, err
			}
		}
		tmpPromqlQuery, err := tr.translateQuery(subquery.Query)
		if err != nil {
			return nil, err
		}
		tmpPromqlSingleExprQuery, err := tr.joinSubqueriesInPromqlQuery(tmpPromqlQuery, subquery.Query)
		if err != nil {
			return nil, err
		}
		promqlQueries[subquery.Name] = tmpPromqlSingleExprQuery
	}

	joinedQuery, _, err := tr.joinExprToPromqlQuery(query.NamedJoin.JoinExpr, promqlQueries)
	if err != nil {
		return nil, err
	}

	outputQuery := &PromqlQuery{
		subqueries:   []promqlExpression{joinedQuery.expr},
		outputPeriod: joinedQuery.outputPeriod,
		outputLabels: joinedQuery.outputLabels,
	}

	err = tr.applyStages(outputQuery, query.NamedJoin.Stages, METRIC_TYPE_GAUGE)
	if err != nil {
		return nil, err
	}

	return outputQuery, nil
}

func (tr *Translator) translateQuery(query *ast.Query) (*PromqlQuery, *model.Error) {
	switch query.Type {
	case ast.QueryTypeDefault:
		result, err := tr.translatePipeline(query)
		if err != nil {
			return nil, err
		}
		return result, nil
	case ast.QueryTypeUnnamedJoin:
		result, err := tr.translateUnnamedJoin(query)
		if err != nil {
			return nil, err
		}
		return result, nil
	case ast.QueryTypeNamedJoin:
		result, err := tr.translateNamedJoin(query)
		if err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, newTranslatorError("unknown query type", query.Metadata)
	}
}

func (tr *Translator) removeTopParenthesis(query *PromqlQuery) {
	for idx := range query.subqueries {
		for {
			typedExpr, isParenthesis := query.subqueries[idx].(*promqlParenthesis)
			if isParenthesis {
				query.subqueries[idx] = typedExpr.expr
			} else {
				break
			}
		}
	}
}

func (tr *Translator) Translate(query *ast.Query) (string, *model.Error) {
	result, err := tr.translateQuery(query)
	if err != nil {
		return "", err
	}
	tr.removeTopParenthesis(result)
	return result.String(), nil
}
