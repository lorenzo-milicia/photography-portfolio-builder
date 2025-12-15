package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/processing"
)

var inputDir string
var outputDir string
var force bool

var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Process images for the website",
	Long:  `Scan a directory for projects and images, strip EXIF data, resize, and convert them for the website.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Processing images from: %s\n", inputDir)
		fmt.Printf("Output directory: %s\n", outputDir)

		// Initialize processor
		processor := processing.NewProcessor(processing.ProcessConfig{
			Widths:             []int{480, 800, 1200, 1920},
			Quality:            85,
			Force:              force,
			GenerateThumbnails: true,
			ThumbnailWidth:     300,
		})

		// Walk through input directory
		err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && isImage(path) {
				fmt.Printf("Processing %s...\n", path)

				// Create Source and Destination
				src := &processing.FileSource{Path: path}

				// Determine project subdirectory for output
				rel, _ := filepath.Rel(inputDir, filepath.Dir(path))
				destDir := filepath.Join(outputDir, rel)
				dst := &processing.MultiFileDestination{Dir: destDir, Root: outputDir}

				if err := processor.ProcessImage(src, dst); err != nil {
					fmt.Printf("Error processing %s: %v\n", path, err)
				}
			}
			return nil
		})

		if err != nil {
			fmt.Printf("Error walking input directory: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Done.")
	},
}

func isImage(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png"
}

func init() {
	imagesCmd.AddCommand(processCmd)

	processCmd.Flags().StringVarP(&inputDir, "input", "i", "photos", "Input directory containing project subfolders")
	processCmd.Flags().StringVarP(&outputDir, "output", "o", "dist/images", "Output directory for processed images")
	processCmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files even if cached")
}
