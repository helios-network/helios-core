package version

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	AppVersion = "v0.0.272"
	GitCommit  = ""
	BuildDate  = ""

	GoVersion = ""
	GoArch    = ""
)

func init() {
	GoVersion = runtime.Version()
	GoArch = runtime.GOARCH
}

func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the application binary version information",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SetOut(cmd.OutOrStdout())
			output := fmt.Sprintf(
				"{\"version\": \"%s\", \"commit\": \"%s\", \"compiledAt\": \"%s\", \"goVersion\": \"%s\", \"goArch\": \"%s\"}",
				AppVersion, GitCommit, BuildDate, GoVersion, GoArch)
			cmd.Println(output)
			return nil
		},
	}
	return cmd
}
