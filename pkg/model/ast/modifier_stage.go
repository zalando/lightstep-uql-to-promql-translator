package ast

import "encoding/xml"

type ModifierStageFill struct {
	XMLName xml.Name      `xml:"ModifierStageFill"`
	Number  NumberLiteral `xml:"Number>_"`
	Metadata
}

func (stage *ModifierStageFill) stage()         {}
func (stage *ModifierStageFill) modifierStage() {}

type ModifierStageFilter struct {
	XMLName xml.Name   `xml:"ModifierStageFilter"`
	Expr    Expression `xml:"Expr>_"`
	Metadata
}

func (stage *ModifierStageFilter) stage()         {}
func (stage *ModifierStageFilter) modifierStage() {}

type ModifierStageGroupBy struct {
	XMLName xml.Name     `xml:"ModifierStageGroupBy"`
	Labels  []Identifier `xml:"Labels>_"`
	Reducer string       `xml:"Reducer"`
	Metadata
}

func (stage *ModifierStageGroupBy) stage()         {}
func (stage *ModifierStageGroupBy) modifierStage() {}

type ModifierStageJoin struct {
	XMLName      xml.Name      `xml:"ModifierStageJoin"`
	Expr         Expression    `xml:"Expr>_"`
	LeftDefault  NumberLiteral `xml:"LeftDefault>_"`
	RightDefault NumberLiteral `xml:"RightDefault>_"`
	Metadata
}

func (stage *ModifierStageJoin) stage()         {}
func (stage *ModifierStageJoin) modifierStage() {}

type ModifierStageTop struct {
	XMLName xml.Name         `xml:"ModifierStageTop"`
	Labels  []Identifier     `xml:"Labels>_"`
	Amount  IntegerLiteral   `xml:"Amount>IntegerLiteral"`
	Reducer string           `xml:"Reducer"`
	Window  *DurationLiteral `xml:"Window>DurationLiteral"`
	Metadata
}

func (stage *ModifierStageTop) stage()         {}
func (stage *ModifierStageTop) modifierStage() {}

type ModifierStageBottom struct {
	XMLName xml.Name         `xml:"ModifierStageBottom"`
	Labels  []Identifier     `xml:"Labels>_"`
	Amount  IntegerLiteral   `xml:"Amount>IntegerLiteral"`
	Reducer string           `xml:"Reducer"`
	Window  *DurationLiteral `xml:"Window>DurationLiteral"`
	Metadata
}

func (stage *ModifierStageBottom) stage()         {}
func (stage *ModifierStageBottom) modifierStage() {}

type ModifierStagePoint struct {
	XMLName     xml.Name     `xml:"ModifierStagePoint"`
	Expressions []Expression `xml:"Expressions>_"`
	Metadata
}

func (stage *ModifierStagePoint) stage()         {}
func (stage *ModifierStagePoint) modifierStage() {}

type ModifierStagePointFilter struct {
	XMLName xml.Name   `xml:"ModifierStagePointFilter"`
	Expr    Expression `xml:"Expr>_"`
	Metadata
}

func (stage *ModifierStagePointFilter) stage()         {}
func (stage *ModifierStagePointFilter) modifierStage() {}

type ModifierStageTimeShift struct {
	XMLName       xml.Name        `xml:"ModifierStageTimeShift"`
	ShiftDuration DurationLiteral `xml:"ShiftDuration>DurationLiteral"`
	Metadata
}

func (stage *ModifierStageTimeShift) stage()         {}
func (stage *ModifierStageTimeShift) modifierStage() {}
