package optimizer

import (
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model/ast"
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/parser"
)

func Optimize(query string) (*ast.Query, *model.Error) {
	queryAst, err := parser.Parse(query)
	if err != nil {
		return nil, err
	}
	return OptimizeQuery(queryAst, DefaultOptimizerConfig())
}
