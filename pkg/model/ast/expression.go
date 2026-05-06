package ast

import (
	"encoding/xml"
)

type Expression interface {
	expression()
	GetMetadata() Metadata
}

type PrefixExpression struct {
	XMLName   xml.Name   `xml:"PrefixExpression"`
	Operation string     `xml:"Operation"`
	Expr      Expression `xml:"Expr>_"`
	Metadata
}

func (_ *PrefixExpression) expression() {}

type InfixExpression struct {
	XMLName   xml.Name   `xml:"InfixExpression"`
	Operation string     `xml:"Operation"`
	LeftExpr  Expression `xml:"LeftExpr>_"`
	RightExpr Expression `xml:"RightExpr>_"`
	Metadata
}

func (_ *InfixExpression) expression() {}

type Identifier struct {
	XMLName xml.Name `xml:"Identifier"`
	Value   string   `xml:"Value"`
	Metadata
}

func (_ *Identifier) expression() {}

type TemplateVariable struct {
	XMLName xml.Name `xml:"TemplateVariable"`
	Value   string   `xml:"Value"`
	Metadata
}

func (_ *TemplateVariable) expression() {}

type Literal interface {
	literal()
	expression()
	String() string
	GetMetadata() Metadata
}

type NumberLiteral interface {
	literal()
	numberLiteral()
	expression()
	String() string
	GetMetadata() Metadata
}

type StringLiteral struct {
	XMLName xml.Name `xml:"StringLiteral"`
	Value   string   `xml:"Value"`
	Metadata
}

func (_ *StringLiteral) literal()       {}
func (_ *StringLiteral) expression()    {}
func (x *StringLiteral) String() string { return x.Value }

type IntegerLiteral struct {
	XMLName xml.Name `xml:"IntegerLiteral"`
	Value   string   `xml:"Value"`
	Metadata
}

func (_ *IntegerLiteral) literal()       {}
func (_ *IntegerLiteral) numberLiteral() {}
func (_ *IntegerLiteral) expression()    {}
func (x *IntegerLiteral) String() string { return x.Value }

type FloatLiteral struct {
	XMLName xml.Name `xml:"FloatLiteral"`
	Value   string   `xml:"Value"`
	Metadata
}

func (_ *FloatLiteral) literal()       {}
func (_ *FloatLiteral) numberLiteral() {}
func (_ *FloatLiteral) expression()    {}
func (x *FloatLiteral) String() string { return x.Value }

type BooleanLiteral struct {
	XMLName xml.Name `xml:"BooleanLiteral"`
	Value   string   `xml:"Value"`
	Metadata
}

func (_ *BooleanLiteral) literal()       {}
func (_ *BooleanLiteral) expression()    {}
func (x *BooleanLiteral) String() string { return x.Value }

type DurationLiteral struct {
	XMLName xml.Name `xml:"DurationLiteral"`
	Value   string   `xml:"Value"`
	Metadata
}

func (_ *DurationLiteral) literal()       {}
func (_ *DurationLiteral) expression()    {}
func (x *DurationLiteral) String() string { return x.Value }
