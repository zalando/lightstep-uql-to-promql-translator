package ast

import "encoding/xml"

type QueryType string

const (
	QueryTypeDefault     QueryType = "default"
	QueryTypeNamedJoin   QueryType = "named_join"
	QueryTypeUnnamedJoin QueryType = "unnamed_join"
)

type Query struct {
	XMLName     xml.Name     `xml:"Query"`
	Type        QueryType    `xml:"Type"`
	Pipeline    []Stage      `xml:"Pipeline>_"`
	NamedJoin   *NamedJoin   `xml:"NamedJoin"`
	UnnamedJoin *UnnamedJoin `xml:"UnnamedJoin"`
	Metadata
}

func (q *Query) ToXml() string {
	result, _ := xml.MarshalIndent(q, "", "  ")
	return string(result)
}

type UnnamedJoin struct {
	XMLName xml.Name `xml:"UnnamedJoin"`
	Left    *Query   `xml:"Left>Query"`
	Right   *Query   `xml:"Right>Query"`
	Stages  []Stage  `xml:"Stages>_"`
	Metadata
}

type NamedJoinPipeline struct {
	XMLName xml.Name      `xml:"NamedJoinPipeline"`
	Name    string        `xml:"Name"`
	Query   *Query        `xml:"Query"`
	Default NumberLiteral `xml:"Default>_"`
}

type NamedJoin struct {
	XMLName  xml.Name            `xml:"NamedJoin"`
	Queries  []NamedJoinPipeline `xml:"Queries>_"`
	JoinExpr Expression          `xml:"JoinExpr>_"`
	Stages   []Stage             `xml:"Stages>_"`
	Metadata
}

type Stage interface {
	stage()
	GetMetadata() Metadata
}

type FetchStage interface {
	stage()
	fetchStage()
	GetMetadata() Metadata
}

type AlignerStage interface {
	stage()
	alignerStage()
	GetMetadata() Metadata
}

type ModifierStage interface {
	stage()
	modifierStage()
	GetMetadata() Metadata
}
