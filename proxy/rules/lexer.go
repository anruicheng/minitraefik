package rules

import "fmt"

type tokenKind int

const (
	tokEOF     tokenKind = iota
	tokAnd               // &&
	tokOr                // ||
	tokNot               // !
	tokLParen            // (
	tokRParen            // )
	tokMatcher           // Host, PathPrefix, Method
	tokValue             // `...` 里的内容
)

// 输入 Host(`api.localhost`) && PathPrefix(`/v1`)
// 输出 [MATCHER "Host"] [LPAREN] [VALUE "api.localhost"] [RPAREN] [AND] [MATCHER "PathPrefix"] [LPAREN] [VALUE "/v1"] [RPAREN] [EOF]
type token struct {
	kind tokenKind
	text string
}

type lexer struct {
	input string
	pos   int
}

func newLexer(input string) *lexer {
	return &lexer{input: input}
}

func (l *lexer) next() (token, error) {
	l.skipSpace()
	if l.pos >= len(l.input) {
		return token{kind: tokEOF}, nil
	}

	switch l.input[l.pos] {
	case '(':
		l.pos++
		return token{kind: tokLParen, text: "("}, nil

	case ')':
		l.pos++
		return token{kind: tokRParen, text: ")"}, nil

	case '!':
		l.pos++
		return token{kind: tokNot, text: "!"}, nil

	case '&':
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '&' {
			l.pos += 2
			return token{kind: tokAnd, text: "&&"}, nil
		}
		return token{}, fmt.Errorf("expected &&")

	case '|':
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '|' {
			l.pos += 2
			return token{kind: tokOr, text: "||"}, nil
		}
		return token{}, fmt.Errorf("expected ||")

	case '`':
		return l.readBacktick()
	}

	ch := l.input[l.pos]
	if isLetter(ch) {
		return l.readIdent(), nil
	}
	return token{}, fmt.Errorf("unexpected char %q", ch)
}

// readBacktick 读取 `...` 里的内容，返回 tokValue。
func (l *lexer) readBacktick() (token, error) {
	l.pos++ // 跳过开头的反引号
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != '`' {
		l.pos++
	}
	if l.pos >= len(l.input) {
		return token{}, fmt.Errorf("unterminated backtick")
	}
	value := l.input[start:l.pos]
	l.pos++ // 跳过结尾的反引号
	return token{kind: tokValue, text: value}, nil
}

// readIdent 读取标识符，返回 tokMatcher。
func (l *lexer) readIdent() token {
	start := l.pos
	for l.pos < len(l.input) && (isLetter(l.input[l.pos]) || isDigit(l.input[l.pos])) {
		l.pos++
	}
	return token{kind: tokMatcher, text: l.input[start:l.pos]}
}

func (l *lexer) skipSpace() {
	for l.pos < len(l.input) {
		switch l.input[l.pos] {
		case ' ', '\t', '\n', '\r':
			l.pos++
		default:
			return
		}
	}
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
