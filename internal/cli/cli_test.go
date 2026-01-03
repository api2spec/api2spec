// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// executeCommand runs a command and returns output and error.
func executeCommand(root *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	err := root.Execute()
	return buf.String(), err
}

func TestRootCommand_Help(t *testing.T) {
	output, err := executeCommand(rootCmd, "--help")
	require.NoError(t, err)

	assert.Contains(t, output, "api2spec")
	assert.Contains(t, output, "code-first OpenAPI specification generator")
	assert.Contains(t, output, "Available Commands")
	assert.Contains(t, output, "generate")
	assert.Contains(t, output, "init")
	assert.Contains(t, output, "check")
	assert.Contains(t, output, "diff")
	assert.Contains(t, output, "watch")
	assert.Contains(t, output, "print")
	assert.Contains(t, output, "version")
}

func TestRootCommand_GlobalFlags(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{
			name:     "config flag short",
			flag:     "-c",
			expected: "config file",
		},
		{
			name:     "config flag long",
			flag:     "--config",
			expected: "config file",
		},
		{
			name:     "output flag short",
			flag:     "-o",
			expected: "output file path",
		},
		{
			name:     "output flag long",
			flag:     "--output",
			expected: "output file path",
		},
		{
			name:     "format flag short",
			flag:     "-f",
			expected: "output format",
		},
		{
			name:     "format flag long",
			flag:     "--format",
			expected: "output format",
		},
		{
			name:     "framework flag",
			flag:     "--framework",
			expected: "web framework",
		},
		{
			name:     "verbose flag short",
			flag:     "-v",
			expected: "verbose output",
		},
		{
			name:     "verbose flag long",
			flag:     "--verbose",
			expected: "verbose output",
		},
		{
			name:     "quiet flag short",
			flag:     "-q",
			expected: "suppress",
		},
		{
			name:     "quiet flag long",
			flag:     "--quiet",
			expected: "suppress",
		},
	}

	output, err := executeCommand(rootCmd, "--help")
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, output, tt.flag)
			assert.Contains(t, output, tt.expected)
		})
	}
}

func TestVersionCommand(t *testing.T) {
	output, err := executeCommand(rootCmd, "version")
	require.NoError(t, err)

	assert.Contains(t, output, "api2spec")
	assert.Contains(t, output, "Commit")
	assert.Contains(t, output, "Build Date")
	assert.Contains(t, output, "Go Version")
	assert.Contains(t, output, "OS/Arch")
}

func TestInitCommand_Help(t *testing.T) {
	output, err := executeCommand(rootCmd, "init", "--help")
	require.NoError(t, err)

	assert.Contains(t, output, "Initialize a new api2spec configuration file")
	assert.Contains(t, output, "--framework")
	assert.Contains(t, output, "--force")
}

func TestGenerateCommand_Help(t *testing.T) {
	output, err := executeCommand(rootCmd, "generate", "--help")
	require.NoError(t, err)

	assert.Contains(t, output, "Generate an OpenAPI specification")
	assert.Contains(t, output, "--mode")
	assert.Contains(t, output, "--merge")
	assert.Contains(t, output, "--dry-run")
	assert.Contains(t, output, "--include")
	assert.Contains(t, output, "--exclude")
}

func TestCheckCommand_Help(t *testing.T) {
	output, err := executeCommand(rootCmd, "check", "--help")
	require.NoError(t, err)

	assert.Contains(t, output, "Check validates that your OpenAPI specification matches your current code")
	assert.Contains(t, output, "--strict")
	assert.Contains(t, output, "--ignore")
	assert.Contains(t, output, "--ci")
}

func TestDiffCommand_Help(t *testing.T) {
	output, err := executeCommand(rootCmd, "diff", "--help")
	require.NoError(t, err)

	assert.Contains(t, output, "Compare two OpenAPI specifications")
	assert.Contains(t, output, "--color")
	assert.Contains(t, output, "--unified")
	assert.Contains(t, output, "--side-by-side")
}

func TestWatchCommand_Help(t *testing.T) {
	output, err := executeCommand(rootCmd, "watch", "--help")
	require.NoError(t, err)

	assert.Contains(t, output, "Watch for file changes")
	assert.Contains(t, output, "--mode")
	assert.Contains(t, output, "--debounce")
	assert.Contains(t, output, "--on-change")
}

func TestPrintCommand_Help(t *testing.T) {
	output, err := executeCommand(rootCmd, "print", "--help")
	require.NoError(t, err)

	assert.Contains(t, output, "Print the OpenAPI specification")
}

func TestGetVersionInfo(t *testing.T) {
	info := GetVersionInfo()
	assert.Contains(t, info, "api2spec")
	assert.Contains(t, info, "commit")
	assert.Contains(t, info, "built")
}
