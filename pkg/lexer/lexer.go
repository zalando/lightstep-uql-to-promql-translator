package lexer

import (
	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
)

const LEXER_MAX_ITERATIONS = 1_000_000

type Lexer struct {
	input                        string
	index                        int
	allowMathSymbolsInIdentifier bool
}

func New(input string) *Lexer {
	return &Lexer{
		input:                        input,
		index:                        0,
		allowMathSymbolsInIdentifier: true,
	}
}

func (lexer *Lexer) isCurrentCharEof() bool {
	return lexer.index >= len(lexer.input)
}

func (lexer *Lexer) currentChar() (byte, bool) {
	if lexer.isCurrentCharEof() {
		return 0, true
	}
	return lexer.input[lexer.index], false
}

func (lexer *Lexer) move() {
	if lexer.isCurrentCharEof() {
		return
	}
	lexer.index += 1
}

func (lexer *Lexer) skipWhitespace() {
	for range LEXER_MAX_ITERATIONS {
		char, eof := lexer.currentChar()
		if eof {
			break
		}
		if isWhitespace(char) {
			lexer.move()
		} else {
			break
		}
	}
}

func (lexer *Lexer) readStringDeclaration() (string, *model.Error) {
	var result []byte
	var isPreviousCharBackslash bool = false

	stringDeclarationChar, _ := lexer.currentChar()
	startIndex := lexer.index
	lexer.move()

	for range LEXER_MAX_ITERATIONS {
		char, isEof := lexer.currentChar()
		if isEof {
			return "", model.NewError("EOF reached while parsing string declaration", startIndex, lexer.index-startIndex)
		}
		if char == stringDeclarationChar && !isPreviousCharBackslash {
			lexer.move()
			break
		}
		if isBackslash(char) {
			isPreviousCharBackslash = true
		} else if isPreviousCharBackslash && char != '"' {
			isPreviousCharBackslash = false
			result = append(result, '\\', char)
		} else {
			isPreviousCharBackslash = false
			result = append(result, char)
		}
		lexer.move()
	}

	return string(result), nil
}

func (lexer *Lexer) readTemplateVariable() (string, *model.Error) {
	startIndex := lexer.index
	lexer.move()
	var result []byte = []byte{'$'}
	for range LEXER_MAX_ITERATIONS {
		char, isEof := lexer.currentChar()
		if isEof {
			return "", model.NewError("EOF reached while parsing template variable", startIndex, lexer.index-startIndex)
		}
		if !isLetter(char) && !isDigit(char) && !isUnderscore(char) && !isTemplateVariableBracket(char) {
			break
		}
		result = append(result, char)
		lexer.move()
	}
	return string(result), nil
}

func (lexer *Lexer) readComment() (string, *model.Error) {
	var result []byte
	for range LEXER_MAX_ITERATIONS {
		char, isEof := lexer.currentChar()
		if isEof {
			break
		}
		if isNewLine(char) {
			break
		}
		result = append(result, char)
		lexer.move()
	}
	return string(result), nil
}

func (lexer *Lexer) readLiteral() (string, *model.Error) {
	var result []byte
	for range LEXER_MAX_ITERATIONS {
		char, isEof := lexer.currentChar()
		if isEof {
			break
		}
		if lexer.allowMathSymbolsInIdentifier {
			if !isLiteralChar(char) {
				break
			}
		} else {
			if !isLiteralCharWithoutDashesAndSlashes(char) {
				break
			}
		}
		result = append(result, char)
		lexer.move()
	}
	return string(result), nil
}

func (lexer *Lexer) readLiteralWithoutDashesAndSlashes() (string, *model.Error) {
	var result []byte
	for range LEXER_MAX_ITERATIONS {
		char, isEof := lexer.currentChar()
		if isEof {
			break
		}
		if !isLiteralCharWithoutDashesAndSlashes(char) {
			break
		}
		result = append(result, char)
		lexer.move()
	}
	return string(result), nil
}

func (lexer *Lexer) newToken(tokenType model.TokenType, value string, index int, length int) *model.Token {
	return &model.Token{Type: tokenType, Value: value, Index: index, Length: length}
}

func (lexer *Lexer) newTokenAutoDerive(tokenType model.TokenType, value string) *model.Token {
	return lexer.newToken(tokenType, value, lexer.index-len(value), len(value))
}

func (lexer *Lexer) FetchNextToken() (*model.Token, *model.Error) {
	lexer.skipWhitespace()
	char, isEof := lexer.currentChar()
	if isEof {
		return lexer.newTokenAutoDerive(model.TypeEOF, ""), nil
	}
	switch {
	case char == '*':
		lexer.move()
		return lexer.newTokenAutoDerive(model.TypeMul, string(char)), nil
	case char == '/':
		lexer.move()
		return lexer.newTokenAutoDerive(model.TypeDiv, string(char)), nil
	case char == '+':
		lexer.move()
		return lexer.newTokenAutoDerive(model.TypeAdd, string(char)), nil
	case char == '-':
		lexer.move()
		return lexer.newTokenAutoDerive(model.TypeDiff, string(char)), nil
	case char == '(':
		lexer.move()
		return lexer.newTokenAutoDerive(model.TypeRoundBracketLeft, string(char)), nil
	case char == ')':
		lexer.move()
		return lexer.newTokenAutoDerive(model.TypeRoundBracketRight, string(char)), nil
	case char == '[':
		lexer.move()
		return lexer.newTokenAutoDerive(model.TypeSquareBracketLeft, string(char)), nil
	case char == ']':
		lexer.move()
		return lexer.newTokenAutoDerive(model.TypeSquareBracketRight, string(char)), nil
	case char == ';':
		lexer.move()
		return lexer.newTokenAutoDerive(model.TypeSemicolon, string(char)), nil
	case char == ',':
		lexer.move()
		return lexer.newTokenAutoDerive(model.TypeComma, string(char)), nil
	case char == '&':
		lexer.move()
		nextChar, isNextEof := lexer.currentChar()
		if !isNextEof && nextChar == '&' {
			lexer.move()
			return lexer.newTokenAutoDerive(model.TypeLogicalAnd, "&&"), nil
		}
		return lexer.newTokenAutoDerive(model.TypeUnknown, string(char)), nil
	case char == '!':
		lexer.move()
		nextChar, isNextEof := lexer.currentChar()
		if !isNextEof {
			if nextChar == '~' {
				lexer.move()
				return lexer.newTokenAutoDerive(model.TypeNotMachRegex, "!~"), nil
			}
			if nextChar == '=' {
				lexer.move()
				return lexer.newTokenAutoDerive(model.TypeNotEquals, "!="), nil
			}
		}
		return lexer.newTokenAutoDerive(model.TypeLogicalNot, string(char)), nil
	case char == '|':
		lexer.move()
		nextChar, isNextEof := lexer.currentChar()
		if !isNextEof && nextChar == '|' {
			lexer.move()
			return lexer.newTokenAutoDerive(model.TypeLogicalOr, "||"), nil
		}
		return lexer.newTokenAutoDerive(model.TypeSeparator, string(char)), nil
	case char == '=':
		lexer.move()
		nextChar, isNextEof := lexer.currentChar()
		if !isNextEof {
			if nextChar == '=' {
				lexer.move()
				return lexer.newTokenAutoDerive(model.TypeEquals, "=="), nil
			}
			if nextChar == '~' {
				lexer.move()
				return lexer.newTokenAutoDerive(model.TypeMatchRegex, "=~"), nil
			}
		}
		return lexer.newTokenAutoDerive(model.TypeAssign, string(char)), nil
	case char == '<':
		lexer.move()
		nextChar, isNextEof := lexer.currentChar()
		if !isNextEof && nextChar == '=' {
			lexer.move()
			return lexer.newTokenAutoDerive(model.TypeLessOrEquals, "<="), nil
		}
		return lexer.newTokenAutoDerive(model.TypeLess, string(char)), nil
	case char == '>':
		lexer.move()
		nextChar, isNextEof := lexer.currentChar()
		if !isNextEof && nextChar == '=' {
			lexer.move()
			return lexer.newTokenAutoDerive(model.TypeMoreOrEquals, ">="), nil
		}
		return lexer.newTokenAutoDerive(model.TypeMore, string(char)), nil
	case isCommentChar(char):
		currentIndex := lexer.index
		value, err := lexer.readComment()
		if err != nil {
			return nil, err
		}
		return lexer.newToken(model.TypeComment, value, currentIndex, len(value)), nil
	case isTemplateVariableChar(char):
		value, err := lexer.readTemplateVariable()
		if err != nil {
			return nil, err
		}
		return lexer.newTokenAutoDerive(model.TypeTemplateVariable, value), nil
	case isStringDeclarationChar(char):
		startIndex := lexer.index
		value, err := lexer.readStringDeclaration()
		if err != nil {
			return nil, err
		}
		endIndex := lexer.index
		return lexer.newToken(model.TypeString, value, startIndex, endIndex-startIndex), nil
	case isDigit(char):
		startIndex := lexer.index
		value, err := lexer.readLiteralWithoutDashesAndSlashes()
		if err != nil {
			return nil, err
		}
		endIndex := lexer.index
		if isInteger(value) {
			return lexer.newToken(model.TypeInteger, value, startIndex, endIndex-startIndex), nil
		}
		if isFloat(value) {
			return lexer.newToken(model.TypeFloat, value, startIndex, endIndex-startIndex), nil
		}
		if isDuration(value) {
			return lexer.newToken(model.TypeDuration, value, startIndex, endIndex-startIndex), nil
		}
		lexer.index = startIndex
		fallthrough
	case isDigit(char) || isLetter(char):
		startIndex := lexer.index
		value, err := lexer.readLiteral()
		if err != nil {
			return nil, err
		}
		endIndex := lexer.index
		if _, isKeyword := model.UQLKeywords[value]; isKeyword {
			if isDisallowMathSymbolsKeyword(value) {
				lexer.allowMathSymbolsInIdentifier = false
			} else {
				lexer.allowMathSymbolsInIdentifier = true
			}
			return lexer.newToken(model.TypeKeyword, value, startIndex, endIndex-startIndex), nil
		}
		if !checkIfStartsLikeIdentidier(value) {
			return lexer.newToken(model.TypeString, value, startIndex, endIndex-startIndex), nil
		}
		return lexer.newToken(model.TypeIdentifier, value, startIndex, endIndex-startIndex), nil
	default:
		lexer.move()
		return lexer.newTokenAutoDerive(model.TypeUnknown, string(char)), nil
	}
}

func (lexer *Lexer) FetchAllTokens() ([]model.Token, *model.Error) {
	var result []model.Token
	for range LEXER_MAX_ITERATIONS {
		token, err := lexer.FetchNextToken()
		if err != nil {
			return result, err
		}
		if token.Type == model.TypeEOF {
			break
		}
		result = append(result, *token)
	}
	return result, nil
}

func (lexer *Lexer) FetchAllTokensWithoutComments() ([]model.Token, *model.Error) {
	var result []model.Token
	for range LEXER_MAX_ITERATIONS {
		token, err := lexer.FetchNextToken()
		if err != nil {
			return result, err
		}
		if token.Type == model.TypeEOF {
			break
		}
		if token.Type == model.TypeComment {
			continue
		}
		result = append(result, *token)
	}
	return result, nil
}
