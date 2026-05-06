package lexer

import (
	"testing"

	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
)

func TestSimpleQuery(t *testing.T) {
	query := `spans count
	| delta
	# my comment
	| filter operation == "my_operation" && service == $my_variable && my_float_attr >= 2.5 && my_another_attr == 0.0.2
	| reduce 1m, 1m
	| group_by ["my_attribute"], sum^
	| top 10`

	tokens, err := TokenizeWithComments(query)
	if err != nil {
		t.Fatalf("lexer error: %s", err.Error())
	}

	expected := []model.Token{
		{Type: model.TypeKeyword, Value: "spans", Length: 5},
		{Type: model.TypeKeyword, Value: "count", Length: 5},
		{Type: model.TypeSeparator, Value: "|", Length: 1},
		{Type: model.TypeKeyword, Value: "delta", Length: 5},
		{Type: model.TypeComment, Value: "# my comment", Length: 12},
		{Type: model.TypeSeparator, Value: "|", Length: 1},
		{Type: model.TypeKeyword, Value: "filter", Length: 6},
		{Type: model.TypeIdentifier, Value: "operation", Length: 9},
		{Type: model.TypeEquals, Value: "==", Length: 2},
		{Type: model.TypeString, Value: "my_operation", Length: 14},
		{Type: model.TypeLogicalAnd, Value: "&&", Length: 2},
		{Type: model.TypeIdentifier, Value: "service", Length: 7},
		{Type: model.TypeEquals, Value: "==", Length: 2},
		{Type: model.TypeTemplateVariable, Value: "$my_variable", Length: 12},
		{Type: model.TypeLogicalAnd, Value: "&&", Length: 2},
		{Type: model.TypeIdentifier, Value: "my_float_attr", Length: 13},
		{Type: model.TypeMoreOrEquals, Value: ">=", Length: 2},
		{Type: model.TypeFloat, Value: "2.5", Length: 3},
		{Type: model.TypeLogicalAnd, Value: "&&", Length: 2},
		{Type: model.TypeIdentifier, Value: "my_another_attr", Length: 15},
		{Type: model.TypeEquals, Value: "==", Length: 2},
		{Type: model.TypeString, Value: "0.0.2", Length: 5},
		{Type: model.TypeSeparator, Value: "|", Length: 1},
		{Type: model.TypeKeyword, Value: "reduce", Length: 6},
		{Type: model.TypeDuration, Value: "1m", Length: 2},
		{Type: model.TypeComma, Value: ",", Length: 1},
		{Type: model.TypeDuration, Value: "1m", Length: 2},
		{Type: model.TypeSeparator, Value: "|", Length: 1},
		{Type: model.TypeKeyword, Value: "group_by", Length: 8},
		{Type: model.TypeSquareBracketLeft, Value: "[", Length: 1},
		{Type: model.TypeString, Value: "my_attribute", Length: 14},
		{Type: model.TypeSquareBracketRight, Value: "]", Length: 1},
		{Type: model.TypeComma, Value: ",", Length: 1},
		{Type: model.TypeKeyword, Value: "sum", Length: 3},
		{Type: model.TypeUnknown, Value: "^", Length: 1},
		{Type: model.TypeSeparator, Value: "|", Length: 1},
		{Type: model.TypeKeyword, Value: "top", Length: 3},
		{Type: model.TypeInteger, Value: "10", Length: 2},
	}

	for idx, token := range tokens {
		if idx >= len(expected) {
			t.Errorf("tokens length does not match expected length")
			break
		}
		if token.Type != expected[idx].Type {
			t.Errorf("types don't match: %s and %s at %d", token.Type, expected[idx].Type, idx)
		}
		if token.Value != expected[idx].Value {
			t.Errorf("values don't match: %s and %s at %d", token.Value, expected[idx].Value, idx)
		}
		if token.Length != expected[idx].Length {
			t.Errorf("lengths don't match: %d and %d at %d", token.Length, expected[idx].Length, idx)
		}
	}
}
