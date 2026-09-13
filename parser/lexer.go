package parser

import (
	"fmt"
	"strings"
)

// Lexer turns raw SQL-like text into a stream of Tokens.
type Lexer struct {
	input string
	pos   int // index of the next unread byte
}

func NewLexer(input string) *Lexer {
	return &Lexer{input: input}
}

func (l *Lexer) peekByte() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) peekByteAt(offset int) byte {
	if l.pos+offset >= len(l.input) {
		return 0
	}
	return l.input[l.pos+offset]
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		c := l.input[l.pos]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			l.pos++
			continue
		}
		break
	}
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isLetter(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
func isIdentByte(c byte) bool { return isLetter(c) || isDigit(c) }

// Next returns the next token in the stream, or an EOF token when the
// input is exhausted.
func (l *Lexer) Next() (Token, error) {
	l.skipWhitespace()
	start := l.pos
	if l.pos >= len(l.input) {
		return Token{Type: EOF, Pos: start}, nil
	}

	c := l.peekByte()

	switch {
	case isLetter(c):
		for l.pos < len(l.input) && isIdentByte(l.input[l.pos]) {
			l.pos++
		}
		word := l.input[start:l.pos]
		upper := strings.ToUpper(word)
		if tt, ok := keywords[upper]; ok {
			return Token{Type: tt, Literal: upper, Pos: start}, nil
		}
		return Token{Type: IDENT, Literal: word, Pos: start}, nil

	case isDigit(c):
		for l.pos < len(l.input) && (isDigit(l.input[l.pos]) || l.input[l.pos] == '.') {
			l.pos++
		}
		return Token{Type: NUMBER, Literal: l.input[start:l.pos], Pos: start}, nil

	case c == '\'' || c == '"':
		quote := c
		l.pos++ // consume opening quote
		var sb strings.Builder
		for {
			if l.pos >= len(l.input) {
				return Token{}, fmt.Errorf("unterminated string literal starting at position %d", start)
			}
			cc := l.input[l.pos]
			if cc == quote {
				l.pos++
				break
			}
			sb.WriteByte(cc)
			l.pos++
		}
		return Token{Type: STRING, Literal: sb.String(), Pos: start}, nil

	case c == '(':
		l.pos++
		return Token{Type: LPAREN, Literal: "(", Pos: start}, nil
	case c == ')':
		l.pos++
		return Token{Type: RPAREN, Literal: ")", Pos: start}, nil
	case c == ',':
		l.pos++
		return Token{Type: COMMA, Literal: ",", Pos: start}, nil
	case c == ';':
		l.pos++
		return Token{Type: SEMICOLON, Literal: ";", Pos: start}, nil
	case c == '*':
		l.pos++
		return Token{Type: STAR, Literal: "*", Pos: start}, nil
	case c == '=':
		l.pos++
		return Token{Type: EQ, Literal: "=", Pos: start}, nil
	case c == '!' && l.peekByteAt(1) == '=':
		l.pos += 2
		return Token{Type: NEQ, Literal: "!=", Pos: start}, nil
	case c == '<' && l.peekByteAt(1) == '>':
		l.pos += 2
		return Token{Type: NEQ, Literal: "<>", Pos: start}, nil
	case c == '<' && l.peekByteAt(1) == '=':
		l.pos += 2
		return Token{Type: LE, Literal: "<=", Pos: start}, nil
	case c == '<':
		l.pos++
		return Token{Type: LT, Literal: "<", Pos: start}, nil
	case c == '>' && l.peekByteAt(1) == '=':
		l.pos += 2
		return Token{Type: GE, Literal: ">=", Pos: start}, nil
	case c == '>':
		l.pos++
		return Token{Type: GT, Literal: ">", Pos: start}, nil
	}

	return Token{}, fmt.Errorf("unexpected character %q at position %d", c, start)
}

// Tokenize consumes the entire input and returns every token,
// including a trailing EOF.
func Tokenize(input string) ([]Token, error) {
	l := NewLexer(input)
	var tokens []Token
	for {
		tok, err := l.Next()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			return tokens, nil
		}
	}
}
