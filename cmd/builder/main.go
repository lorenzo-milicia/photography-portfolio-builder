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
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/generator"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "builder", "server":
		runBuilder()
	case "generate":
		runGenerate()
	case "serve":
		runServe()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
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

func runGenerate() {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	baseURL := fs.String("base-url", "", "Base URL for the site (e.g., '' for root or '/preview' for local preview)")
	outputDir := fs.String("output", "output", "Output directory for generated site")
	contentDir := fs.String("content", "content", "Content directory")
	debug := fs.Bool("debug", false, "Enable debug logging")
	fs.Parse(os.Args[2:])

	setupLogging(*debug)
	log.Info().Msg("Starting static site generation")

	// Get absolute paths
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get working directory")
	}

	absContentDir := filepath.Join(workDir, *contentDir)
	absOutputDir := filepath.Join(workDir, *outputDir)

	log.Info().
		Str("contentDir", absContentDir).
		Str("outputDir", absOutputDir).
		Str("baseURL", *baseURL).
		Msg("Generation settings")

	// Create generator
	gen := generator.NewGenerator(absContentDir, absOutputDir, assets.TemplatesFS, assets.StaticFS)

	// Generate site
	if err := gen.Generate(*baseURL); err != nil {
		log.Fatal().Err(err).Msg("Generation failed")
	}

	log.Info().Str("outputDir", filepath.Join(absOutputDir, "public")).Msg("Site generated successfully")
}

func runServe() {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 8000, "Port to serve on")
	dir := fs.String("dir", "output/public", "Directory to serve")
	fs.Parse(os.Args[2:])

	setupLogging(false)

	// Get absolute path
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get working directory")
	}

	serveDir := filepath.Join(workDir, *dir)

	// Check if directory exists
	if _, err := os.Stat(serveDir); os.IsNotExist(err) {
		log.Fatal().Str("dir", serveDir).Msg("Directory does not exist. Run 'generate' first.")
	}

	// Create file server
	fileServer := http.FileServer(http.Dir(serveDir))
	http.Handle("/", fileServer)

	addr := fmt.Sprintf(":%d", *port)
	log.Info().
		Str("address", fmt.Sprintf("http://localhost%s", addr)).
		Str("directory", serveDir).
		Msg("Serving static files")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal().Err(err).Msg("Server failed")
	}
}
