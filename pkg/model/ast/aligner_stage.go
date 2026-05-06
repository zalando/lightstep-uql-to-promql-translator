package ast

import "encoding/xml"

type AlignerStageDelta struct {
	XMLName      xml.Name         `xml:"AlignerStageDelta"`
	InputWindow  *DurationLiteral `xml:"InputWindow>DurationLiteral"`
	OutputPeriod *DurationLiteral `xml:"OutputPeriod>DurationLiteral"`
	Metadata
}

func (stage *AlignerStageDelta) stage()        {}
func (stage *AlignerStageDelta) alignerStage() {}

type AlignerStageRate struct {
	XMLName      xml.Name         `xml:"AlignerStageRate"`
	InputWindow  *DurationLiteral `xml:"InputWindow>DurationLiteral"`
	OutputPeriod *DurationLiteral `xml:"OutputPeriod>DurationLiteral"`
	Metadata
}

func (stage *AlignerStageRate) stage()        {}
func (stage *AlignerStageRate) alignerStage() {}

type AlignerStageLatest struct {
	XMLName      xml.Name         `xml:"AlignerStageLatest"`
	InputWindow  *DurationLiteral `xml:"InputWindow>DurationLiteral"`
	OutputPeriod *DurationLiteral `xml:"OutputPeriod>DurationLiteral"`
	Metadata
}

func (stage *AlignerStageLatest) stage()        {}
func (stage *AlignerStageLatest) alignerStage() {}

type AlignerStageReduce struct {
	XMLName      xml.Name         `xml:"AlignerStageReduce"`
	InputWindow  *DurationLiteral `xml:"InputWindow>DurationLiteral"`
	OutputPeriod *DurationLiteral `xml:"OutputPeriod>DurationLiteral"`
	Reducer      string           `xml:"Reducer"`
	Metadata
}

func (stage *AlignerStageReduce) stage()        {}
func (stage *AlignerStageReduce) alignerStage() {}
