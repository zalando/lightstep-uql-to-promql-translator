package ast

type Metadata struct {
	SourceIndex  int `xml:"-"`
	SourceLength int `xml:"-"`
}

func (m *Metadata) GetMetadata() Metadata {
	return *m
}
