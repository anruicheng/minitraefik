package rules

import (
	"net/http"
	"strings"
)

type Node interface {
	Match(r *http.Request) bool
}

// 叶子节点
type HostMatcher struct{ Host string }

func (m *HostMatcher) Match(r *http.Request) bool {
	return strings.EqualFold(r.Host, m.Host)
}

type PathPrefixMatcher struct{ Prefix string }

func (m *PathPrefixMatcher) Match(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, m.Prefix)
}

type MethodMatcher struct{ Method string }

func (m *MethodMatcher) Match(r *http.Request) bool {
	return strings.EqualFold(r.Method, m.Method)
}

type AndNode struct{ Left, Right Node }

func (n *AndNode) Match(r *http.Request) bool {
	return n.Left.Match(r) && n.Right.Match(r)
}

type OrNode struct{ Left, Right Node }

func (n *OrNode) Match(r *http.Request) bool {
	return n.Left.Match(r) || n.Right.Match(r)
}

type NotNode struct{ Child Node }

func (n *NotNode) Match(r *http.Request) bool {
	return !n.Child.Match(r)
}
