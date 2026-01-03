// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version information set via ldflags during build.
var (
	// Version is the semantic version of the application.
	Version = "dev"

	// Commit is the git commit hash.
	Commit = "unknown"

	// BuildDate is the date the binary was built.
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  `Print the version, commit hash, build date, and Go version.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("api2spec %s\n", Version)
		cmd.Printf("  Commit:     %s\n", Commit)
		cmd.Printf("  Build Date: %s\n", BuildDate)
		cmd.Printf("  Go Version: %s\n", runtime.Version())
		cmd.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

// GetVersionInfo returns formatted version information.
func GetVersionInfo() string {
	return fmt.Sprintf("api2spec %s (commit: %s, built: %s)", Version, Commit, BuildDate)
}
