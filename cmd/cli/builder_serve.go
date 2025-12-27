package cli

import (
	"github.com/spf13/cobra"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/builder"
)

var builderServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the interactive builder server",
	Long:  `Start the interactive web-based builder server for managing projects, layouts, and generating the static site.`,
	Run: func(cmd *cobra.Command, args []string) {
		builder.ServeNew()
	},
}

func init() {
	builderCmd.AddCommand(builderServeCmd)
}
