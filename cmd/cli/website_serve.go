package cli

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var servePort int
var serveDir string

var websiteServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve generated static site",
	Long:  `Serve the generated static site from the output directory for quick local preview.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Resolve path
		workDir, err := os.Getwd()
		if err != nil {
			fmt.Printf("failed to get working directory: %v\n", err)
			os.Exit(1)
		}

		var dir string
		if filepath.IsAbs(serveDir) {
			dir = serveDir
		} else {
			dir = filepath.Join(workDir, serveDir)
		}

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			fmt.Printf("Directory does not exist: %s. Run 'website build' first.\n", dir)
			os.Exit(1)
		}

		addr := fmt.Sprintf(":%d", servePort)
		fmt.Printf("Serving %s at http://localhost%s\n", dir, addr)

		fs := http.FileServer(http.Dir(dir))
		http.Handle("/", fs)

		if err := http.ListenAndServe(addr, nil); err != nil {
			fmt.Printf("Server failed: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	websiteCmd.AddCommand(websiteServeCmd)
	websiteServeCmd.Flags().IntVarP(&servePort, "port", "p", 8000, "Port to serve on")
	websiteServeCmd.Flags().StringVarP(&serveDir, "dir", "d", "dist", "Directory to serve (relative to workspace)")
}
