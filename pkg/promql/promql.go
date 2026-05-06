package promql

import "strings"

type promqlExpression interface {
	promqlExpression()
	String() string
}

type PromqlQuery struct {
	subqueries   []promqlExpression
	outputPeriod *string
	outputLabels []string
}

func (p *PromqlQuery) String() string {
	result := []string{}
	for _, q := range p.subqueries {
		result = append(result, q.String())
	}
	return strings.Join(result, " or ")
}

type promqlSingleExpressionQuery struct {
	expr         promqlExpression
	outputPeriod *string
	outputLabels []string
}

type promqlVectorSelector struct {
	name          string
	matchers      []promqlLabelMatcher
	rangeSelector *string
	resolution    *string
	offset        *string
}

func (p *promqlVectorSelector) promqlExpression() {}
func (p *promqlVectorSelector) String() string {
	metricNameMatcher := promqlLabelMatcher{
		operation: "=",
		key:       "otel_metric_name",
		value:     "\"" + p.name + "\"",
	}
	strMatchers := []string{metricNameMatcher.String()}
	for _, m := range p.matchers {
		strMatchers = append(strMatchers, m.String())
	}
	result := "{" + strings.Join(strMatchers, ", ") + "}"
	if p.rangeSelector != nil {
		result += "[" + *p.rangeSelector
		if p.resolution != nil {
			result += ":" + *p.resolution
		}
		result += "]"
	}
	if p.offset != nil {
		result += " offset " + *p.offset
	}
	return result
}

type promqlLabelMatcher struct {
	operation string
	key       string
	value     string
}

func (p *promqlLabelMatcher) promqlExpression() {}
func (p *promqlLabelMatcher) String() string {
	return p.key + p.operation + p.value
}

type promqlGroupByExpression struct {
	operation string
	expr      promqlExpression
	labels    []string
}

func (p *promqlGroupByExpression) promqlExpression() {}
func (p *promqlGroupByExpression) String() string {
	result := p.operation
	if len(p.labels) > 0 {
		result += " by (" + strings.Join(p.labels, ", ") + ") "
	}
	result += "(" + p.expr.String() + ")"
	return result
}

type promqlTopOrBottomExpression struct {
	operation  string
	mainExpr   promqlExpression
	reduceExpr promqlExpression
	labels     []string
	amount     string
}

func (p *promqlTopOrBottomExpression) promqlExpression() {}
func (p *promqlTopOrBottomExpression) String() string {
	result := p.mainExpr.String()
	result += " and "
	if len(p.labels) > 0 {
		result += "on (" + strings.Join(p.labels, ", ") + ") "
	}
	result += p.operation
	if len(p.labels) > 0 {
		result += " by (" + strings.Join(p.labels, ", ") + ") "
	}
	result += "(" + p.amount + ", " + p.reduceExpr.String() + ")"
	return result
}

type promqlFunctionCall struct {
	fname string
	args  []promqlExpression
}

func (p *promqlFunctionCall) promqlExpression() {}
func (p *promqlFunctionCall) String() string {
	result := p.fname + "("
	strArgs := []string{}
	for _, arg := range p.args {
		strArgs = append(strArgs, arg.String())
	}
	result += strings.Join(strArgs, ", ")
	result += ")"
	return result
}

type promqlUnaryExpression struct {
	operation string
	expr      promqlExpression
}

func (p *promqlUnaryExpression) promqlExpression() {}
func (p *promqlUnaryExpression) String() string {
	return p.operation + p.expr.String()
}

type promqlBinaryExpression struct {
	operation string
	leftExpr  promqlExpression
	rightExpr promqlExpression
}

func (p *promqlBinaryExpression) promqlExpression() {}
func (p *promqlBinaryExpression) String() string {
	result := p.leftExpr.String() + " " + p.operation + " " + p.rightExpr.String()
	return result
}

type promqlJoinType string

const (
	joinTypeDefault    promqlJoinType = "default"
	joinTypeGroupLeft  promqlJoinType = "group_left"
	joinTypeGroupRight promqlJoinType = "group_right"
)

type promqlJoinBinaryExpression struct {
	operation      string
	joinType       promqlJoinType
	joinAttributes []string
	leftExpr       promqlExpression
	rightExpr      promqlExpression
}

func (p *promqlJoinBinaryExpression) promqlExpression() {}
func (p *promqlJoinBinaryExpression) String() string {
	formattedAttrs := make([]string, 0, len(p.joinAttributes))
	for _, attr := range p.joinAttributes {
		formattedAttrs = append(formattedAttrs, promqlAttributeFormat(attr))
	}
	switch p.joinType {
	case joinTypeGroupRight:
		labels := "on(" + strings.Join(formattedAttrs, ", ") + ")"
		args := []string{p.leftExpr.String(), p.operation, labels, "group_right()", p.rightExpr.String()}
		result := strings.Join(args, " ")
		return result
	case joinTypeGroupLeft:
		labels := "on(" + strings.Join(formattedAttrs, ", ") + ")"
		args := []string{p.leftExpr.String(), p.operation, labels, "group_left()", p.rightExpr.String()}
		result := strings.Join(args, " ")
		return result
	default:
		args := []string{p.leftExpr.String(), p.operation, p.rightExpr.String()}
		result := strings.Join(args, " ")
		return result
	}
}

type promqlSubqueryExpression struct {
	expr          promqlExpression
	rangeSelector string
	resolution    *string
}

func (p *promqlSubqueryExpression) promqlExpression() {}
func (p *promqlSubqueryExpression) String() string {
	result := p.expr.String() + "[" + p.rangeSelector
	if p.resolution != nil {
		result += ":" + *p.resolution
	}
	result += "]"
	return result
}

type promqlScalarValue struct {
	value string
}

func (p *promqlScalarValue) promqlExpression() {}
func (p *promqlScalarValue) String() string {
	return p.value
}

type promqlParenthesis struct {
	expr promqlExpression
}

func (p *promqlParenthesis) promqlExpression() {}
func (p *promqlParenthesis) String() string {
	return "(" + p.expr.String() + ")"
}

type promqlConstantVector struct {
	value string
}

func (p *promqlConstantVector) promqlExpression() {}
func (p *promqlConstantVector) String() string {
	return "vector(" + p.value + ")"
}
