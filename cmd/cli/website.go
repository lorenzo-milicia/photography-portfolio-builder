package cli

import (
	"github.com/spf13/cobra"
)

var websiteCmd = &cobra.Command{
	Use:   "website",
	Short: "Website management commands",
	Long:  `Parent command for all website related operations.`,
}

func init() {
	rootCmd.AddCommand(websiteCmd)
}
