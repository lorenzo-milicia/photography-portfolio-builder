package cli

import (
	"github.com/spf13/cobra"
)

var builderCmd = &cobra.Command{
	Use:   "builder",
	Short: "Interactive builder server commands",
	Long:  `Commands for running the interactive web-based builder server.`,
}

func init() {
	rootCmd.AddCommand(builderCmd)
}
