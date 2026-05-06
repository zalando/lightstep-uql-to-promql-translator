package ast

import "encoding/xml"

type LogsFetchType string

const LogsFetchTypeCount LogsFetchType = "count"

type FetchStageLogs struct {
	XMLName   xml.Name      `xml:"FetchStageLogs"`
	FetchType LogsFetchType `xml:"FetchType"`
	Metadata
}

func (stage *FetchStageLogs) stage()      {}
func (stage *FetchStageLogs) fetchStage() {}

type FetchStageMetric struct {
	XMLName    xml.Name `xml:"FetchStageMetric"`
	MetricName string   `xml:"MetricName"`
	Metadata
}

func (stage *FetchStageMetric) stage()      {}
func (stage *FetchStageMetric) fetchStage() {}

type SpansFetchType string

type FetchStageSpans struct {
	XMLName   xml.Name `xml:"FetchStageSpans"`
	FetchType string   `xml:"FetchType"`
	Metadata
}

func (stage *FetchStageSpans) stage()      {}
func (stage *FetchStageSpans) fetchStage() {}

type FetchStageConstant struct {
	XMLName xml.Name `xml:"FetchStageConstant"`
	Value   Literal  `xml:"Value>_"`
	Metadata
}

func (stage *FetchStageConstant) stage()      {}
func (stage *FetchStageConstant) fetchStage() {}
