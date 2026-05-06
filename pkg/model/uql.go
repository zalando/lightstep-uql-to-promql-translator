package model

var Stages = []string{"logs", "metric", "spans", "constant", "fetch", "spans_sample", "assemble", "trace_filter", "summarize_by"}
var Aligners = []string{"delta", "rate", "latest", "align", "reduce"}
var Modifiers = []string{"fill", "filter", "group_by", "join", "top", "bottom", "point", "point_filter", "time_shift"}
var ArithmeticFunctions = []string{"pow", "percentile", "dist_sum", "dist_count", "abs", "timestamp", "floor", "ceil", "round"}
var FilterOperators = []string{"defined", "undefined", "contains", "phrase_match"}
var ReducerOperators = []string{"min", "mean", "max", "sum", "distribution", "count", "count_nonzero", "std_dev"}
var FetchTypes = []string{"count", "count_unadjusted", "latency", "lightstep.bytesize"}
var JoinOperators = []string{"with", "join"}

var UQLKeywords = buildKeywords(Stages, Aligners, Modifiers, ArithmeticFunctions,
	FilterOperators, ReducerOperators, FetchTypes, JoinOperators)

func buildKeywords(values ...[]string) map[string]any {
	result := make(map[string]any)
	for _, items := range values {
		for _, item := range items {
			result[item] = nil
		}
	}
	return result
}
