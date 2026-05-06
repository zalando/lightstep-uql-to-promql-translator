package streams

import "github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"

const (
	LEXER_MAX_ITERATIONS = 1024
)

type Lexer struct {
	input string
	index int
}

func NewLexer(input string) *Lexer {
	return &Lexer{
		input: input,
		index: 0,
	}
}

func (l *Lexer) currentChar() (byte, bool) {
	if l.index >= len(l.input) {
		return 0, true
	}
	return l.input[l.index], false
}

func (l *Lexer) move() {
	l.index++
}

func (l *Lexer) readIdentifier() (string, *model.Error) {
	result := ""

	for range LEXER_MAX_ITERATIONS {
		char, isEof := l.currentChar()
		if isEof || isWhitespace(char) {
			return result, nil
		}
		result += string(char)
		l.move()
	}

	return "", model.NewError("too many iterations needed to parse query", l.index, 0)
}

func (l *Lexer) readString() (string, *model.Error) {
	l.move()
	result := ""

	for range LEXER_MAX_ITERATIONS {
		char, isEof := l.currentChar()
		if isEof {
			return "", model.NewError("unexpected EOF", l.index, 0)
		}
		l.move()
		if char == '"' {
			nextChar, isEof := l.currentChar()
			if isEof {
				return result, nil
			} else {
				if nextChar == ',' || isWhitespace(nextChar) || nextChar == ')' {
					return result, nil
				}
			}
		}
		result += string(char)
	}

	return "", model.NewError("too many iterations needed to parse query", l.index, 0)
}

func (l *Lexer) Tokenize() ([]model.Token, *model.Error) {
	if len(l.input) <= 0 {
		return nil, model.NewError("empty input query", 0, 0)
	}

	var result []model.Token = nil

	for range LEXER_MAX_ITERATIONS {
		char, isEof := l.currentChar()
		if isEof {
			return result, nil
		}
		switch {
		case isWhitespace(char):
			l.move()
		case isLetter(char):
			startIndex := l.index
			token, err := l.readIdentifier()
			if err != nil {
				return nil, err
			}
			result = append(result, model.Token{Type: model.TypeString, Value: token, Index: startIndex, Length: len(token)})
		case char == '"':
			startIndex := l.index
			token, err := l.readString()
			if err != nil {
				return nil, err
			}
			result = append(result, model.Token{Type: model.TypeString, Value: token, Index: startIndex, Length: len(token) + 2})
		case char == '(':
			result = append(result, model.Token{Type: model.TypeRoundBracketLeft, Value: "(", Index: l.index, Length: 1})
			l.move()
		case char == ')':
			result = append(result, model.Token{Type: model.TypeRoundBracketRight, Value: ")", Index: l.index, Length: 1})
			l.move()
		case char == ',':
			l.move()
		default:
			return nil, model.NewError("unexpected char", l.index, 1)
		}
	}

	return nil, model.NewError("too many iterations needed to parse query", l.index, 0)
}
