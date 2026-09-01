package pattern

import "testing"

func TestReanchor(t *testing.T) {
	tests := []struct {
		name    string
		dir     string
		pattern string
		want    string
	}{
		{"root keeps anchored pattern", ".", "/foo", "/foo"},
		{"root keeps middle slash", ".", "bar/baz", "bar/baz"},
		{"no slash matches at any depth", "sub", "foo", "foo"},
		{"glob without slash", "sub", "*.log", "*.log"},
		{"trailing slash only is unanchored", "sub", "build/", "build/"},
		{"leading slash", "sub", "/foo", "/sub/foo"},
		{"middle slash", "sub", "bar/baz", "/sub/bar/baz"},
		{"leading and trailing slash", "sub", "/foo/", "/sub/foo/"},
		{"leading double star", "sub", "**/foo", "/sub/**/foo"},
		{"trailing double star", "sub", "foo/**", "/sub/foo/**"},
		{"nested dir", "a/b", "/foo", "/a/b/foo"},
		{"trailing spaces are trimmed before deciding", "sub", "build/  ", "build/  "},
		{"escaped trailing space counts", "sub", "build/\\ ", "/sub/build/\\ "},
		{"trailing backslash is kept", "sub", "build/\\", "/sub/build/\\"},
		{"inner space does not trim", "sub", "a b/c", "/sub/a b/c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Reanchor(tt.dir, tt.pattern)
			if got != tt.want {
				t.Errorf("Reanchor(%q, %q) = %q, want %q", tt.dir, tt.pattern, got, tt.want)
			}
		})
	}
}
