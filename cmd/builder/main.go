package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.lorenzomilicia.dev/photography-portfolio-builder/assets"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/builder"
)

func main() {
	Execute()
}

func printUsage() {
	fmt.Println("Photography Portfolio Builder")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  builder <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  builder    Start the interactive builder server")
	fmt.Println("  generate   Generate static site for production")
	fmt.Println("  serve      Serve the generated static site")
	fmt.Println()
	fmt.Println("Run 'builder <command> -h' for command-specific options")
}

func setupLogging(debug bool) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}
}

func runBuilder() {
	fs := flag.NewFlagSet("builder", flag.ExitOnError)
	port := fs.Int("port", 8080, "Port to run the builder server on")
	debug := fs.Bool("debug", false, "Enable debug logging")
	fs.Parse(os.Args[2:])

	setupLogging(*debug)
	log.Info().Msg("Starting Photography Portfolio Builder")

	// Get absolute paths
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get working directory")
	}

	log.Debug().Str("workDir", workDir).Msg("Working directory")

	// Setup paths
	contentDir := filepath.Join(workDir, "content")
	outputDir := filepath.Join(workDir, "output")

	// Create builder server
	log.Info().Msg("Initializing server")
	srv, err := builder.NewServer(assets.TemplatesFS, assets.StaticFS, contentDir, outputDir)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create server")
	}

	// Setup routes
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Start server
	addr := fmt.Sprintf(":%d", *port)
	log.Info().
		Str("address", fmt.Sprintf("http://localhost%s", addr)).
		Int("port", *port).
		Msg("Server listening")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal().Err(err).Msg("Server failed")
	}
}
