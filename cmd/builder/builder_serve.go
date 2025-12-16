package cli

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.lorenzomilicia.dev/photography-portfolio-builder/assets"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/builder"
)

var (
	builderPort       int
	builderDebug      bool
	builderContentDir string
	builderOutputDir  string
)

var builderServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the interactive builder server",
	Long:  `Start the interactive web-based builder server for managing projects, layouts, and generating the static site.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Setup logging
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		if builderDebug {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
		} else {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
		}

		log.Info().Msg("Starting Photography Portfolio Builder Server")

		// Get absolute paths
		workDir, err := os.Getwd()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get working directory")
		}

		log.Debug().Str("workDir", workDir).Msg("Working directory")

		// Setup paths (resolve relative to workDir if not absolute)
		contentDir := builderContentDir
		if !filepath.IsAbs(contentDir) {
			contentDir = filepath.Join(workDir, contentDir)
		}

		outputDir := builderOutputDir
		if !filepath.IsAbs(outputDir) {
			outputDir = filepath.Join(workDir, outputDir)
		}

		// Setup photos directory (separate from content)
		photosDir := "photos"
		if !filepath.IsAbs(photosDir) {
			photosDir = filepath.Join(workDir, photosDir)
		}

		// Create builder server
		log.Info().Msg("Initializing server")
		srv, err := builder.NewServer(assets.TemplatesFS, assets.StaticFS, contentDir, photosDir, outputDir)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create server")
		}

		// Setup routes
		mux := http.NewServeMux()
		srv.RegisterRoutes(mux)

		// Start server
		addr := fmt.Sprintf(":%d", builderPort)
		log.Info().
			Str("address", fmt.Sprintf("http://localhost%s", addr)).
			Int("port", builderPort).
			Msg("Server listening")

		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatal().Err(err).Msg("Server failed")
		}
	},
}

func init() {
	builderCmd.AddCommand(builderServeCmd)

	builderServeCmd.Flags().IntVarP(&builderPort, "port", "p", 8080, "Port to run the builder server on")
	builderServeCmd.Flags().BoolVar(&builderDebug, "debug", false, "Enable debug logging")
	builderServeCmd.Flags().StringVarP(&builderContentDir, "content", "c", "content", "Content directory")
	builderServeCmd.Flags().StringVarP(&builderOutputDir, "output", "o", "dist", "Output directory where processed images are stored")
}
