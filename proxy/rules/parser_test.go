package rules

import (
	"net/http/httptest"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		rule string
		reqs []req
	}{
		{
			rule: "Host(`api.localhost`)",
			reqs: []req{
				{method: "GET", url: "http://api.localhost/users", want: true},
			},
		},
		{
			rule: "PathPrefix(`/v1`)",
			reqs: []req{
				{method: "GET", url: "http://localhost/v1/users", want: true},
			},
		},
		{
			rule: "Host(`api.localhost`) && PathPrefix(`/v1`)",
			reqs: []req{
				{method: "GET", url: "http://api.localhost/v1/users", want: true},
				{method: "GET", url: "http://api.localhost/v2/users", want: false},
				{method: "GET", url: "http://other.localhost/v1/users", want: false},
			},
		},
		{
			rule: "!Method(`DELETE`) && PathPrefix(`/admin`)",
			reqs: []req{
				{method: "GET", url: "http://x/admin", want: true},
				{method: "POST", url: "http://x/admin", want: true},
				{method: "DELETE", url: "http://x/admin", want: false},
			},
		},
		{
			// 括号强制先算 ||，再和 Host 做 &&
			rule: "Host(`api.localhost`) && (PathPrefix(`/v1`) || PathPrefix(`/v2`))",
			reqs: []req{
				{method: "GET", url: "http://api.localhost/v1/x", want: true},
				{method: "GET", url: "http://api.localhost/v2/x", want: true},
				{method: "GET", url: "http://api.localhost/v3/x", want: false},
				{method: "GET", url: "http://other.localhost/v1/x", want: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.rule, func(t *testing.T) {
			node, err := Parse(tt.rule)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			for _, r := range tt.reqs {
				got := node.Match(httptest.NewRequest(r.method, r.url, nil))
				if got != r.want {
					t.Errorf("%s %s: got %v, want %v", r.method, r.url, got, r.want)
				}
			}
		})
	}
}

type req struct {
	method string
	url    string
	want   bool
}
