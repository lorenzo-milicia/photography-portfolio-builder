package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.lorenzomilicia.dev/photography-portfolio-builder/assets"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/generator"
)

var host string
var contentDirCLI string
var outputDirCLI string

var websiteBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the static website",
	Long:  `Generate the static website using processed images and content definitions.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Building website...\n")
		fmt.Printf("Content directory: %s\n", contentDirCLI)
		fmt.Printf("Output directory: %s\n", outputDirCLI)
		if host != "" {
			fmt.Printf("Using image host: %s\n", host)
		} else {
			fmt.Println("Using local images (no host specified)")
		}

		// If host not provided via flag, allow environment variable override
		if host == "" {
			// Prefer IMAGE_HOST, fall back to IMAGE_URL_PREFIX for backwards compatibility
			if v := os.Getenv("IMAGE_HOST"); v != "" {
				host = v
			} else if v := os.Getenv("IMAGE_URL_PREFIX"); v != "" {
				host = v
			}
		}

		// Create generator
		gen := generator.NewGenerator(contentDirCLI, outputDirCLI, assets.TemplatesFS, assets.StaticFS)

		// Generate site (baseURL empty for root-relative paths, imageURLPrefix from --host flag)
		if err := gen.Generate("", host); err != nil {
			fmt.Printf("Error generating site: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Website build complete!")
	},
}

func init() {
	websiteCmd.AddCommand(websiteBuildCmd)

	websiteBuildCmd.Flags().StringVar(&host, "host", "", "Host URL for images (e.g., https://my-bucket.s3.amazonaws.com)")
	websiteBuildCmd.Flags().StringVarP(&contentDirCLI, "content", "c", "content", "Content directory")
	websiteBuildCmd.Flags().StringVarP(&outputDirCLI, "output", "o", "dist", "Output directory for the static site")
}
