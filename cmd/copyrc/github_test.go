package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGithubRepo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantOrg  string
		wantRepo string
		wantErr  bool
	}{
		{
			name:     "simple repo",
			input:    "github.com/org/repo",
			wantOrg:  "org",
			wantRepo: "repo",
			wantErr:  false,
		},
		{
			name:     "repo with ref",
			input:    "github.com/golang/tools@master",
			wantOrg:  "golang",
			wantRepo: "tools",
			wantErr:  false,
		},
		{
			name:     "repo with From prefix",
			input:    "From github.com/golang/tools@master",
			wantOrg:  "golang",
			wantRepo: "tools",
			wantErr:  false,
		},
		{
			name:    "invalid format",
			input:   "not/enough/parts",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org, repo, err := parseGithubRepo(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOrg, org)
			assert.Equal(t, tt.wantRepo, repo)
		})
	}
}
