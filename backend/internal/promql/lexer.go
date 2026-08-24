package promql

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Lexer struct {
	src  string
	pos  int
	line int
	col  int
}

func NewLexer(src string) *Lexer {
	return &Lexer{src: src, line: 1, col: 1}
}

func (l *Lexer) Next() Token {
	l.skipWS()
	if l.pos >= len(l.src) {
		return Token{Kind: TokEOF, Line: l.line, Col: l.col}
	}
	line, col := l.line, l.col
	r, w := utf8.DecodeRuneInString(l.src[l.pos:])
	switch r {
	case '{':
		l.adv(w)
		return Token{Kind: TokLBrace, Lit: "{", Line: line, Col: col}
	case '}':
		l.adv(w)
		return Token{Kind: TokRBrace, Lit: "}", Line: line, Col: col}
	case '(':
		l.adv(w)
		return Token{Kind: TokLParen, Lit: "(", Line: line, Col: col}
	case ')':
		l.adv(w)
		return Token{Kind: TokRParen, Lit: ")", Line: line, Col: col}
	case '[':
		return l.durationBrack(line, col)
	case ']':
		l.adv(w)
		return Token{Kind: TokRBrack, Lit: "]", Line: line, Col: col}
	case ',':
		l.adv(w)
		return Token{Kind: TokComma, Lit: ",", Line: line, Col: col}
	case '+':
		l.adv(w)
		return Token{Kind: TokAdd, Lit: "+", Line: line, Col: col}
	case '-':
		l.adv(w)
		return Token{Kind: TokSub, Lit: "-", Line: line, Col: col}
	case '*':
		l.adv(w)
		return Token{Kind: TokMul, Lit: "*", Line: line, Col: col}
	case '/':
		l.adv(w)
		return Token{Kind: TokDiv, Lit: "/", Line: line, Col: col}
	case '%':
		l.adv(w)
		return Token{Kind: TokMod, Lit: "%", Line: line, Col: col}
	case '^':
		l.adv(w)
		return Token{Kind: TokPow, Lit: "^", Line: line, Col: col}
	case '"', '\'':
		return l.string(r, line, col)
	case '=':
		l.adv(w)
		if l.peek() == '~' {
			l.adv(1)
			return Token{Kind: TokRe, Lit: "=~", Line: line, Col: col}
		}
		return Token{Kind: TokEq, Lit: "=", Line: line, Col: col}
	case '!':
		l.adv(w)
		if l.peek() == '=' {
			l.adv(1)
			return Token{Kind: TokNe, Lit: "!=", Line: line, Col: col}
		}
		if l.peek() == '~' {
			l.adv(1)
			return Token{Kind: TokNre, Lit: "!~", Line: line, Col: col}
		}
		return Token{Kind: TokError, Lit: "unexpected '!'", Line: line, Col: col}
	case '>':
		l.adv(w)
		if l.peek() == '=' {
			l.adv(1)
			return Token{Kind: TokGte, Lit: ">=", Line: line, Col: col}
		}
		return Token{Kind: TokGt, Lit: ">", Line: line, Col: col}
	case '<':
		l.adv(w)
		if l.peek() == '=' {
			l.adv(1)
			return Token{Kind: TokLte, Lit: "<=", Line: line, Col: col}
		}
		return Token{Kind: TokLt, Lit: "<", Line: line, Col: col}
	}
	if unicode.IsDigit(r) || r == '.' {
		return l.numberOrDur(line, col)
	}
	if isIdentStart(r) {
		return l.ident(line, col)
	}
	return Token{Kind: TokError, Lit: fmt.Sprintf("unexpected %q", r), Line: line, Col: col}
}

func (l *Lexer) durationBrack(line, col int) Token {
	l.adv(1)
	start := l.pos
	for l.pos < len(l.src) {
		r, w := utf8.DecodeRuneInString(l.src[l.pos:])
		if r == ']' {
			lit := strings.TrimSpace(l.src[start:l.pos])
			l.adv(w)
			return Token{Kind: TokDuration, Lit: lit, Line: line, Col: col}
		}
		l.adv(w)
	}
	return Token{Kind: TokError, Lit: "unclosed [duration]", Line: line, Col: col}
}

func (l *Lexer) string(q rune, line, col int) Token {
	l.adv(utf8.RuneLen(q))
	var b strings.Builder
	for l.pos < len(l.src) {
		r, w := utf8.DecodeRuneInString(l.src[l.pos:])
		if r == q {
			l.adv(w)
			return Token{Kind: TokString, Lit: b.String(), Line: line, Col: col}
		}
		if r == '\\' {
			l.adv(w)
			if l.pos >= len(l.src) {
				break
			}
			nr, nw := utf8.DecodeRuneInString(l.src[l.pos:])
			l.adv(nw)
			b.WriteRune(nr)
			continue
		}
		l.adv(w)
		b.WriteRune(r)
	}
	return Token{Kind: TokError, Lit: "unterminated string", Line: line, Col: col}
}

func (l *Lexer) numberOrDur(line, col int) Token {
	start := l.pos
	for l.pos < len(l.src) {
		r, w := utf8.DecodeRuneInString(l.src[l.pos:])
		if unicode.IsDigit(r) || r == '.' || r == 'e' || r == 'E' {
			l.adv(w)
			continue
		}
		break
	}
	lit := l.src[start:l.pos]
	if l.pos < len(l.src) {
		r, _ := utf8.DecodeRuneInString(l.src[l.pos:])
		if r == 's' || r == 'm' || r == 'h' || r == 'd' {
			return Token{Kind: TokError, Lit: "duration must be inside []", Line: line, Col: col}
		}
	}
	return Token{Kind: TokNumber, Lit: lit, Line: line, Col: col}
}

func (l *Lexer) ident(line, col int) Token {
	start := l.pos
	for l.pos < len(l.src) {
		r, w := utf8.DecodeRuneInString(l.src[l.pos:])
		if isIdentCont(r) {
			l.adv(w)
			continue
		}
		break
	}
	lit := l.src[start:l.pos]
	switch lit {
	case "by":
		return Token{Kind: TokBy, Lit: lit, Line: line, Col: col}
	case "without":
		return Token{Kind: TokWithout, Lit: lit, Line: line, Col: col}
	}
	return Token{Kind: TokIdent, Lit: lit, Line: line, Col: col}
}

func (l *Lexer) skipWS() {
	for l.pos < len(l.src) {
		r, w := utf8.DecodeRuneInString(l.src[l.pos:])
		if r == '#' {
			for l.pos < len(l.src) {
				r, w = utf8.DecodeRuneInString(l.src[l.pos:])
				l.adv(w)
				if r == '\n' {
					break
				}
			}
			continue
		}
		if unicode.IsSpace(r) {
			l.adv(w)
			continue
		}
		return
	}
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.src[l.pos:])
	return r
}

func (l *Lexer) adv(w int) {
	for i := 0; i < w && l.pos < len(l.src); i++ {
		if l.src[l.pos] == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		l.pos++
	}
}

func isIdentStart(r rune) bool {
	return r == '_' || r == ':' || unicode.IsLetter(r)
}

func isIdentCont(r rune) bool {
	return isIdentStart(r) || unicode.IsDigit(r)
}
