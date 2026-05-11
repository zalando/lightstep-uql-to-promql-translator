package lexer

import (
	"strconv"
	"strings"
)

func isDigit(char byte) bool {
	return '0' <= char && char <= '9'
}

func isLetter(char byte) bool {
	return isLetterLowercase(char) || isLetterUppercase(char)
}

func isLetterLowercase(char byte) bool {
	return 'a' <= char && char <= 'z'
}

func isLetterUppercase(char byte) bool {
	return 'A' <= char && char <= 'Z'
}

func isWhitespace(char byte) bool {
	return char == ' ' || char == '\n' || char == '\t' || char == '\r'
}

func isCommentChar(char byte) bool {
	return char == '#'
}

func isTemplateVariableChar(char byte) bool {
	return char == '$'
}

func isStringDeclarationChar(char byte) bool {
	return char == '\'' || char == '"' || char == '`'
}

func isBackslash(char byte) bool {
	return char == '\\'
}

func isUnderscore(char byte) bool {
	return char == '_'
}

func isNewLine(char byte) bool {
	return char == '\n'
}

func isDot(char byte) bool {
	return char == '.'
}

func isColon(char byte) bool {
	return char == ':'
}

func isSlash(char byte) bool {
	return char == '/'
}

func isDash(char byte) bool {
	return char == '-'
}

func isTemplateVariableBracket(char byte) bool {
	return char == '{' || char == '}'
}

func isLiteralChar(char byte) bool {
	return isLetter(char) || isDigit(char) || isDot(char) ||
		isUnderscore(char) || isColon(char) || isDash(char) || isSlash(char)
}

func isLiteralCharWithoutDashesAndSlashes(char byte) bool {
	return isLetter(char) || isDigit(char) || isDot(char) || isUnderscore(char) || isColon(char)
}

func isInteger(line string) bool {
	_, err := strconv.ParseInt(line, 10, 64)
	return err == nil
}

func isFloat(line string) bool {
	_, err := strconv.ParseFloat(line, 64)
	return err == nil
}

func isDuration(line string) bool {
	if len(line) < 2 {
		return false
	}
	var digitPart strings.Builder
	var letterPart strings.Builder
	for i := 0; i < len(line); i++ {
		if isDigit(line[i]) {
			digitPart.WriteByte(line[i])
		} else {
			letterPart.WriteByte(line[i])
		}
	}
	if !isInteger(digitPart.String()) {
		return false
	}
	switch letterPart.String() {
	case "ms", "s", "m", "h", "d", "w":
		return true
	default:
		return false
	}
}

func checkIfStartsLikeIdentidier(line string) bool {
	if len(line) == 0 {
		return false
	}
	return isLetter(line[0])
}

func isDisallowMathSymbolsKeyword(keyword string) bool {
	switch keyword {
	case "join":
		return true
	case "point":
		return true
	default:
		return false
	}
}
