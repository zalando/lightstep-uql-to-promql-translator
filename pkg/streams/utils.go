package streams

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
