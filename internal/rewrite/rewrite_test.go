package rewrite

import "testing"

func TestApply(t *testing.T) {
	tests := []struct {
		name    string
		content string
		drop    map[int]bool
		want    string
	}{
		{
			name:    "empty input",
			content: "",
			drop:    nil,
			want:    "",
		},
		{
			name:    "no drops",
			content: "a\nb\nc\n",
			drop:    nil,
			want:    "a\nb\nc\n",
		},
		{
			name:    "drop middle line lf",
			content: "a\nb\nc\n",
			drop:    map[int]bool{2: true},
			want:    "a\nc\n",
		},
		{
			name:    "drop middle line crlf",
			content: "a\r\nb\r\nc\r\n",
			drop:    map[int]bool{2: true},
			want:    "a\r\nc\r\n",
		},
		{
			name:    "drop first line",
			content: "a\nb\n",
			drop:    map[int]bool{1: true},
			want:    "b\n",
		},
		{
			name:    "drop last line with trailing newline",
			content: "a\nb\n",
			drop:    map[int]bool{2: true},
			want:    "a\n",
		},
		{
			name:    "drop trailing partial line (no final newline)",
			content: "a\nb",
			drop:    map[int]bool{2: true},
			want:    "a\n",
		},
		{
			name:    "keep trailing partial line",
			content: "a\nb",
			drop:    map[int]bool{1: true},
			want:    "b",
		},
		{
			name:    "drop all lines",
			content: "a\nb\nc\n",
			drop:    map[int]bool{1: true, 2: true, 3: true},
			want:    "",
		},
		{
			name:    "mixed line endings preserved",
			content: "a\r\nb\nc\r\n",
			drop:    map[int]bool{2: true},
			want:    "a\r\nc\r\n",
		},
		{
			name:    "drop number out of range is no-op",
			content: "a\nb\n",
			drop:    map[int]bool{99: true},
			want:    "a\nb\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Apply(tt.content, tt.drop)
			if got != tt.want {
				t.Errorf("Apply() = %q, want %q", got, tt.want)
			}
		})
	}
}
