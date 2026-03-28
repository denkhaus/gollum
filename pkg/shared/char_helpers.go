package shared

// IsIdentStart returns true if ch can start an identifier
func IsIdentStart(ch byte) bool {
	return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

// IsIdentPart returns true if ch can be part of an identifier
func IsIdentPart(ch byte) bool {
	return IsIdentStart(ch) || IsDigit(ch)
}

// IsDigit returns true if ch is a digit
func IsDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
