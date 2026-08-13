package testutil

import (
	"os"
	"slices"
	"testing"
)

func TestGitEnvNames(t *testing.T) {
	tests := []struct {
		name    string
		environ []string
		want    []string
	}{
		{
			name:    "picks git variables",
			environ: []string{"PATH=/usr/bin", "GIT_INDEX_FILE=/repo/.git/index", "GIT_DIR=/repo/.git"},
			want:    []string{"GIT_INDEX_FILE", "GIT_DIR"},
		},
		{
			name:    "skips names that merely contain GIT_",
			environ: []string{"LEGIT_VAR=1", "MY_GIT_DIR=/elsewhere"},
			want:    nil,
		},
		{
			name:    "skips entries without a separator",
			environ: []string{"GIT_MALFORMED"},
			want:    nil,
		},
		{
			name:    "empty environment yields nothing",
			environ: nil,
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gitEnvNames(tt.environ); !slices.Equal(got, tt.want) {
				t.Errorf("gitEnvNames(%q) = %q, want %q", tt.environ, got, tt.want)
			}
		})
	}
}

func TestSetupRepoIsolatesGitEnv(t *testing.T) {
	t.Setenv("GIT_INDEX_FILE", "/elsewhere/.git/index")

	SetupRepo(t)

	if value, ok := os.LookupEnv("GIT_INDEX_FILE"); ok {
		t.Errorf("GIT_INDEX_FILE = %q, want unset", value)
	}
}
