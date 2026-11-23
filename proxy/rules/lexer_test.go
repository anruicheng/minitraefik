package rules

import (
	"reflect"
	"testing"
)

func TestLexer(t *testing.T) {
	tests := []struct {
		input string
		want  []token
	}{
		{
			// 最典型的规则
			input: "Host(`api.localhost`)",
			want: []token{
				{kind: tokMatcher, text: "Host"},
				{kind: tokLParen, text: "("},
				{kind: tokValue, text: "api.localhost"},
				{kind: tokRParen, text: ")"},
				{kind: tokEOF},
			},
		},
		{
			// 两个匹配器用 && 连接
			input: "Host(`api.localhost`) && PathPrefix(`/v1`)",
			want: []token{
				{kind: tokMatcher, text: "Host"},
				{kind: tokLParen, text: "("},
				{kind: tokValue, text: "api.localhost"},
				{kind: tokRParen, text: ")"},
				{kind: tokAnd, text: "&&"},
				{kind: tokMatcher, text: "PathPrefix"},
				{kind: tokLParen, text: "("},
				{kind: tokValue, text: "/v1"},
				{kind: tokRParen, text: ")"},
				{kind: tokEOF},
			},
		},
		{
			input: "!Method(`DELETE`) && (PathPrefix(`/v1`) || PathPrefix(`/v2`))",
			want: []token{
				{kind: tokNot, text: "!"},
				{kind: tokMatcher, text: "Method"},
				{kind: tokLParen, text: "("},
				{kind: tokValue, text: "DELETE"},
				{kind: tokRParen, text: ")"},
				{kind: tokAnd, text: "&&"},
				{kind: tokLParen, text: "("},
				{kind: tokMatcher, text: "PathPrefix"},
				{kind: tokLParen, text: "("},
				{kind: tokValue, text: "/v1"},
				{kind: tokRParen, text: ")"},
				{kind: tokOr, text: "||"},
				{kind: tokMatcher, text: "PathPrefix"},
				{kind: tokLParen, text: "("},
				{kind: tokValue, text: "/v2"},
				{kind: tokRParen, text: ")"},
				{kind: tokRParen, text: ")"},
				{kind: tokEOF},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := lexAll(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("input %q\n got: %+v\nwant: %+v", tt.input, got, tt.want)
			}
		})
	}
}

func lexAll(input string) ([]token, error) {
	l := newLexer(input)
	var toks []token
	for {
		tok, err := l.next()
		if err != nil {
			return toks, err
		}
		toks = append(toks, tok)
		if tok.kind == tokEOF {
			return toks, nil
		}
	}
}
