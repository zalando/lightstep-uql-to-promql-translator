package optimizer

import (
	"fmt"

	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model/ast"
)

const (
	MAX_DISJUNCTS_IN_FILTER_EXPRESSION = 128
)

func Filter_MergeStagesIntoSingleFilterStage(pipeline []ast.Stage) []ast.Stage {
	var singleFilterStage *ast.ModifierStageFilter = nil
	var stagesBefore []ast.Stage = nil
	var stagesAfter []ast.Stage = nil

	for _, stage := range pipeline {
		if filter, isFilter := stage.(*ast.ModifierStageFilter); isFilter {
			if singleFilterStage == nil {
				singleFilterStage = filter
			} else {
				singleFilterStage.Expr = &ast.InfixExpression{
					Operation: "&&",
					LeftExpr:  singleFilterStage.Expr,
					RightExpr: filter.Expr,
				}
			}
		} else {
			if singleFilterStage == nil {
				if _, isFetchStage := stage.(ast.FetchStage); isFetchStage {
					stagesBefore = append(stagesBefore, stage)
				} else {
					stagesAfter = append(stagesAfter, stage)
				}
			} else {
				stagesAfter = append(stagesAfter, stage)
			}
		}
	}

	var result []ast.Stage = stagesBefore
	if singleFilterStage != nil {
		result = append(result, singleFilterStage)
	}
	result = append(result, stagesAfter...)
	return result
}

func Filter_ConvertSingleAttributeConjunctionToRegexp(pipeline []ast.Stage) []ast.Stage {
	for idx := range pipeline {
		if stage, ok := pipeline[idx].(*ast.ModifierStageFilter); ok {
			expr, _, _, _ := convertSingleAttributeConjunctionToRegexp(stage.Expr)
			stage.Expr = expr
		}
	}
	return pipeline
}

func Filter_PushLogicalNegationsDownExpressionTree(pipeline []ast.Stage) ([]ast.Stage, *model.Error) {
	for idx := range pipeline {
		if stage, ok := pipeline[idx].(*ast.ModifierStageFilter); ok {
			expr, err := pushLogicalNegationsDownExpressionTree(stage.Expr, false)
			if err != nil {
				return nil, err
			}
			stage.Expr = expr
		}
	}
	return pipeline, nil
}

func Filter_ConvertExpressionToDisjunctiveNormalForm(pipeline []ast.Stage) ([]ast.Stage, *model.Error) {
	for idx := range pipeline {
		if stage, ok := pipeline[idx].(*ast.ModifierStageFilter); ok {
			exprs, err := convertExpressionToDisjunctiveNormalForm(stage.Expr, MAX_DISJUNCTS_IN_FILTER_EXPRESSION)
			if err != nil {
				return nil, err
			}
			stage.Expr = convertMultipleExpressionsToSingle(exprs)
		}
	}
	return pipeline, nil
}

func Filter_ConvertContainsOperationToRegexp(pipeline []ast.Stage) ([]ast.Stage, *model.Error) {
	for idx := range pipeline {
		if stage, ok := pipeline[idx].(*ast.ModifierStageFilter); ok {
			expr, err := convertContainsOperationToRegexp(stage.Expr)
			if err != nil {
				return nil, err
			}
			stage.Expr = expr
		}
	}
	return pipeline, nil
}

func Filter_ConvertPhraseMatchOperationToRegexp(pipeline []ast.Stage) ([]ast.Stage, *model.Error) {
	for idx := range pipeline {
		if stage, ok := pipeline[idx].(*ast.ModifierStageFilter); ok {
			expr, err := convertPhraseMatchOperationToRegexp(stage.Expr)
			if err != nil {
				return nil, err
			}
			stage.Expr = expr
		}
	}
	return pipeline, nil
}

func PointFilter_ConvertSingleAttributeConjunctionToRegexp(pipeline []ast.Stage) []ast.Stage {
	for idx := range pipeline {
		if stage, ok := pipeline[idx].(*ast.ModifierStagePointFilter); ok {
			expr, _, _, _ := convertSingleAttributeConjunctionToRegexp(stage.Expr)
			stage.Expr = expr
		}
	}
	return pipeline
}

func PointFilter_PushLogicalNegationsDownExpressionTree(pipeline []ast.Stage) ([]ast.Stage, *model.Error) {
	for idx := range pipeline {
		if stage, ok := pipeline[idx].(*ast.ModifierStagePointFilter); ok {
			expr, err := pushLogicalNegationsDownExpressionTree(stage.Expr, false)
			if err != nil {
				return nil, err
			}
			stage.Expr = expr
		}
	}
	return pipeline, nil
}

func PointFilter_ConvertExpressionToDisjunctiveNormalForm(pipeline []ast.Stage) ([]ast.Stage, *model.Error) {
	for idx := range pipeline {
		if stage, ok := pipeline[idx].(*ast.ModifierStagePointFilter); ok {
			exprs, err := convertExpressionToDisjunctiveNormalForm(stage.Expr, MAX_DISJUNCTS_IN_FILTER_EXPRESSION)
			if err != nil {
				return nil, err
			}
			stage.Expr = convertMultipleExpressionsToSingle(exprs)
		}
	}
	return pipeline, nil
}

func PointFilter_ConvertContainsOperationToRegexp(pipeline []ast.Stage) ([]ast.Stage, *model.Error) {
	for idx := range pipeline {
		if stage, ok := pipeline[idx].(*ast.ModifierStagePointFilter); ok {
			expr, err := convertContainsOperationToRegexp(stage.Expr)
			if err != nil {
				return nil, err
			}
			stage.Expr = expr
		}
	}
	return pipeline, nil
}

func ConvertTemplateVariablesToStrings(pipeline []ast.Stage) []ast.Stage {
	for idx := range pipeline {
		switch typedStage := pipeline[idx].(type) {
		case *ast.ModifierStageFilter:
			typedStage.Expr = convertTemplateVariablesToString(typedStage.Expr)
		}
	}
	return pipeline
}

func OptimizeQueryDefault(query *ast.Query, config OptimizerConfig) (*ast.Query, *model.Error) {
	// filter optimizations
	if config.Filter.ConvertTemplateVariablesToString {
		query.Pipeline = ConvertTemplateVariablesToStrings(query.Pipeline)
	}
	if config.Filter.MergeStagesIntoSingleStage {
		query.Pipeline = Filter_MergeStagesIntoSingleFilterStage(query.Pipeline)
	}
	if config.Filter.ConvertSingleAttributeConjunctionToRegexp {
		query.Pipeline = Filter_ConvertSingleAttributeConjunctionToRegexp(query.Pipeline)
	}
	if config.Filter.PushLogicalNegationsDownExpressionTree || config.Filter.ConvertExpressionToDisjunctiveNormalForm {
		pipeline, err := Filter_PushLogicalNegationsDownExpressionTree(query.Pipeline)
		if err != nil {
			return nil, err
		}
		query.Pipeline = pipeline
	}
	if config.Filter.ConvertExpressionToDisjunctiveNormalForm {
		pipeline, err := Filter_ConvertExpressionToDisjunctiveNormalForm(query.Pipeline)
		if err != nil {
			return nil, err
		}
		query.Pipeline = pipeline
	}
	if config.Filter.ConvertContainsOperationToRegexp {
		pipeline, err := Filter_ConvertContainsOperationToRegexp(query.Pipeline)
		if err != nil {
			return nil, err
		}
		query.Pipeline = pipeline
	}
	if config.Filter.ConvertPhraseMatchOperationToRegexp {
		pipeline, err := Filter_ConvertPhraseMatchOperationToRegexp(query.Pipeline)
		if err != nil {
			return nil, err
		}
		query.Pipeline = pipeline
	}

	// point_filter optimizations
	if config.PointFilter.ConvertSingleAttributeConjunctionToRegexp {
		query.Pipeline = PointFilter_ConvertSingleAttributeConjunctionToRegexp(query.Pipeline)
	}
	if config.PointFilter.PushLogicalNegationsDownExpressionTree || config.PointFilter.ConvertExpressionToDisjunctiveNormalForm {
		pipeline, err := PointFilter_PushLogicalNegationsDownExpressionTree(query.Pipeline)
		if err != nil {
			return nil, err
		}
		query.Pipeline = pipeline
	}
	if config.PointFilter.ConvertExpressionToDisjunctiveNormalForm {
		pipeline, err := PointFilter_ConvertExpressionToDisjunctiveNormalForm(query.Pipeline)
		if err != nil {
			return nil, err
		}
		query.Pipeline = pipeline
	}
	if config.PointFilter.ConvertContainsOperationToRegexp {
		pipeline, err := PointFilter_ConvertContainsOperationToRegexp(query.Pipeline)
		if err != nil {
			return nil, err
		}
		query.Pipeline = pipeline
	}

	// custom optimizations
	for _, optimizationFunc := range config.CustomOptimizations {
		pipeline, err := optimizationFunc(query.Pipeline)
		if err != nil {
			return nil, err
		}
		query.Pipeline = pipeline
	}

	return query, nil
}

func OptimizeQueryUnnamedJoin(query *ast.Query, config OptimizerConfig) (*ast.Query, *model.Error) {
	leftQuery, err := OptimizeQuery(query.UnnamedJoin.Left, config)
	if err != nil {
		return nil, err
	}
	rightQuery, err := OptimizeQuery(query.UnnamedJoin.Right, config)
	if err != nil {
		return nil, err
	}
	return &ast.Query{
		Type:      ast.QueryTypeUnnamedJoin,
		Metadata:  query.Metadata,
		Pipeline:  nil,
		NamedJoin: nil,
		UnnamedJoin: &ast.UnnamedJoin{
			Metadata: query.UnnamedJoin.Metadata,
			Stages:   query.UnnamedJoin.Stages,
			Left:     leftQuery,
			Right:    rightQuery,
		},
	}, nil
}

func OptimizeQueryNamedJoin(query *ast.Query, config OptimizerConfig) (*ast.Query, *model.Error) {
	result := make([]ast.NamedJoinPipeline, 0, len(query.NamedJoin.Queries))

	for _, subquery := range query.NamedJoin.Queries {
		newQuery, err := OptimizeQuery(subquery.Query, config)
		if err != nil {
			return nil, err
		}
		result = append(result, ast.NamedJoinPipeline{
			Name:    subquery.Name,
			Default: subquery.Default,
			Query:   newQuery,
		})
	}

	return &ast.Query{
		Type:        ast.QueryTypeNamedJoin,
		Metadata:    query.Metadata,
		Pipeline:    nil,
		UnnamedJoin: nil,
		NamedJoin: &ast.NamedJoin{
			Metadata: query.NamedJoin.Metadata,
			JoinExpr: query.NamedJoin.JoinExpr,
			Stages:   query.NamedJoin.Stages,
			Queries:  result,
		},
	}, nil
}

func OptimizeQuery(query *ast.Query, config OptimizerConfig) (*ast.Query, *model.Error) {
	switch query.Type {
	case ast.QueryTypeDefault:
		return OptimizeQueryDefault(query, config)
	case ast.QueryTypeUnnamedJoin:
		return OptimizeQueryUnnamedJoin(query, config)
	case ast.QueryTypeNamedJoin:
		return OptimizeQueryNamedJoin(query, config)
	default:
		return nil, newOptimizeError(fmt.Sprintf("unsupported query type: %s", query.Type), query.GetMetadata())
	}
}
