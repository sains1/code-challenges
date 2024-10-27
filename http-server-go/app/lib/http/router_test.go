package http

import "testing"

func TestGetSegment(t *testing.T) {
	// arrange
	tests := []struct {
		route    string
		expected []PathSegment
	}{
		{route: "/", expected: []PathSegment{{Part: "", IsVar: false}}},
		{route: "/hello-world", expected: []PathSegment{{Part: "hello-world", IsVar: false}}},
		{route: "hello-world", expected: []PathSegment{{Part: "hello-world", IsVar: false}}},
		{route: "/HeLlO-wOrLd", expected: []PathSegment{{Part: "hello-world", IsVar: false}}},
		{route: "hello-world/", expected: []PathSegment{{Part: "hello-world", IsVar: false}}},
		{route: "/hello-world/nested", expected: []PathSegment{{Part: "hello-world", IsVar: false}, {Part: "nested", IsVar: false}}},
		{route: "/hello-world/nested/again", expected: []PathSegment{{Part: "hello-world", IsVar: false}, {Part: "nested", IsVar: false}, {Part: "again", IsVar: false}}},
		{route: "/{variable}", expected: []PathSegment{{Part: "variable", IsVar: true}}},
		{route: "/{VaRiAbLe}", expected: []PathSegment{{Part: "VaRiAbLe", IsVar: true}}},
		{route: "{variable}", expected: []PathSegment{{Part: "variable", IsVar: true}}},
		{route: "{variable}/", expected: []PathSegment{{Part: "variable", IsVar: true}}},
		{route: "/nested/{variable}", expected: []PathSegment{{Part: "nested", IsVar: false}, {Part: "variable", IsVar: true}}},
		{route: "/nested/{variable}/again", expected: []PathSegment{{Part: "nested", IsVar: false}, {Part: "variable", IsVar: true}, {Part: "again", IsVar: false}}},
	}

	for _, test := range tests {
		// act
		actual, err := getPathSegments(test.route)

		// assert
		if err != nil {
			t.Error(err)
			return
		}

		if len(test.expected) != len(actual) {
			t.Errorf("expected %d segments but got %d (%v)", len(test.expected), len(actual), actual)
			return
		}

		for i, got := range actual {

			exp := test.expected[i]

			if exp.Part != got.Part {
				t.Errorf("expected %s but got %s", exp.Part, got.Part)
				return
			}

			if exp.IsVar != got.IsVar {
				if exp.IsVar {
					t.Errorf("expected a var but got a regular part")
				} else {
					t.Errorf("expected a regular part but got a var")
				}
				return
			}
		}
	}
}

func TestMatch(t *testing.T) {
	// arrange
	type Exp = struct {
		match bool
		vars  map[string]string
	}

	tests := []struct {
		name     string
		route    string
		target   []PathSegment
		expected Exp
	}{
		// simple matches
		{
			name:     "/ match",
			route:    "/",
			target:   []PathSegment{{Part: "", IsVar: false}},
			expected: Exp{match: true, vars: map[string]string{}},
		},
		{
			name:     "simple path match",
			route:    "/hello-world",
			target:   []PathSegment{{Part: "hello-world", IsVar: false}},
			expected: Exp{match: true, vars: map[string]string{}},
		},
		{
			name:     "missing part of segment",
			route:    "/hello",
			target:   []PathSegment{{Part: "hello-world", IsVar: false}},
			expected: Exp{match: false, vars: map[string]string{}},
		},
		{
			name:     "route matches prefix but not whole segment",
			route:    "/hello-world",
			target:   []PathSegment{{Part: "hello", IsVar: false}},
			expected: Exp{match: false, vars: map[string]string{}},
		},
		{
			name:     "case insensitive match",
			route:    "/HELLO-WORLD",
			target:   []PathSegment{{Part: "hello-world", IsVar: false}},
			expected: Exp{match: true, vars: map[string]string{}},
		},
		// nested routing
		{
			name:     "matches simple nested",
			route:    "/hello-world/nested",
			target:   []PathSegment{{Part: "hello-world", IsVar: false}, {Part: "nested", IsVar: false}},
			expected: Exp{match: true, vars: map[string]string{}},
		},
		{
			name:     "matches simple nested trailing suffix",
			route:    "/hello-world/nested/",
			target:   []PathSegment{{Part: "hello-world", IsVar: false}, {Part: "nested", IsVar: false}},
			expected: Exp{match: true, vars: map[string]string{}},
		},
		{
			name:     "does not match when route has additional parts",
			route:    "/hello-world/nested/",
			target:   []PathSegment{{Part: "hello-world", IsVar: false}},
			expected: Exp{match: false, vars: map[string]string{}},
		},
		{
			name:     "does not match when missing trailing segments",
			route:    "/hello-world",
			target:   []PathSegment{{Part: "hello-world", IsVar: false}, {Part: "nested", IsVar: false}},
			expected: Exp{match: false, vars: map[string]string{}},
		},
		// vars
		{
			name:     "simple variable",
			route:    "/my-variable-content",
			target:   []PathSegment{{Part: "variable1", IsVar: true}},
			expected: Exp{match: true, vars: map[string]string{"variable1": "my-variable-content"}},
		},
		{
			name:     "simple variable with static prefix",
			route:    "/static/my-variable-content",
			target:   []PathSegment{{Part: "static", IsVar: false}, {Part: "variable1", IsVar: true}},
			expected: Exp{match: true, vars: map[string]string{"variable1": "my-variable-content"}},
		},
		{
			name:     "simple variable with static prefix and suffix",
			route:    "/static/my-variable-content/static",
			target:   []PathSegment{{Part: "static", IsVar: false}, {Part: "variable1", IsVar: true}, {Part: "static", IsVar: false}},
			expected: Exp{match: true, vars: map[string]string{"variable1": "my-variable-content"}},
		},
		{
			name:     "multi-vars",
			route:    "/my-variable-content-1/my-variable-content-2/",
			target:   []PathSegment{{Part: "variable1", IsVar: true}, {Part: "variable2", IsVar: true}},
			expected: Exp{match: true, vars: map[string]string{"variable1": "my-variable-content-1", "variable2": "my-variable-content-2"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// act
			gotmatch, gotvars := match(test.route, test.target)

			// assert
			if gotmatch != test.expected.match {
				if test.expected.match {
					t.Errorf("expected a match but got no match")
				} else {
					t.Errorf("expected not to match but got match")
				}
				return
			}

			if len(gotvars) != len(test.expected.vars) {
				t.Errorf("expected %d vars but got %d (%v)", len(test.expected.vars), len(gotvars), gotvars)
				return
			}

			for k, v := range test.expected.vars {
				match, exists := gotvars[k]
				if !exists {
					t.Errorf("expected %s to be in output vars but didn't exist", k)
					return
				}

				if match != v {
					t.Errorf("expected var to be %s but got %s", v, match)
					return
				}
			}
		})
	}
}
