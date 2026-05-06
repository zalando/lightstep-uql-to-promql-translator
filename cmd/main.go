package main

import (
	"log"

	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/promql"
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/server"
)

func TranslateUQLToPromQL(query string) (string, *model.Error) {
	metricTypes := map[string]promql.MetricType{
		"http_requests_total": promql.METRIC_TYPE_SUM,
		"cpu_usage":           promql.METRIC_TYPE_GAUGE,
		"request_duration":    promql.METRIC_TYPE_HISTOGRAM,
	}

	metricConfig := promql.SpecialMetricConfig{
		SpansCount:           "spans.count",
		SpansLatency:         "spans.latency",
		SpansCountUnadjusted: "spans.count_unadjusted",
		LogsCount:            "logs.count",
	}

	return promql.Translate(query, metricTypes, metricConfig)
}

func main() {
	srv := server.New(":8080", TranslateUQLToPromQL)
	log.Fatal(srv.Start())
}
