package parse

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Line
	}{
		{
			name: "empty",
			in:   "",
			want: nil,
		},
		{
			name: "single entry without trailing newline",
			in:   "*.log",
			want: []Line{{LineNo: 1, Kind: KindEntry, Text: "*.log"}},
		},
		{
			name: "single entry with trailing newline",
			in:   "*.log\n",
			want: []Line{{LineNo: 1, Kind: KindEntry, Text: "*.log"}},
		},
		{
			name: "comment",
			in:   "# comment\n",
			want: []Line{{LineNo: 1, Kind: KindComment, Text: "# comment"}},
		},
		{
			name: "blank line (empty)",
			in:   "\n",
			want: []Line{{LineNo: 1, Kind: KindBlank, Text: ""}},
		},
		{
			name: "blank line (whitespace only)",
			in:   "   \n",
			want: []Line{{LineNo: 1, Kind: KindBlank, Text: "   "}},
		},
		{
			name: "crlf line endings",
			in:   "*.log\r\n",
			want: []Line{{LineNo: 1, Kind: KindEntry, Text: "*.log"}},
		},
		{
			name: "escaped hash is entry",
			in:   "\\#foo",
			want: []Line{{LineNo: 1, Kind: KindEntry, Text: "\\#foo"}},
		},
		{
			name: "negation is entry",
			in:   "!*.log",
			want: []Line{{LineNo: 1, Kind: KindEntry, Text: "!*.log"}},
		},
		{
			name: "mixed kinds with line numbers",
			in:   "*.log\n# comment\n\nfoo/",
			want: []Line{
				{LineNo: 1, Kind: KindEntry, Text: "*.log"},
				{LineNo: 2, Kind: KindComment, Text: "# comment"},
				{LineNo: 3, Kind: KindBlank, Text: ""},
				{LineNo: 4, Kind: KindEntry, Text: "foo/"},
			},
		},
		{
			name: "mixed line endings preserved as content but stripped from Text",
			in:   "a\r\nb\nc",
			want: []Line{
				{LineNo: 1, Kind: KindEntry, Text: "a"},
				{LineNo: 2, Kind: KindEntry, Text: "b"},
				{LineNo: 3, Kind: KindEntry, Text: "c"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}
