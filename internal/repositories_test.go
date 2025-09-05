package internal

import (
	"testing"

	"github.com/The-Skyscape/devtools/pkg/testutils"
)

func TestParseRepoName(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"https://github.com/user/repo.git", "repo"},
		{"https://github.com/user/repo", "repo"},
		{"git@github.com:user/repo.git", "repo"},
		{"git@github.com:user/repo", "repo"},
		{"https://gitlab.com/group/subgroup/project.git", "project"},
		{"git@bitbucket.org:team/project.git", "project"},
		{"https://example.com/path/to/repo.git", "repo"},
		{"invalid-url", "invalid-url"},
		{"", ""},
		{"https://github.com/", "github.com"},
		{"https://github.com", "github.com"},
		{"https://github.com/user/repo/", "repo"},
		{"/", ""},
		{"https://", ""},
	}
	
	for _, tc := range testCases {
		result := parseRepoName(tc.input)
		testutils.AssertEqual(t, tc.expected, result)
	}
}