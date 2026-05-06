package promql

import (
	"fmt"
	"testing"
)

func TestSampleQuery(t *testing.T) {
	query := `
	metric my_metric
	| filter (attr1=="1" || attr2=="2") && attr3=="3" && contains(attr4, "test")
	`

	result, err := Translate(query, map[string]MetricType{}, SpecialMetricConfig{})
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%s\n", result)
}

func TestOneOfQuery(t *testing.T) {
	query := `
	metric my_metric
	| filter a=="1" && b=="2" && (c=="3" || c=="4" || c=="5") && (d=="6")
	| delta
	`

	result, err := Translate(query, map[string]MetricType{}, SpecialMetricConfig{})
	if err != nil {
		t.Fatal(err)
	}

	expected := `delta({otel_metric_name="my_metric", a="1", b="2", c=~"^(3|4|5)$", d="6"}[$__rate_interval])`

	if result != expected {
		t.Errorf("wrong result: %s", result)
	}
}

func TestFetchAlignQuery(t *testing.T) {
	query := `
	fetch lightstep.hourly_active_time_series
	| delta 1h, 5m
	| align
	| group_by [lightstep.value_type], sum
	`

	result, err := Translate(query, map[string]MetricType{}, SpecialMetricConfig{})
	if err != nil {
		t.Fatal(err)
	}

	expected := `sum by (lightstep_value_type) (last_over_time(delta({otel_metric_name="lightstep.hourly_active_time_series"}[1h:5m])[$__interval:$__interval]))`

	if result != expected {
		t.Errorf("wrong output: %s", result)
	}
}

func TestUnnamedJoin(t *testing.T) {
	query := `
	(
		metric my_metric | filter attr=="a" | rate | group_by [a, b], sum;
		metric my_metric | rate | group_by [a], sum;
	)
	| join left / right + left
	`

	result, err := Translate(query, map[string]MetricType{}, SpecialMetricConfig{})
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%s\n", result)
}
