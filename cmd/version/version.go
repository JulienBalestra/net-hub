/*
Package version exposes the version information for an application
through an optional Cobra command.

To properly use it, add to go build the following flag:
	-ldflags '-s -w \
	-X github.com/JulienBalestra/net-hub/cmd/version.Version=$(VERSION) \
	-X github.com/JulienBalestra/net-hub/cmd/version.Commit=$(REVISION) \
	-X github.com/JulienBalestra/net-hub/cmd/version.Package=$(PROJECT)'


*/

package version

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Package is filled at linking time
	// This could be statically set and doesn't need the ldflags
	Package = "github.com/JulienBalestra/net-hub"

	// Version holds the complete version number. Filled in at linking time.
	Version = "0.0.0+unknown"

	// Commit is filled with the VCS (e.g. git) revision being used to build
	// the program at linking time.
	Commit = "+unknown"
)

// DisplayVersion print to stdout the package/version/revision
func DisplayVersion() {
	fmt.Printf(`package: %s
version: %s
commit: %s
go: %s
`, Package, Version, Commit, runtime.Version())
}

// NewCommand creates a command version
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:        "version",
		Short:      "Details about version, revision and compiler",
		SuggestFor: []string{"Version", "v", "V"},
		Args:       cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			DisplayVersion()
		},
	}
}
