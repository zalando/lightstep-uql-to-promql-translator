package promql

import (
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/optimizer"
)

type MetricType string

const (
	METRIC_TYPE_SUM       MetricType = "sum"
	METRIC_TYPE_GAUGE     MetricType = "gauge"
	METRIC_TYPE_HISTOGRAM MetricType = "histogram"
)

type SpecialMetricConfig struct {
	SpansCount           string
	SpansCountUnadjusted string
	SpansLatency         string
	LogsCount            string
}

func Translate(
	query string,
	metricNameToType map[string]MetricType,
	metricConfig SpecialMetricConfig,
) (string, *model.Error) {
	queryAst, err := optimizer.Optimize(query)
	if err != nil {
		return "", err
	}

	translatorInstance := New(metricNameToType, metricConfig)
	result, err := translatorInstance.Translate(queryAst)
	if err != nil {
		return "", err
	}

	return result, nil
}
