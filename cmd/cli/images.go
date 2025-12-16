package cli

import (
	"github.com/spf13/cobra"
)

var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Image management commands",
	Long:  `Parent command for all image related operations.`,
}

func init() {
	rootCmd.AddCommand(imagesCmd)
}
