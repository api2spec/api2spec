// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/config"
)

func TestDetectProjectInfo(t *testing.T) {
	tests := []struct {
		name         string
		goModContent string
		wantTitle    string
		wantModule   string
	}{
		{
			name: "simple module",
			goModContent: `module github.com/user/myapp

go 1.21
`,
			wantTitle:  "Myapp API",
			wantModule: "github.com/user/myapp",
		},
		{
			name: "module with hyphens",
			goModContent: `module github.com/user/my-awesome-api

go 1.21
`,
			wantTitle:  "My Awesome Api API",
			wantModule: "github.com/user/my-awesome-api",
		},
		{
			name: "module with underscores",
			goModContent: `module github.com/user/my_api_service

go 1.21
`,
			wantTitle:  "My Api Service API",
			wantModule: "github.com/user/my_api_service",
		},
		{
			name: "simple name",
			goModContent: `module api

go 1.21
`,
			wantTitle:  "Api API",
			wantModule: "api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			goModPath := filepath.Join(tmpDir, "go.mod")
			err := os.WriteFile(goModPath, []byte(tt.goModContent), 0644)
			require.NoError(t, err)

			info := detectProjectInfo(tmpDir)

			assert.Equal(t, tt.wantModule, info.Module)
			assert.Equal(t, tt.wantTitle, info.Title)
		})
	}
}

func TestDetectProjectInfo_NoGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	info := detectProjectInfo(tmpDir)

	assert.Empty(t, info.Module)
	assert.Empty(t, info.Title)
}

func TestDetectEntryPoints(t *testing.T) {
	tests := []struct {
		name     string
		dirs     []string
		expected []string
	}{
		{
			name:     "cmd and internal",
			dirs:     []string{"cmd", "internal"},
			expected: []string{"./cmd", "./internal"},
		},
		{
			name:     "all common directories",
			dirs:     []string{"cmd", "internal", "pkg", "api", "handlers"},
			expected: []string{"./cmd", "./internal", "./pkg", "./api", "./handlers"},
		},
		{
			name:     "no common directories",
			dirs:     []string{"src", "lib"},
			expected: []string{"."},
		},
		{
			name:     "just handlers",
			dirs:     []string{"handlers"},
			expected: []string{"./handlers"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create the directories
			for _, dir := range tt.dirs {
				err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755)
				require.NoError(t, err)
			}

			paths := detectEntryPoints(tmpDir)

			assert.Equal(t, tt.expected, paths)
		})
	}
}

func TestDetectEntryPoints_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	paths := detectEntryPoints(tmpDir)

	assert.Equal(t, []string{"."}, paths)
}

func TestBuildConfigYAML(t *testing.T) {
	cfg := config.Default()
	cfg.Framework = "chi"
	cfg.Output = "openapi.yaml"
	cfg.Format = "yaml"

	yaml := buildConfigYAML(cfg)

	assert.Contains(t, yaml, "# api2spec configuration file")
	assert.Contains(t, yaml, "framework: chi")
	assert.Contains(t, yaml, "output: openapi.yaml")
}
