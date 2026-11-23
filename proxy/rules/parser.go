package rules

import "fmt"

type parser struct {
	lex *lexer
	cur token
	err error
}

func Parse(input string) (Node, error) {
	p := &parser{lex: newLexer(input)}
	p.advance()
	node := p.parseOr()
	if p.err != nil {
		return nil, p.err
	}
	if p.cur.kind != tokEOF {
		return nil, fmt.Errorf("unexpected %q", p.cur.text)
	}
	return node, nil
}

// parseOr: 处理 ||，优先级最低
func (p *parser) parseOr() Node {
	left := p.parseAnd()
	for p.cur.kind == tokOr && p.err == nil {
		p.advance()
		left = &OrNode{Left: left, Right: p.parseAnd()}
	}
	return left
}

// parseAnd: 处理 &&，优先级居中
func (p *parser) parseAnd() Node {
	left := p.parseUnary()
	for p.cur.kind == tokAnd && p.err == nil {
		p.advance()
		left = &AndNode{Left: left, Right: p.parseUnary()}
	}
	return left
}

// parseUnary: 处理 !、括号、叶子节点，优先级最高
func (p *parser) parseUnary() Node {
	switch p.cur.kind {
	case tokNot:
		p.advance()
		return &NotNode{Child: p.parseUnary()}

	case tokLParen:
		p.advance()
		node := p.parseOr()
		p.expect(tokRParen)
		return node

	case tokMatcher:
		return p.parseMatcherCall()

	default:
		p.fail(fmt.Errorf("unexpected %q", p.cur.text))
		return nil
	}
}

// parseMatcherCall: 处理 Host(`x`) 这种调用
func (p *parser) parseMatcherCall() Node {
	name := p.cur.text
	p.advance()
	p.expect(tokLParen)

	if p.cur.kind != tokValue {
		p.fail(fmt.Errorf("expected value in %s(...)", name))
		return nil
	}
	value := p.cur.text
	p.advance()
	p.expect(tokRParen)
	if p.err != nil {
		return nil
	}

	switch name {
	case "Host":
		return &HostMatcher{Host: value}
	case "PathPrefix":
		return &PathPrefixMatcher{Prefix: value}
	case "Method":
		return &MethodMatcher{Method: value}
	default:
		p.fail(fmt.Errorf("unknown matcher %q", name))
		return nil
	}
}

func (p *parser) expect(kind tokenKind) {
	if p.cur.kind != kind {
		p.fail(fmt.Errorf("expected %s, got %q", kindName(kind), p.cur.text))
		return
	}
	p.advance()
}

func (p *parser) advance() {
	if p.err != nil {
		return
	}
	tok, err := p.lex.next()
	if err != nil {
		p.err = err
		return
	}
	p.cur = tok
}

func (p *parser) fail(err error) {
	if p.err == nil {
		p.err = err
	}
}

func kindName(k tokenKind) string {
	switch k {
	case tokLParen:
		return "'('"
	case tokRParen:
		return "')'"
	case tokEOF:
		return "end of input"
	default:
		return "token"
	}
}
