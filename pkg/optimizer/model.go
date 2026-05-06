package optimizer

import (
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model/ast"
)

type OptimizationFunc func([]ast.Stage) ([]ast.Stage, *model.Error)

type OptimizerConfigFilter struct {
	MergeStagesIntoSingleStage                bool
	ConvertSingleAttributeConjunctionToRegexp bool
	PushLogicalNegationsDownExpressionTree    bool
	ConvertExpressionToDisjunctiveNormalForm  bool
	ConvertContainsOperationToRegexp          bool
	ConvertPhraseMatchOperationToRegexp       bool
}

type OptimizerConfigPointFilter struct {
	ConvertSingleAttributeConjunctionToRegexp bool
	PushLogicalNegationsDownExpressionTree    bool
	ConvertExpressionToDisjunctiveNormalForm  bool
	ConvertContainsOperationToRegexp          bool
}

type OptimizerConfig struct {
	Filter              OptimizerConfigFilter
	PointFilter         OptimizerConfigPointFilter
	CustomOptimizations []OptimizationFunc
}

func DefaultOptimizerConfig() OptimizerConfig {
	return OptimizerConfig{
		Filter: OptimizerConfigFilter{
			MergeStagesIntoSingleStage:                true,
			ConvertSingleAttributeConjunctionToRegexp: true,
			PushLogicalNegationsDownExpressionTree:    true,
			ConvertExpressionToDisjunctiveNormalForm:  true,
			ConvertContainsOperationToRegexp:          true,
			ConvertPhraseMatchOperationToRegexp:       true,
		},
		PointFilter: OptimizerConfigPointFilter{
			ConvertSingleAttributeConjunctionToRegexp: true,
			PushLogicalNegationsDownExpressionTree:    true,
			ConvertExpressionToDisjunctiveNormalForm:  true,
			ConvertContainsOperationToRegexp:          true,
		},
		CustomOptimizations: nil,
	}
}
